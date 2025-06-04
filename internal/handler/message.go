package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"telegram-tax-bot/internal/manager"
	"telegram-tax-bot/internal/model"
	"telegram-tax-bot/internal/utils"
	"time"

	reportbuilder "telegram-tax-bot/internal/report_builder"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (r *Registry) handleMessage(msg *tgbotapi.Message) {
	userID := msg.From.ID
	s := manager.GetSession(userID)
	text := msg.Text

	switch s.PendingAction {
	case "awaiting_edit_index":
		handleAwaitingEditIndex(msg, s, r.bot)
		return
	case "awaiting_date":
		handleAwaitingDate(msg, s, r.bot)
		return
	case "awaiting_new_in":
		handleAwaitingNewIn(msg, s, r.bot)
		return
	case "awaiting_new_out":
		handleAwaitingNewOut(msg, s, r.bot)
		return
	case "awaiting_new_country":
		handleAwaitingNewCountry(msg, s, r.bot)
		return
	case "awaiting_add_out":
		handleAwaitingAddOut(msg, s, r.bot)
		return
	case "awaiting_add_country":
		handleAwaitingAddCountry(msg, s, r.bot)
		return
	case "awaiting_add_open_country":
		handleAddOpenCountry(msg, s, r.bot)
		return
	case "awaiting_tail_out":
		handleAwaitingTailOut(msg, s, r.bot)
		return
	case "awaiting_tail_country":
		handleAwaitingTailCountry(msg, s, r.bot)
		return
	case "awaiting_head_in":
		handleAwaitingHeadIn(msg, s, r.bot)
		return
	case "awaiting_head_country":
		handleAwaitingHeadCountry(msg, s, r.bot)
		return
	case "awaiting_add_in":
		handleAddin(msg, s, r.bot)
		return
	}

	// ✅ Загрузка JSON-файла
	if msg.Document != nil && s.Data.Current == "upload_pending" {
		handleInputFile(msg, s, r.bot)
		return
	}

	// ✅ Команды и кнопки
	switch {
	case strings.HasPrefix(text, "/start"), text == "🔙 Назад в меню":
		handleStartCommand(s, msg, r.bot)
	case strings.HasPrefix(text, "/help"), text == "ℹ️ Помощь":
		handleHelpCommand(msg, r.bot)
	case text == "📎 Загрузить файл", text == "📎 Загрузить новый файл":
		handleUploadCommand(s, msg, r.bot)
	case text == "🗑 Сбросить":
		handleResetCommand(s, msg, r.bot)
	case text == "📅 Отчёт на заданную дату":
		handleSetDateCommand(s, msg, r.bot)
	case text == "📋 Показать текущие данные":
		handlePeriodsCommand(s, msg, r.bot)
	case text == "📊 Отчёт":
		handleShowReport(s, msg, r.bot)
	default:
		if strings.HasPrefix(text, "{") {
			handleJSONInput(msg, s, r.bot)
		} else {
			r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❓ Неизвестная команда. Введите /help, чтобы посмотреть список."))
		}
	}
}

func handleAwaitingEditIndex(msg *tgbotapi.Message, s *model.Session, bot *tgbotapi.BotAPI) {
	index, err := strconv.Atoi(strings.TrimSpace(msg.Text))

	if err != nil || index < 1 || index > len(s.Data.Periods) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Введите корректный номер периода."))
		return
	}

	s.EditingIndex = index - 1
	s.PendingAction = "awaiting_edit_field"
	s.SaveSession()

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

func handleAwaitingDate(msg *tgbotapi.Message, s *model.Session, bot *tgbotapi.BotAPI) {
	date, err := utils.ParseDate(msg.Text)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Неверный формат даты. Используйте ДД.ММ.ГГГГ."))
		return
	}
	s.Data.Current = date.Format("02.01.2006")
	s.PendingAction = ""
	s.SaveSession()
	report := reportbuilder.BuildReport(s.Data)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ Дата расчета установлена: %s\n\n%s", s.Data.Current, report)))
}

func handleAwaitingNewIn(msg *tgbotapi.Message, s *model.Session, bot *tgbotapi.BotAPI) {
	newDate, err := utils.ParseDate(msg.Text)
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
	oldDate, _ := utils.ParseDate(curr.In)
	if newDate.Equal(oldDate) {
		s.PendingAction = ""
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "ℹ️ Дата въезда не изменилась."))
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, formatPeriodList(s.Data.Periods, s.Data.Current)))
		return
	}

	// Проверка конфликта с предыдущим периодом
	if index > 0 {
		prev := s.Data.Periods[index-1]
		prevOut, err := utils.ParseDate(prev.Out)
		if err == nil {
			switch {
			case newDate.Before(prevOut):
				// конфликт → предлагаем подвинуть out предыдущего периода
				s.TempEditedIn = newDate.Format("02.01.2006")
				s.PendingAction = "resolve_in_conflict"
				s.SaveSession()

				text := fmt.Sprintf("⚠️ Новая дата въезда пересекается с предыдущим периодом (%s). Что сделать?",
					utils.FormatDate(prevOut))
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
				s.SaveSession()

				text := fmt.Sprintf("⚠️ Между %s и %s обнаружен разрыв. Что сделать?",
					utils.FormatDate(prevOut.AddDate(0, 0, 1)), utils.FormatDate(newDate))
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
	s.SaveSession()
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Дата въезда обновлена."))
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, formatPeriodList(s.Data.Periods, s.Data.Current)))
}

func handleAwaitingNewOut(msg *tgbotapi.Message, s *model.Session, bot *tgbotapi.BotAPI) {
	newDate, err := utils.ParseDate(msg.Text)
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
	oldDate, _ := utils.ParseDate(curr.Out)
	if newDate.Equal(oldDate) {
		s.PendingAction = ""
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "ℹ️ Дата выезда не изменилась."))
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, formatPeriodList(s.Data.Periods, s.Data.Current)))
		return
	}

	// Проверка конфликта с следующим периодом
	if index < len(s.Data.Periods)-1 {
		next := s.Data.Periods[index+1]
		nextIn, err := utils.ParseDate(next.In)
		if err == nil {
			switch {
			case newDate.After(nextIn):
				s.TempEditedOut = newDate.Format("02.01.2006")
				s.PendingAction = "resolve_out_conflict"
				s.SaveSession()

				text := fmt.Sprintf("⚠️ Новая дата выезда пересекается со следующим периодом (%s). Что сделать?",
					utils.FormatDate(nextIn))
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
				s.SaveSession()

				text := fmt.Sprintf("⚠️ Между %s и %s образовался разрыв. Что сделать?",
					utils.FormatDate(newDate.AddDate(0, 0, 1)), utils.FormatDate(nextIn))
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
	s.SaveSession()
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Дата выезда обновлена."))
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, formatPeriodList(s.Data.Periods, s.Data.Current)))
}

func handleAwaitingNewCountry(msg *tgbotapi.Message, s *model.Session, bot *tgbotapi.BotAPI) {
	newCountry := strings.TrimSpace(msg.Text)
	if newCountry == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Название страны не может быть пустым."))
		return
	}
	s.Data.Periods[s.EditingIndex].Country = newCountry
	s.PendingAction = ""
	s.SaveSession()
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Страна обновлена."))
}

func handleAwaitingAddOut(msg *tgbotapi.Message, s *model.Session, bot *tgbotapi.BotAPI) {
	if len(s.Temp) == 0 {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Внутренняя ошибка. Начните добавление заново."))
		s.PendingAction = ""
		return
	}
	date, err := utils.ParseDate(msg.Text)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Неверный формат даты. Используйте ДД.ММ.ГГГГ."))
		return
	}
	inDate, err := utils.ParseDate(s.Temp[0].In)
	if err != nil || date.Before(inDate) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Дата выезда не может быть раньше даты въезда."))
		return
	}
	s.Temp[0].Out = date.Format("02.01.2006")
	s.PendingAction = "awaiting_add_country"
	s.SaveSession()
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🌍 Укажите название страны:"))
}

func handleAwaitingAddCountry(msg *tgbotapi.Message, s *model.Session, bot *tgbotapi.BotAPI) {
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
	newIn, errIn := utils.ParseDate(period.In)
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
		lastOutDate, err := utils.ParseDate(lastOut)
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
	s.SaveSession()

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Новый период добавлен."))
	handlePeriodsCommand(s, msg, bot)
}

func handleAddOpenCountry(msg *tgbotapi.Message, s *model.Session, bot *tgbotapi.BotAPI) {
	country := strings.TrimSpace(msg.Text)
	if country == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Название страны не может быть пустым"))
		return
	}
	s.Data.Periods = append(s.Data.Periods, model.Period{
		In:      s.Data.Current,
		Out:     "",
		Country: country,
	})
	s.PendingAction = ""
	s.SaveSession()

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Новый период добавлен."))
	handlePeriodsCommand(s, msg, bot)
}

func handleAwaitingTailOut(msg *tgbotapi.Message, s *model.Session, bot *tgbotapi.BotAPI) {
	date, err := utils.ParseDate(msg.Text)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Неверный формат даты. Используйте ДД.ММ.ГГГГ."))
		return
	}

	s.Temp = []model.Period{{Out: date.Format("02.01.2006")}}
	s.PendingAction = "awaiting_tail_country"
	s.SaveSession()

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🌍 Укажите название страны:"))
}

func handleAwaitingTailCountry(msg *tgbotapi.Message, s *model.Session, bot *tgbotapi.BotAPI) {
	country := strings.TrimSpace(msg.Text)
	if country == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Страна не может быть пустой."))
		return
	}
	if len(s.Temp) == 0 {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Внутренняя ошибка: начните добавление заново."))
		s.PendingAction = ""
		return
	}

	period := s.Temp[0]
	period.Country = country

	if len(s.Data.Periods) > 0 {
		first := s.Data.Periods[0]
		if first.In != "" {
			firstIn, err := utils.ParseDate(first.In)
			outDate, errOut := utils.ParseDate(period.Out)
			if err == nil && errOut == nil && outDate.After(firstIn) {
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Дата выезда не может быть после начала первого периода."))
				s.PendingAction = ""
				s.Temp = nil
				return
			}
		}
	}

	s.Data.Periods = append([]model.Period{period}, s.Data.Periods...)
	s.PendingAction = ""
	s.Temp = nil
	s.SaveSession()

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Новый период добавлен."))
	handlePeriodsCommand(s, msg, bot)
}

func handleAwaitingHeadIn(msg *tgbotapi.Message, s *model.Session, bot *tgbotapi.BotAPI) {
	date, err := utils.ParseDate(msg.Text)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Неверный формат даты. Используйте ДД.ММ.ГГГГ."))
		return
	}

	s.Temp = []model.Period{{In: date.Format("02.01.2006")}}
	s.PendingAction = "awaiting_head_country"
	s.SaveSession()

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🌍 Укажите название страны:"))
}

func handleAwaitingHeadCountry(msg *tgbotapi.Message, s *model.Session, bot *tgbotapi.BotAPI) {
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

	if len(s.Data.Periods) > 0 {
		last := s.Data.Periods[len(s.Data.Periods)-1]
		lastOut := last.Out
		if lastOut == "" {
			lastOut = s.Data.Current
		}
		newIn, err1 := utils.ParseDate(period.In)
		lastOutDate, err2 := utils.ParseDate(lastOut)
		if err1 == nil && err2 == nil && newIn.Before(lastOutDate) {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Невозможно добавить период: нарушен хронологический порядок."))
			s.PendingAction = ""
			s.Temp = nil
			return
		}
	}

	s.Data.Periods = append(s.Data.Periods, period)
	s.PendingAction = ""
	s.Temp = nil
	s.SaveSession()

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Новый период добавлен."))
	handlePeriodsCommand(s, msg, bot)
}

func handleAddin(msg *tgbotapi.Message, s *model.Session, bot *tgbotapi.BotAPI) {
	text := strings.TrimSpace(msg.Text)
	_, err := utils.ParseDate(text)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Неверный формат даты. Введите ДД.ММ.ГГГГ."))
		return
	}
	s.Temp = []model.Period{{In: text}} // сохраняем только дату in во временное хранилище
	s.PendingAction = "awaiting_add_out"
	s.SaveSession()
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📆 Введите дату выезда (ДД.ММ.ГГГГ):"))
}

func handleJSONInput(msg *tgbotapi.Message, s *model.Session, bot *tgbotapi.BotAPI) {
	s.BackupSession()
	err := json.Unmarshal([]byte(msg.Text), &s.Data)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Ошибка в формате JSON."))
		return
	}
	if s.Data.Current == "" {
		s.Data.Current = time.Now().Format("02.01.2006")
	}
	s.SaveSession()
	report := reportbuilder.BuildReport(s.Data)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, report))
}

func handleInputFile(msg *tgbotapi.Message, s *model.Session, bot *tgbotapi.BotAPI) {
	fileID := msg.Document.FileID
	file, _ := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	url := file.Link(bot.Token)
	resp, err := http.Get(url)

	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Не удалось загрузить файл."))
		return
	}

	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	msg.Text = string(body)
	s.Data.Current = "" // сбрасываем флаг после загрузки
	handleJSONInput(msg, s, bot)
}
