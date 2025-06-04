package handler

import (
	"fmt"
	"os"
	"strings"
	"telegram-tax-bot/internal/keyboard"
	"telegram-tax-bot/internal/manager"
	"telegram-tax-bot/internal/model"
	"telegram-tax-bot/internal/utils"

	reportbuilder "telegram-tax-bot/internal/report_builder"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (r *Registry) handleCallback(callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	session := manager.GetSession(userID)
	data := callback.Data
	chatID := callback.Message.Chat.ID
	message := callback.Message

	// Remove inline keyboard from the message that triggered the callback
	removeInlineKeyboard(r.bot, chatID, message.MessageID)

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
		handleShowReport(session, message, r.bot)

	case "add_period":
		handleAddPeriod(message, r.bot)

	case "add_tail":
		handleAddTail(session, message, r.bot)

	case "add_head":
		handleAddHead(session, message, r.bot)

	case "add_full":
		handleAddFull(session, message, r.bot)

	case "edit_period":
		handleEditPeriod(session, message, r.bot)

	case "adjust_prev_out":
		handleAdjustPrevOut(session, callback.Message, r.bot)
		handlePeriodsCommand(session, callback.Message, r.bot)

	case "edit_in":
		handleEdinIn(session, callback.Message, r.bot)

	case "edit_out":
		handleEditOut(session, callback.Message, r.bot)

	case "edit_country":
		handleEditCountry(session, callback.Message, r.bot)

	default:
		r.bot.Send(tgbotapi.NewMessage(chatID, "❓ Неизвестная кнопка."))
	}

	r.bot.Request(tgbotapi.NewCallback(callback.ID, ""))
}

func handleStartCommand(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, "🔘 Выберите действие:")
	reply.ReplyMarkup = keyboard.BuildMainMenu(s)
	bot.Send(reply)
}

func handleHelpCommand(msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	helpText := `ℹ️ Этот бот помогает определить налоговое резидентство на основе загруженных периодов пребывания в разных странах.

📎 С чего начать?
1. Сформируйте JSON-файл со списком ваших поездок (пример формата можно получить по кнопке «Загрузить файл»).
2. Отправьте файл через команду /upload_report или с помощью кнопки 📎.
3. Бот рассчитает, в какой стране вы провели больше всего времени за последний год.

📅 Как задать дату расчёта?
— Нажмите «📅 Задать дату» и укажите день, на который нужен расчёт (например: 15.04.2025).

📊 Что покажет отчёт?
— Страну, где вы провели больше всего дней.
— Если есть страна с 183+ днями — вы налоговый резидент этой страны.

🔁 Другие функции:
— /reset — сбросить все данные
— /periods — показать список загруженных периодов

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
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📅 Введите дату в формате ДД.ММ.ГГГГ:"))
}

func handleUploadCommand(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	s.Data.Current = "upload_pending"
	s.SaveSession()

	newMsg := tgbotapi.NewMessage(msg.Chat.ID, "📎 Пришлите JSON-файл документом.")
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

	bot.Send(tgbotapi.NewMessage(chatID, "➕ Добавлен период «unknown». Дата въезда обновлена."))
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

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📌 Следующий период сдвинут, дата выезда обновлена."))
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, formatPeriodList(s.Data.Periods, s.Data.Current)))
}

func handleKeepConflict(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	if s.PendingAction == "confirm_conflict_in" {
		s.Data.Periods[s.EditingIndex].In = s.TempEditedIn
		s.PendingAction = ""
		s.SaveSession()
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Дата въезда обновлена."))
	} else if s.PendingAction == "confirm_conflict_out" {
		s.Data.Periods[s.EditingIndex].Out = s.TempEditedOut
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

func handleShowReport(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	report := reportbuilder.BuildReport(s.Data)
	reply := tgbotapi.NewMessage(msg.Chat.ID, report)
	reply.ReplyMarkup = keyboard.BuildBackToMenu()
	bot.Send(reply)
}

func handleAddPeriod(msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	// меню выбора варианта добавления
	reply := tgbotapi.NewMessage(msg.Chat.ID, "➕ Что добавить?")
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
	bot.Send(reply)
}

func handleAddTail(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	s.PendingAction = "awaiting_tail_out"
	s.SaveSession()

	replay := tgbotapi.NewMessage(msg.Chat.ID, "📆 Введите дату выезда (ДД.ММ.ГГГГ):")
	replay.ReplyMarkup = keyboard.BuildBackToMenu()
	bot.Send(replay)
}

func handleAddHead(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	s.PendingAction = "awaiting_head_in"
	s.SaveSession()

	replay := tgbotapi.NewMessage(msg.Chat.ID, "📆 Введите дату въезда (ДД.ММ.ГГГГ):")
	replay.ReplyMarkup = keyboard.BuildBackToMenu()
	bot.Send(replay)
}

func handleAddFull(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	s.PendingAction = "awaiting_add_in"
	s.SaveSession()

	replay := tgbotapi.NewMessage(msg.Chat.ID, "📆 Введите дату въезда (ДД.ММ.ГГГГ):")
	replay.ReplyMarkup = keyboard.BuildBackToMenu()
	bot.Send(replay)
}

func handleEditPeriod(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	if s.IsEmpty() {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📭 Нет сохранённых периодов для редактирования."))
	} else {
		s.PendingAction = "awaiting_edit_index"
		s.SaveSession()
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✏️ Введите номер периода для редактирования:"))
	}
}

func handleAdjustPrevOut(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	newIn, _ := utils.ParseDate(s.TempEditedIn)

	s.Data.Periods[s.EditingIndex-1].Out = newIn.Format("02.01.2006")
	s.Data.Periods[s.EditingIndex].In = newIn.Format("02.01.2006")
	s.PendingAction = ""
	s.TempEditedIn = ""
	s.SaveSession()

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📌 Предыдущий период подвинут. Дата въезда обновлена."))
}

func handleEdinIn(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	s.PendingAction = "awaiting_new_in"
	s.SaveSession()
	curr := s.Data.Periods[s.EditingIndex].In
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✏️ Текущая дата въезда: %s. Введите новую:", curr)))
}

func handleEditOut(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	s.PendingAction = "awaiting_new_out"
	s.SaveSession()
	curr := s.Data.Periods[s.EditingIndex].Out
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✏️ Текущая дата выезда: %s. Введите новую:", curr)))
}

func handleEditCountry(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	s.PendingAction = "awaiting_new_country"
	s.SaveSession()
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🌍 Введите новое название страны:"))
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
		flag := ""
		if p.Country == "unknown" {
			flag = "🕳 "
		} else if code, ok := utils.CountryCodeMap[p.Country]; ok {
			flag = utils.CountryToFlag(code) + " "
		}
		builder.WriteString(fmt.Sprintf("%d. %s%s (%s — %s)\n", i+1, flag, p.Country, in, out))
	}
	return builder.String()
}

// removeInlineKeyboard clears the inline keyboard from a message without deleting the message itself.
func removeInlineKeyboard(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	// Telegram may return an error if the original message is too old or was
	// already edited. We try to clear the markup and silently ignore any
	// failure. Previously the message was deleted on failure, but that lead
	// to losing the user's history. Now we simply ignore the error.
	empty := tgbotapi.NewInlineKeyboardMarkup()
	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, empty)
	_, _ = bot.Request(edit)
}
