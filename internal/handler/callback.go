package handler

import (
	"fmt"
	"os"
	"strings"
	"telegram-tax-bot/internal/keyboard"
	"telegram-tax-bot/internal/model"
	"telegram-tax-bot/internal/service"
	"telegram-tax-bot/internal/utils"

	reportbuilder "telegram-tax-bot/internal/report_builder"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (r *Registry) handleCallback(callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	session := service.GetSession(userID)
	data := callback.Data
	chatID := callback.Message.Chat.ID
	message := callback.Message

	switch data {
	case "start":
		handleStartCommand(session, message, r.bot)
	case "help":
		handleHelpCommand(message, r.bot)
	case "reset":
		handleResetCommand(session, message, r.bot)
	case "set_date":
		handleSetDateCommand(session, message, r.bot)
	case "upload_report", "upload_file":
		handleUploadCommand(session, message, r.bot)
	case "periods":
		handlePeriodsCommand(session, message, r.bot)
	case "add_gap_period":
		handleAddGapPeriod(session, callback, r.bot)
	case "adjust_next_in":
		handleAdjustNextIn(session, message, r.bot)
	case "keep_conflict":
		handleKeepConflict(session, message, r.bot)
	case "cancel_edit":
		handleCancelEdit(session, message, r.bot)
	case "show_report":
		report := reportbuilder.BuildReport(session.Data)
		msg := tgbotapi.NewMessage(chatID, report)
		msg.ReplyMarkup = keyboard.BuildBackToMenu()
		r.bot.Send(msg)
	case "add_period":
		// меню выбора варианта добавления
		reply := tgbotapi.NewMessage(chatID, "➕ Что добавить?")
		reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🗓 Хвостовой (только выезд)", "add_tail"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⏮ Начальный (только въезд)", "add_head"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📄 Полный (въезд+выезд)", "add_full"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ Отменить", "start"),
			),
		)
		r.bot.Send(reply)

	case "add_tail":
		session.PendingAction = "awaiting_tail_out"
		session.SaveSession()

		replay := tgbotapi.NewMessage(chatID, "📆 Введите дату выезда (ДД.MM.YYYY):")
		replay.ReplyMarkup = keyboard.BuildBackToMenu()
		r.bot.Send(replay)

	case "add_head":
		session.PendingAction = "awaiting_head_in"
		session.SaveSession()

		replay := tgbotapi.NewMessage(chatID, "📆 Введите дату въезда (ДД.MM.YYYY):")
		replay.ReplyMarkup = keyboard.BuildBackToMenu()
		r.bot.Send(replay)

	case "add_full":
		session.PendingAction = "awaiting_full_in"
		session.SaveSession()

		replay := tgbotapi.NewMessage(chatID, "📆 Введите дату въезда (ДД.MM.YYYY):")
		replay.ReplyMarkup = keyboard.BuildBackToMenu()
		r.bot.Send(replay)
		if session.IsEmpty() {
			r.bot.Send(tgbotapi.NewMessage(chatID, "📭 Нет сохранённых периодов для удаления."))
		} else {
			session.PendingAction = "awaiting_delete_index"
			session.SaveSession()
			r.bot.Send(tgbotapi.NewMessage(chatID, "❌ Укажите номер периода, который нужно удалить:"))
		}
	case "edit_period":
		if session.IsEmpty() {
			r.bot.Send(tgbotapi.NewMessage(chatID, "📭 Нет сохранённых периодов для редактирования."))
		} else {
			session.PendingAction = "awaiting_edit_index"
			session.SaveSession()
			r.bot.Send(tgbotapi.NewMessage(chatID, "✏️ Введите номер периода, который хотите отредактировать:"))
		}
	case "adjust_prev_out":
		newIn, _ := utils.ParseDate(session.TempEditedIn)
		session.Data.Periods[session.EditingIndex-1].Out = newIn.Format("02.01.2006")
		session.Data.Periods[session.EditingIndex].In = newIn.Format("02.01.2006")
		session.PendingAction = ""
		session.TempEditedIn = ""
		session.SaveSession()
		r.bot.Send(tgbotapi.NewMessage(chatID, "📌 Предыдущий период подвинут. Дата въезда обновлена."))
		handlePeriodsCommand(session, callback.Message, r.bot)
	case "edit_in":
		session.PendingAction = "awaiting_new_in"
		session.SaveSession()
		curr := session.Data.Periods[session.EditingIndex].In
		r.bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✏️ Текущая дата %s. Введите новую дату въезда:", curr)))
	case "edit_out":
		session.PendingAction = "awaiting_new_out"
		session.SaveSession()
		curr := session.Data.Periods[session.EditingIndex].Out
		r.bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✏️ Текущая дата %s. Введите новую дату выезда:", curr)))
	case "edit_country":
		session.PendingAction = "awaiting_new_country"
		session.SaveSession()
		r.bot.Send(tgbotapi.NewMessage(chatID, "🌍 Введите новое название страны:"))
	default:
		r.bot.Send(tgbotapi.NewMessage(chatID, "❓ Неизвестная кнопка."))
	}

	r.bot.Request(tgbotapi.NewCallback(callback.ID, ""))
	// continue
}

func handleStartCommand(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, "🔘 Выберите действие:")
	reply.ReplyMarkup = keyboard.BuildMainMenu(s)
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
	newMsg.ReplyMarkup = keyboard.BuildBackToMenu()
	bot.Send(newMsg)
}

func handleResetCommand(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	if s.IsEmpty() {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📭 У вас пока нет сохранённых периодов."))
		return
	}
	s.Data = model.Data{}
	s.Backup = model.Data{}
	s.Temp = nil
	_ = os.Remove(fmt.Sprintf("%s/data.json", s.HistoryDir))
	s.SaveSession()

	newMsg := tgbotapi.NewMessage(msg.Chat.ID, "✅ Данные сброшены.")
	newMsg.ReplyMarkup = keyboard.BuildBackToMenu()
	bot.Send(newMsg)
}

func handleSetDateCommand(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	if s.IsEmpty() {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📭 У вас пока нет сохранённых периодов."))
		return
	}
	s.PendingAction = "awaiting_date"
	s.SaveSession()
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📅 Введите дату в формате ДД.ММ.ГГГГ"))
}

func handleUploadCommand(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	s.Data.Current = "upload_pending"
	s.SaveSession()

	newMsg := tgbotapi.NewMessage(msg.Chat.ID, "📎 Пожалуйста, отправьте JSON-файл как документ.")
	newMsg.ReplyMarkup = keyboard.BuildBackToMenu()
	bot.Send(newMsg)
}

func handlePeriodsCommand(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	if s.IsEmpty() {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📭 У вас пока нет сохранённых периодов."))
		return
	}
	msgText := s.BuildPeriodsList()
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

func handleAddGapPeriod(s *model.Session, callback *tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) {
	chatID := callback.Message.Chat.ID

	newIn, _ := utils.ParseDate(s.TempEditedIn)
	prev := s.Data.Periods[s.EditingIndex-1]

	prevOut, _ := utils.ParseDate(prev.Out)
	newGapStart := prevOut.AddDate(0, 0, 1)
	newGapEnd := newIn.AddDate(0, 0, -1)

	newGap := model.Period{
		In:      newGapStart.Format("02.01.2006"),
		Out:     newGapEnd.Format("02.01.2006"),
		Country: "unknown",
	}

	// Вставить "unknown" перед текущим
	s.Data.Periods = append(
		s.Data.Periods[:s.EditingIndex],
		append([]model.Period{newGap}, s.Data.Periods[s.EditingIndex:]...)...,
	)

	// Обновляем in текущего периода
	s.Data.Periods[s.EditingIndex+1].In = newIn.Format("02.01.2006")
	s.EditingIndex++ // корректируем индекс
	s.PendingAction = ""
	s.TempEditedIn = ""
	s.SaveSession()

	bot.Send(tgbotapi.NewMessage(chatID, "➕ Добавлен период 'unknown'. Дата въезда обновлена."))
	handlePeriodsCommand(s, callback.Message, bot)
}

func handleAdjustNextIn(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	index := s.EditingIndex
	if index+1 >= len(s.Data.Periods) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Ошибка: следующего периода не существует."))
		return
	}

	newOut, err := utils.ParseDate(s.TempEditedOut)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Ошибка при обработке даты."))
		return
	}

	// ✅ Обновляем out у текущего периода и in у следующего
	s.Data.Periods[index].Out = s.TempEditedOut
	s.Data.Periods[index+1].In = newOut.AddDate(0, 0, 1).Format("02.01.2006")

	s.PendingAction = ""
	s.TempEditedOut = ""
	s.SaveSession()

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📌 Следующий период сдвинут. Дата выезда обновлена."))
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, formatPeriodList(s.Data.Periods, s.Data.Current)))
}

func handleKeepConflict(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	if s.PendingAction == "confirm_conflict_in" {
		s.Data.Periods[s.EditingIndex].In = s.TempEditedIn
		s.PendingAction = ""
		s.SaveSession()
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Дата въезда обновлена."))
	} else if s.PendingAction == "confirm_conflict_out" {
		s.Data.Periods[s.EditingIndex].Out = s.TempEditedIn
		s.PendingAction = ""
		s.SaveSession()
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Дата выезда обновлена."))
	} else {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Нет ожидаемого конфликта."))
		return
	}

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, formatPeriodList(s.Data.Periods, s.Data.Current)))
}

func handleCancelEdit(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	s.PendingAction = ""
	s.TempEditedOut = ""
	s.SaveSession()

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Изменение отменено."))
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, formatPeriodList(s.Data.Periods, s.Data.Current)))
}

func formatPeriodList(periods []model.Period, current string) string {
	var builder strings.Builder
	builder.WriteString("📋 Список периодов:\n\n")
	for i, p := range periods {
		in := p.In
		if in == "" {
			in = "—"
		}
		out := p.Out
		if out == "" {
			out = "по " + current
		}
		builder.WriteString(fmt.Sprintf("%d. %s — %s (%s)\n", i+1, in, out, p.Country))
	}
	return builder.String()
}
