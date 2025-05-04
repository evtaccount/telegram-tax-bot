package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func handleJSONInput(msg *tgbotapi.Message, s *Session, bot *tgbotapi.BotAPI) {
	backupSession(s)
	err := json.Unmarshal([]byte(msg.Text), &s.Data)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Ошибка в формате JSON"))
		return
	}
	if s.Data.Current == "" {
		s.Data.Current = time.Now().Format("02.01.2006")
	}
	saveSession(s)
	report := buildReport(s.Data)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, report))
}

func handleAwaitingAddOut(msg *tgbotapi.Message, s *Session, bot *tgbotapi.BotAPI) {
	if len(s.Temp) == 0 {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Внутренняя ошибка. Начните добавление заново."))
		s.PendingAction = ""
		return
	}
	date, err := parseDate(msg.Text)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Неверный формат даты. Используйте ДД.ММ.ГГГГ."))
		return
	}
	inDate, err := parseDate(s.Temp[0].In)
	if err != nil || date.Before(inDate) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Дата выезда не может быть раньше даты въезда."))
		return
	}
	s.Temp[0].Out = date.Format("02.01.2006")
	s.PendingAction = "awaiting_add_country"
	saveSession(s)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🌍 Укажите название страны:"))
}

func handleAwaitingAddCountry(msg *tgbotapi.Message, s *Session, bot *tgbotapi.BotAPI) {
	country := strings.TrimSpace(msg.Text)
	if country == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Страна не может быть пустой."))
		return
	}
	if len(s.Temp) == 0 {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Внутренняя ошибка: временный буфер пуст."))
		s.PendingAction = ""
		return
	}
	period := s.Temp[0]
	period.Country = country

	// Проверка хронологического порядка
	newIn, errIn := parseDate(period.In)
	if errIn != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Некорректная дата въезда."))
		return
	}

	if len(s.Data.Periods) > 0 {
		last := s.Data.Periods[len(s.Data.Periods)-1]
		lastOut := last.Out
		if lastOut == "" {
			lastOut = s.Data.Current
		}
		lastOutDate, err := parseDate(lastOut)
		if err == nil && newIn.Before(lastOutDate) {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Невозможно добавить период: нарушен хронологический порядок."))
			s.PendingAction = ""
			s.Temp = nil
			return
		}
	}

	s.Data.Periods = append(s.Data.Periods, period)
	s.Temp = nil
	s.PendingAction = ""
	saveSession(s)

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Новый период добавлен."))
	handlePeriodsCommand(s, msg, bot)
}

func handleAddGapPeriod(s *Session, callback *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) {
	chatID := callback.Message.Chat.ID

	newIn, _ := parseDate(s.TempEditedIn)
	prev := s.Data.Periods[s.EditingIndex-1]

	prevOut, _ := parseDate(prev.Out)
	newGapStart := prevOut.AddDate(0, 0, 1)
	newGapEnd := newIn.AddDate(0, 0, -1)

	newGap := Period{
		In:      newGapStart.Format("02.01.2006"),
		Out:     newGapEnd.Format("02.01.2006"),
		Country: "unknown",
	}

	// Вставить "unknown" перед текущим
	s.Data.Periods = append(
		s.Data.Periods[:s.EditingIndex],
		append([]Period{newGap}, s.Data.Periods[s.EditingIndex:]...)...,
	)

	// Обновляем in текущего периода
	s.Data.Periods[s.EditingIndex+1].In = newIn.Format("02.01.2006")
	s.EditingIndex++ // корректируем индекс
	s.PendingAction = ""
	s.TempEditedIn = ""
	saveSession(s)

	bot.Send(tgbotapi.NewMessage(chatID, "➕ Добавлен период 'unknown'. Дата въезда обновлена."))
	handlePeriodsCommand(s, callback.Message, bot)
}

func handleAwaitingNewIn(msg *tgbotapi.Message, s *Session, bot *tgbotapi.BotAPI) {
	newDate, err := parseDate(msg.Text)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Неверный формат даты."))
		return
	}

	index := s.EditingIndex
	if index < 0 || index >= len(s.Data.Periods) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Ошибка: индекс периода вне допустимого диапазона."))
		return
	}

	curr := s.Data.Periods[index]
	oldDate, _ := parseDate(curr.In)
	if newDate.Equal(oldDate) {
		s.PendingAction = ""
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "ℹ️ Дата въезда не изменилась."))
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, formatPeriodList(s.Data.Periods, s.Data.Current)))
		return
	}

	// Проверка конфликта с предыдущим периодом
	if index > 0 {
		prev := s.Data.Periods[index-1]
		prevOut, err := parseDate(prev.Out)
		if err == nil {
			switch {
			case newDate.Before(prevOut):
				// конфликт → предлагаем подвинуть out предыдущего периода
				s.TempEditedIn = newDate.Format("02.01.2006")
				s.PendingAction = "resolve_in_conflict"
				saveSession(s)

				text := fmt.Sprintf("⚠️ Новая дата въезда пересекается с предыдущим периодом (%s). Что сделать?",
					formatDate(prevOut))
				markup := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📌 Подвинуть предыдущий период", "adjust_prev_out")),
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✅ Оставить как есть", "keep_conflict")),
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("❌ Отменить", "cancel_edit")),
				)
				msg := tgbotapi.NewMessage(msg.Chat.ID, text)
				msg.ReplyMarkup = markup
				bot.Send(msg)
				return

			case newDate.After(prevOut.AddDate(0, 0, 1)):
				// зазор → предлагаем действия
				s.TempEditedIn = newDate.Format("02.01.2006")
				s.PendingAction = "resolve_in_gap"
				saveSession(s)

				text := fmt.Sprintf("⚠️ Между %s и %s обнаружен разрыв. Что сделать?",
					formatDate(prevOut.AddDate(0, 0, 1)), formatDate(newDate))
				markup := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📌 Подвинуть предыдущий период", "adjust_prev_out")),
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✅ Оставить как есть", "keep_conflict")),
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("❌ Отменить", "cancel_edit")),
				)
				msg := tgbotapi.NewMessage(msg.Chat.ID, text)
				msg.ReplyMarkup = markup
				bot.Send(msg)
				return
			}
		}
	}

	// Всё в порядке, обновляем
	s.Data.Periods[index].In = newDate.Format("02.01.2006")
	s.PendingAction = ""
	saveSession(s)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Дата въезда обновлена."))
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, formatPeriodList(s.Data.Periods, s.Data.Current)))
}

func handleAwaitingNewOut(msg *tgbotapi.Message, s *Session, bot *tgbotapi.BotAPI) {
	newDate, err := parseDate(msg.Text)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Неверный формат даты."))
		return
	}

	index := s.EditingIndex
	if index < 0 || index >= len(s.Data.Periods) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Ошибка: индекс периода вне допустимого диапазона."))
		return
	}

	curr := s.Data.Periods[index]
	oldDate, _ := parseDate(curr.Out)
	if newDate.Equal(oldDate) {
		s.PendingAction = ""
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "ℹ️ Дата выезда не изменилась."))
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, formatPeriodList(s.Data.Periods, s.Data.Current)))
		return
	}

	// Проверка конфликта с следующим периодом
	if index < len(s.Data.Periods)-1 {
		next := s.Data.Periods[index+1]
		nextIn, err := parseDate(next.In)
		if err == nil {
			switch {
			case newDate.After(nextIn):
				s.TempEditedOut = newDate.Format("02.01.2006")
				s.PendingAction = "resolve_out_conflict"
				saveSession(s)

				text := fmt.Sprintf("⚠️ Новая дата выезда пересекается со следующим периодом (%s). Что сделать?",
					formatDate(nextIn))
				markup := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📌 Подвинуть следующий период", "adjust_next_in")),
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✅ Оставить как есть", "keep_conflict")),
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("❌ Отменить", "cancel_edit")),
				)
				msg := tgbotapi.NewMessage(msg.Chat.ID, text)
				msg.ReplyMarkup = markup
				bot.Send(msg)
				return

			case newDate.Before(nextIn.AddDate(0, 0, -1)):
				s.TempEditedOut = newDate.Format("02.01.2006")
				s.PendingAction = "resolve_out_gap"
				saveSession(s)

				text := fmt.Sprintf("⚠️ Между %s и %s образовался зазор. Что сделать?",
					formatDate(newDate.AddDate(0, 0, 1)), formatDate(nextIn))
				markup := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📌 Подвинуть следующий период", "adjust_next_in")),
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✅ Оставить как есть", "keep_conflict")),
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("❌ Отменить", "cancel_edit")),
				)
				msg := tgbotapi.NewMessage(msg.Chat.ID, text)
				msg.ReplyMarkup = markup
				bot.Send(msg)
				return
			}
		}
	}

	// Всё в порядке, обновляем
	s.Data.Periods[index].Out = newDate.Format("02.01.2006")
	s.PendingAction = ""
	saveSession(s)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Дата выезда обновлена."))
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, formatPeriodList(s.Data.Periods, s.Data.Current)))
}

func handleAwaitingNewCountry(msg *tgbotapi.Message, s *Session, bot *tgbotapi.BotAPI) {
	newCountry := strings.TrimSpace(msg.Text)
	if newCountry == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Название страны не может быть пустым."))
		return
	}
	s.Data.Periods[s.EditingIndex].Country = newCountry
	s.PendingAction = ""
	saveSession(s)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Страна обновлена."))
}

func handleAwaitingDate(msg *tgbotapi.Message, s *Session, bot *tgbotapi.BotAPI) {
	date, err := parseDate(msg.Text)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Неверный формат даты. Используйте ДД.ММ.ГГГГ"))
		return
	}
	s.Data.Current = date.Format("02.01.2006")
	s.PendingAction = ""
	saveSession(s)
	report := buildReport(s.Data)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ Дата расчета установлена: %s\n\n%s", s.Data.Current, report)))
}

func handleAwaitingEditIndex(msg *tgbotapi.Message, s *Session, bot *tgbotapi.BotAPI) {
	index, err := strconv.Atoi(strings.TrimSpace(msg.Text))

	if err != nil || index < 1 || index > len(s.Data.Periods) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Введите корректный номер периода."))
		return
	}

	s.EditingIndex = index - 1
	s.PendingAction = "awaiting_edit_field"
	saveSession(s)

	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Изменить дату въезда (in)", "edit_in"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📆 Изменить дату выезда (out)", "edit_out"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌍 Изменить страну", "edit_country"),
		),
	)
	from := s.Data.Periods[s.EditingIndex].In
	till := s.Data.Periods[s.EditingIndex].Out
	msgText := fmt.Sprintf("Выбран период с %s по %s. Что изменить?", from, till)
	msgToSend := tgbotapi.NewMessage(msg.Chat.ID, msgText)
	msgToSend.ReplyMarkup = buttons
	bot.Send(msgToSend)
}

func handleKeepConflict(msg *tgbotapi.Message, s *Session, bot *tgbotapi.BotAPI) {
	if s.PendingAction == "confirm_conflict_in" {
		s.Data.Periods[s.EditingIndex].In = s.TempEditedIn
		s.PendingAction = ""
		saveSession(s)
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Дата въезда обновлена."))
	} else if s.PendingAction == "confirm_conflict_out" {
		s.Data.Periods[s.EditingIndex].Out = s.TempEditedIn
		s.PendingAction = ""
		saveSession(s)
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Дата выезда обновлена."))
	} else {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Нет ожидаемого конфликта."))
		return
	}

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, formatPeriodList(s.Data.Periods, s.Data.Current)))
}

func handleAdjustNextIn(s *Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	index := s.EditingIndex
	if index+1 >= len(s.Data.Periods) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Ошибка: следующего периода не существует."))
		return
	}

	newOut, err := parseDate(s.TempEditedOut)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Ошибка при обработке даты."))
		return
	}

	// ✅ Обновляем out у текущего периода и in у следующего
	s.Data.Periods[index].Out = s.TempEditedOut
	s.Data.Periods[index+1].In = newOut.AddDate(0, 0, 1).Format("02.01.2006")

	s.PendingAction = ""
	s.TempEditedOut = ""
	saveSession(s)

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📌 Следующий период сдвинут. Дата выезда обновлена."))
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, formatPeriodList(s.Data.Periods, s.Data.Current)))
}

func handleCancelEdit(s *Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	s.PendingAction = ""
	s.TempEditedOut = ""
	saveSession(s)

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Изменение отменено."))
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, formatPeriodList(s.Data.Periods, s.Data.Current)))
}

func handleAddOpenCountry(msg *tgbotapi.Message, s *Session, bot *tgbotapi.BotAPI) {
	country := strings.TrimSpace(msg.Text)
	if country == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Название страны не может быть пустым"))
		return
	}
	s.Data.Periods = append(s.Data.Periods, Period{
		In:      s.Data.Current,
		Out:     "",
		Country: country,
	})
	s.PendingAction = ""
	saveSession(s)

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Новый период добавлен."))
	handlePeriodsCommand(s, msg, bot)
}

func handleStartCommand(s *Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, "🔘 Выберите действие:")
	reply.ReplyMarkup = buildMainMenu(s)
	bot.Send(reply)
}

func handleHelpCommand(msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	helpText := `ℹ️ Этот бот помогает определить налоговое резидентство на основе загруженных периодов пребывания в разных странах.

📎 С чего начать?
1. Сформируйте JSON-файл со списком ваших поездок (формат пример — по кнопке "Загрузить файл").
2. Отправьте файл через команду /upload_report или с помощью кнопки 📎.
3. Бот рассчитает, в какой стране вы провели больше всего времени за последний год.

📅 Как задать дату расчета?
— Выберите "📅 Задать дату" и укажите дату, на которую хотите сделать расчет (например: 15.04.2025).

📊 Что покажет отчет?
— Страну, в которой вы провели больше всего дней.
— Если есть страна с 183+ днями — вы налоговый резидент этой страны.

🔁 Другие функции:
— /reset: сбросить все данные
— /periods: показать список загруженных периодов

💬 Используйте /start для возврата в главное меню.`

	newMsg := tgbotapi.NewMessage(msg.Chat.ID, helpText)
	newMsg.ReplyMarkup = buildBackToMenu()
	bot.Send(newMsg)
}

func handleResetCommand(s *Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	if isEmpty(s) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📭 У вас пока нет сохранённых периодов."))
		return
	}
	s.Data = Data{}
	s.Backup = Data{}
	s.Temp = nil
	_ = os.Remove(fmt.Sprintf("%s/data.json", s.HistoryDir))
	saveSession(s)

	newMsg := tgbotapi.NewMessage(msg.Chat.ID, "✅ Данные сброшены.")
	newMsg.ReplyMarkup = buildBackToMenu()
	bot.Send(newMsg)
}

func handleSetDateCommand(s *Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	if isEmpty(s) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📭 У вас пока нет сохранённых периодов."))
		return
	}
	s.PendingAction = "awaiting_date"
	saveSession(s)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📅 Введите дату в формате ДД.ММ.ГГГГ"))
}

func handleUploadCommand(s *Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	s.Data.Current = "upload_pending"
	saveSession(s)

	newMsg := tgbotapi.NewMessage(msg.Chat.ID, "📎 Пожалуйста, отправьте JSON-файл как документ.")
	newMsg.ReplyMarkup = buildBackToMenu()
	bot.Send(newMsg)
}

func handlePeriodsCommand(s *Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	if isEmpty(s) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📭 У вас пока нет сохранённых периодов."))
		return
	}
	msgText := buildPeriodsList(s)
	newMsg := tgbotapi.NewMessage(msg.Chat.ID, msgText)
	newMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✏️ Отредактировать период", "edit_period")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("➕ Добавить период", "add_period")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить период", "delete_period")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📊 Отчёт", "show_report")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Назад в меню", "start")),
	)
	bot.Send(newMsg)
}

func handleInputFile(msg *tgbotapi.Message, s *Session, bot *tgbotapi.BotAPI) {
	fileID := msg.Document.FileID
	file, _ := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	url := file.Link(bot.Token)
	resp, err := http.Get(url)

	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Не удалось загрузить файл"))
		return
	}

	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	msg.Text = string(body)
	s.Data.Current = "" // сбрасываем флаг после загрузки
	handleJSONInput(msg, s, bot)
}
