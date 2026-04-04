package handler

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"telegram-tax-bot/internal/keyboard"
	"telegram-tax-bot/internal/manager"
	"telegram-tax-bot/internal/model"
	"telegram-tax-bot/internal/utils"

	reportbuilder "telegram-tax-bot/internal/report_builder"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (r *Registry) handleCallback(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	userID := callback.From.ID
	session := manager.GetSession(userID)
	data := callback.Data

	msg := callback.Message.Message
	if msg == nil {
		return
	}
	chatID := msg.Chat.ID

	// Remove inline keyboard from the message that triggered the callback
	removeInlineKeyboard(ctx, b, chatID, msg.ID)

	switch data {
	case "start":
		handleStartCommand(ctx, session, msg, b)
	case "help":
		handleHelpCommand(ctx, msg, b)
	case "reset":
		handleResetCommand(ctx, session, msg, b)
	case "set_date":
		handleSetDateCommand(ctx, session, msg, b)
	case "upload_report", "upload_file":
		handleUploadCommand(ctx, session, msg, b)
	case "periods":
		handlePeriodsCommand(ctx, session, msg, b)
	case "add_gap_period":
		handleAddGapPeriod(ctx, session, msg, b)
	case "adjust_next_in":
		handleAdjustNextIn(ctx, session, msg, b)
	case "keep_conflict":
		handleKeepConflict(ctx, session, msg, b)
	case "cancel_edit":
		handleCancelEdit(ctx, session, msg, b)
	case "show_report":
		handleShowReport(ctx, session, msg, b)

	case "add_period":
		handleAddPeriod(ctx, msg, b)

	case "add_tail":
		handleAddTail(ctx, session, msg, b)

	case "add_head":
		handleAddHead(ctx, session, msg, b)

	case "add_full":
		handleAddFull(ctx, session, msg, b)

	case "edit_period":
		handleEditPeriod(ctx, session, msg, b)

	case "adjust_prev_out":
		handleAdjustPrevOut(ctx, session, msg, b)
		handlePeriodsCommand(ctx, session, msg, b)

	case "edit_in":
		handleEdinIn(ctx, session, msg, b)

	case "edit_out":
		handleEditOut(ctx, session, msg, b)

	case "edit_country":
		handleEditCountry(ctx, session, msg, b)

	default:
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: "❓ Неизвестная кнопка."})
	}

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: callback.ID})
}

func handleStartCommand(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        "🔘 Выберите действие:",
		ReplyMarkup: keyboard.BuildMainMenu(s),
	})
}

func handleHelpCommand(ctx context.Context, msg *models.Message, b *bot.Bot) {
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

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        helpText,
		ReplyMarkup: keyboard.BuildBackToMenu(),
	})
}

func handleCommandsCommand(ctx context.Context, msg *models.Message, b *bot.Bot) {
	txt := `/start - главное меню
/help - справка
/upload_report - загрузить данные
/periods - показать периоды
/reset - сбросить данные`
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        txt,
		ReplyMarkup: keyboard.BuildBackToMenu(),
	})
}

func handleResetCommand(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	if s.IsEmpty() {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "📭 У вас пока нет сохранённых периодов."})
		return
	}
	s.Data = model.Data{}
	s.Backup = model.Data{}
	s.Temp = nil
	_ = os.Remove(fmt.Sprintf("%s/data.json", s.HistoryDir))
	s.SaveSession()

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        "✅ Данные сброшены.",
		ReplyMarkup: keyboard.BuildBackToMenu(),
	})
}

func handleSetDateCommand(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	if s.IsEmpty() {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "📭 У вас пока нет сохранённых периодов."})
		return
	}
	s.PendingAction = "awaiting_date"
	s.SaveSession()
	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "📅 Введите дату в формате ДД.ММ.ГГГГ:"})
}

func handleUploadCommand(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	s.Data.Current = "upload_pending"
	s.SaveSession()

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        "📎 Пришлите JSON-файл документом.",
		ReplyMarkup: keyboard.BuildBackToMenu(),
	})
}

func handlePeriodsCommand(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	if s.IsEmpty() {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "📭 У вас пока нет сохранённых периодов."})
		return
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        s.BuildPeriodsList(),
		ReplyMarkup: keyboard.BuildPeriodsMenu(),
	})
}

func handleAddGapPeriod(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	chatID := msg.Chat.ID

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

	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: "➕ Добавлен период «unknown». Дата въезда обновлена."})
	handlePeriodsCommand(ctx, s, msg, b)
}

func handleAdjustNextIn(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	index := s.EditingIndex
	if index+1 >= len(s.Data.Periods) {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Ошибка: следующего периода не существует."})
		return
	}

	newOut, err := utils.ParseDate(s.TempEditedOut)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Ошибка при обработке даты."})
		return
	}

	// ✅ Обновляем out у текущего периода и in у следующего
	s.Data.Periods[index].Out = s.TempEditedOut
	s.Data.Periods[index+1].In = newOut.AddDate(0, 0, 1).Format("02.01.2006")

	s.PendingAction = ""
	s.TempEditedOut = ""
	s.SaveSession()

	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "📌 Следующий период сдвинут, дата выезда обновлена."})
	handlePeriodsCommand(ctx, s, msg, b)
}

func handleKeepConflict(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	if s.PendingAction == "confirm_conflict_in" {
		s.Data.Periods[s.EditingIndex].In = s.TempEditedIn
		s.PendingAction = ""
		s.SaveSession()
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "✅ Дата въезда обновлена."})
	} else if s.PendingAction == "confirm_conflict_out" {
		s.Data.Periods[s.EditingIndex].Out = s.TempEditedOut
		s.PendingAction = ""
		s.SaveSession()
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "✅ Дата выезда обновлена."})
	} else {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⚠️ Нет ожидаемого конфликта."})
		return
	}

	handlePeriodsCommand(ctx, s, msg, b)
}

func handleCancelEdit(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	s.PendingAction = ""
	s.TempEditedOut = ""
	s.SaveSession()

	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "❌ Изменение отменено."})
	handlePeriodsCommand(ctx, s, msg, b)
}

// handleBack cancels the current step and shows the appropriate menu.
func handleBack(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	switch s.PendingAction {
	case "awaiting_edit_index":
		s.PendingAction = ""
		s.SaveSession()
		handlePeriodsCommand(ctx, s, msg, b)
	case "awaiting_delete_index":
		s.PendingAction = ""
		s.SaveSession()
		handlePeriodsCommand(ctx, s, msg, b)
	case "awaiting_edit_field":
		handleEditPeriod(ctx, s, msg, b)
	case "awaiting_new_in", "awaiting_new_out", "awaiting_new_country":
		s.PendingAction = "awaiting_edit_field"
		s.SaveSession()
		from := s.Data.Periods[s.EditingIndex].In
		till := s.Data.Periods[s.EditingIndex].Out
		txt := fmt.Sprintf("Выбран период с %s по %s. Что изменить?", from, till)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      msg.Chat.ID,
			Text:        txt,
			ReplyMarkup: keyboard.BuildEditFieldMenu(),
		})
	default:
		handleStartCommand(ctx, s, msg, b)
	}
}

func handleShowReport(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	report := reportbuilder.BuildReport(s.Data)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        report,
		ReplyMarkup: keyboard.BuildBackToMenu(),
	})
}

func handleAddPeriod(ctx context.Context, msg *models.Message, b *bot.Bot) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        "➕ Что добавить?",
		ReplyMarkup: keyboard.BuildAddPeriodMenu(),
	})
}

func handleAddTail(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	s.PendingAction = "awaiting_tail_out"
	s.SaveSession()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        "📆 Введите дату выезда (ДД.ММ.ГГГГ):",
		ReplyMarkup: keyboard.BuildBackToMenu(),
	})
}

func handleAddHead(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	s.PendingAction = "awaiting_head_in"
	s.SaveSession()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        "📆 Введите дату въезда (ДД.ММ.ГГГГ):",
		ReplyMarkup: keyboard.BuildBackToMenu(),
	})
}

func handleAddFull(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	s.PendingAction = "awaiting_add_in"
	s.SaveSession()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        "📆 Введите дату въезда (ДД.ММ.ГГГГ):",
		ReplyMarkup: keyboard.BuildBackToMenu(),
	})
}

func handleEditPeriod(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	if s.IsEmpty() {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "📭 Нет сохранённых периодов для редактирования."})
		return
	}

	s.PendingAction = "awaiting_edit_index"
	s.SaveSession()

	text := s.BuildPeriodsList() + "\n✏️ Введите номер периода для редактирования:"
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        text,
		ReplyMarkup: keyboard.BuildBack(),
	})
}

func handleAdjustPrevOut(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	newIn, _ := utils.ParseDate(s.TempEditedIn)

	s.Data.Periods[s.EditingIndex-1].Out = newIn.Format("02.01.2006")
	s.Data.Periods[s.EditingIndex].In = newIn.Format("02.01.2006")
	s.PendingAction = ""
	s.TempEditedIn = ""
	s.SaveSession()

	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "📌 Предыдущий период подвинут. Дата въезда обновлена."})
	handlePeriodsCommand(ctx, s, msg, b)
}

func handleEdinIn(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	s.PendingAction = "awaiting_new_in"
	s.SaveSession()
	curr := s.Data.Periods[s.EditingIndex].In
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        fmt.Sprintf("✏️ Текущая дата въезда: %s. Введите новую:", curr),
		ReplyMarkup: keyboard.BuildBack(),
	})
}

func handleEditOut(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	s.PendingAction = "awaiting_new_out"
	s.SaveSession()
	curr := s.Data.Periods[s.EditingIndex].Out
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        fmt.Sprintf("✏️ Текущая дата выезда: %s. Введите новую:", curr),
		ReplyMarkup: keyboard.BuildBack(),
	})
}

func handleEditCountry(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	s.PendingAction = "awaiting_new_country"
	s.SaveSession()
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        "🌍 Введите новое название страны:",
		ReplyMarkup: keyboard.BuildBack(),
	})
}

func handleDeletePeriod(ctx context.Context, s *model.Session, msg *models.Message, b *bot.Bot) {
	if s.IsEmpty() {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "📭 Нет сохранённых периодов для удаления."})
		return
	}

	s.PendingAction = "awaiting_delete_index"
	s.SaveSession()

	text := s.BuildPeriodsList() + "\n🗑 Введите номер периода для удаления:"
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        text,
		ReplyMarkup: keyboard.BuildBack(),
	})
}

func handleAwaitingDeleteIndex(ctx context.Context, msg *models.Message, s *model.Session, b *bot.Bot) {
	index, err := strconv.Atoi(strings.TrimSpace(msg.Text))
	if err != nil || index < 1 || index > len(s.Data.Periods) {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Введите корректный номер периода."})
		return
	}

	idx := index - 1
	s.Data.Periods = append(s.Data.Periods[:idx], s.Data.Periods[idx+1:]...)
	s.PendingAction = ""
	s.SaveSession()

	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "🗑 Период удалён."})
	if s.IsEmpty() {
		handleStartCommand(ctx, s, msg, b)
	} else {
		handlePeriodsCommand(ctx, s, msg, b)
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
func removeInlineKeyboard(ctx context.Context, b *bot.Bot, chatID int64, messageID int) {
	// Telegram may return an error if the message is too old or was already
	// edited — silently ignore to avoid losing the user's message history.
	b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:    chatID,
		MessageID: messageID,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{},
		},
	})
}
