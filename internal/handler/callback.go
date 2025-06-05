package handler

import (
	"fmt"
	"os"
	"strconv"
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

func handleCommandsCommand(msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	txt := `/start - главное меню
/help - справка
/upload_report - загрузить данные
/periods - показать периоды
/reset - сбросить данные`
	newMsg := tgbotapi.NewMessage(msg.Chat.ID, txt)
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
	newMsg.ReplyMarkup = keyboard.BuildPeriodsMenu()
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
	handlePeriodsCommand(s, msg, bot)
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

	handlePeriodsCommand(s, msg, bot)
}

func handleCancelEdit(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	s.PendingAction = ""
	s.TempEditedOut = ""
	s.SaveSession()

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Изменение отменено."))
	handlePeriodsCommand(s, msg, bot)
}

// handleBack cancels the current step and shows the appropriate menu.
func handleBack(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	switch s.PendingAction {
	case "awaiting_edit_index":
		s.PendingAction = ""
		s.SaveSession()
		handlePeriodsCommand(s, msg, bot)
	case "awaiting_delete_index":
		s.PendingAction = ""
		s.SaveSession()
		handlePeriodsCommand(s, msg, bot)
	case "awaiting_edit_field":
		handleEditPeriod(s, msg, bot)
	case "awaiting_new_in", "awaiting_new_out", "awaiting_new_country":
		s.PendingAction = "awaiting_edit_field"
		s.SaveSession()
		buttons := keyboard.BuildEditFieldMenu()
		from := s.Data.Periods[s.EditingIndex].In
		till := s.Data.Periods[s.EditingIndex].Out
		txt := fmt.Sprintf("Выбран период с %s по %s. Что изменить?", from, till)
		reply := tgbotapi.NewMessage(msg.Chat.ID, txt)
		reply.ReplyMarkup = buttons
		bot.Send(reply)
	default:
		handleStartCommand(s, msg, bot)
	}
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
	reply.ReplyMarkup = keyboard.BuildAddPeriodMenu()
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
		return
	}

	s.PendingAction = "awaiting_edit_index"
	s.SaveSession()

	text := s.BuildPeriodsList() + "\n✏️ Введите номер периода для редактирования:"
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ReplyMarkup = keyboard.BuildBack()
	bot.Send(reply)
}

func handleAdjustPrevOut(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	newIn, _ := utils.ParseDate(s.TempEditedIn)

	s.Data.Periods[s.EditingIndex-1].Out = newIn.Format("02.01.2006")
	s.Data.Periods[s.EditingIndex].In = newIn.Format("02.01.2006")
	s.PendingAction = ""
	s.TempEditedIn = ""
	s.SaveSession()

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📌 Предыдущий период подвинут. Дата въезда обновлена."))
	handlePeriodsCommand(s, msg, bot)
}

func handleEdinIn(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	s.PendingAction = "awaiting_new_in"
	s.SaveSession()
	curr := s.Data.Periods[s.EditingIndex].In
	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✏️ Текущая дата въезда: %s. Введите новую:", curr))
	reply.ReplyMarkup = keyboard.BuildBack()
	bot.Send(reply)
}

func handleEditOut(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	s.PendingAction = "awaiting_new_out"
	s.SaveSession()
	curr := s.Data.Periods[s.EditingIndex].Out
	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✏️ Текущая дата выезда: %s. Введите новую:", curr))
	reply.ReplyMarkup = keyboard.BuildBack()
	bot.Send(reply)
}

func handleEditCountry(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	s.PendingAction = "awaiting_new_country"
	s.SaveSession()
	reply := tgbotapi.NewMessage(msg.Chat.ID, "🌍 Введите новое название страны:")
	reply.ReplyMarkup = keyboard.BuildBack()
	bot.Send(reply)
}

func handleDeletePeriod(s *model.Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	if s.IsEmpty() {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📭 Нет сохранённых периодов для удаления."))
		return
	}

	s.PendingAction = "awaiting_delete_index"
	s.SaveSession()

	text := s.BuildPeriodsList() + "\n🗑 Введите номер периода для удаления:"
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ReplyMarkup = keyboard.BuildBack()
	bot.Send(reply)
}

func handleAwaitingDeleteIndex(msg *tgbotapi.Message, s *model.Session, bot *tgbotapi.BotAPI) {
	index, err := strconv.Atoi(strings.TrimSpace(msg.Text))
	if err != nil || index < 1 || index > len(s.Data.Periods) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Введите корректный номер периода."))
		return
	}

	idx := index - 1
	s.Data.Periods = append(s.Data.Periods[:idx], s.Data.Periods[idx+1:]...)
	s.PendingAction = ""
	s.SaveSession()

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🗑 Период удалён."))
	if s.IsEmpty() {
		handleStartCommand(s, msg, bot)
	} else {
		handlePeriodsCommand(s, msg, bot)
	}
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
	// No inline keyboards are used anymore, so nothing to remove.
}
