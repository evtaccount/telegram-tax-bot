package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"telegram-tax-bot/internal/keyboard"
	"telegram-tax-bot/internal/manager"
	"telegram-tax-bot/internal/model"
	"telegram-tax-bot/internal/utils"
	"time"

	reportbuilder "telegram-tax-bot/internal/report_builder"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (r *Registry) handleMessage(ctx context.Context, b *bot.Bot, msg *models.Message) {
	userID := msg.From.ID
	s := manager.GetSession(userID)
	s.Mu.Lock()
	defer s.Mu.Unlock()
	text := msg.Text

	// ✅ Загрузка JSON-файла
	if msg.Document != nil && s.PendingAction == "awaiting_upload" {
		handleInputFile(ctx, msg, s, b)
		return
	}

	// ✅ Команды и кнопки имеют приоритет над ожидаемыми действиями
	switch {
	case strings.HasPrefix(text, "/start"), text == "🔙 Назад в меню":
		handleStartCommand(ctx, s, msg, b)
		return
	case text == "🔙 Назад":
		handleBack(ctx, s, msg, b)
		return
	case strings.HasPrefix(text, "/help"), text == "ℹ️ Помощь":
		handleHelpCommand(ctx, msg, b)
		return
	case strings.HasPrefix(text, "/commands"), text == "📖 Команды":
		handleCommandsCommand(ctx, msg, b)
		return
	case text == "📎 Загрузить файл", text == "📎 Загрузить новый файл":
		handleUploadCommand(ctx, s, msg, b)
		return
	case strings.HasPrefix(text, "/upload_report"):
		handleUploadCommand(ctx, s, msg, b)
		return
	case text == "🗑 Сбросить", strings.HasPrefix(text, "/reset"):
		handleResetCommand(ctx, s, msg, b)
		return
	case strings.HasPrefix(text, "/periods"):
		handlePeriodsCommand(ctx, s, msg, b)
		return
	case text == "📅 Отчёт на заданную дату":
		handleSetDateCommand(ctx, s, msg, b)
		return
	case text == "📋 Показать текущие данные":
		handlePeriodsCommand(ctx, s, msg, b)
		return
	case text == "📊 Отчёт":
		handleShowReport(ctx, s, msg, b)
		return
	case text == "✏️ Отредактировать период":
		handleEditPeriod(ctx, s, msg, b)
		return
	case text == "➕ Добавить период":
		handleAddPeriod(ctx, msg, b)
		return
	case text == "🗑 Удалить период":
		handleDeletePeriod(ctx, s, msg, b)
		return
	case text == "📅 Изменить дату въезда (in)":
		handleEdinIn(ctx, s, msg, b)
		return
	case text == "📆 Изменить дату выезда (out)":
		handleEditOut(ctx, s, msg, b)
		return
	case text == "🌍 Изменить страну":
		handleEditCountry(ctx, s, msg, b)
		return
	case text == "🗓 Хвостовой (только выезд)":
		handleAddTail(ctx, s, msg, b)
		return
	case text == "⏮ Начальный (только въезд)":
		handleAddHead(ctx, s, msg, b)
		return
	case text == "📄 Полный (въезд+выезд)":
		handleAddFull(ctx, s, msg, b)
		return
	case text == "📌 Подвинуть предыдущий период":
		handleAdjustPrevOut(ctx, s, msg, b)
		return
	case text == "📌 Подвинуть следующий период":
		handleAdjustNextIn(ctx, s, msg, b)
		return
	case text == "✅ Оставить как есть":
		handleKeepConflict(ctx, s, msg, b)
		return
	case text == "❌ Отменить":
		if strings.HasPrefix(s.PendingAction, "resolve_") {
			handleCancelEdit(ctx, s, msg, b)
		} else if s.PendingAction == "" {
			// Отмена выбора действия во время просмотра списка периодов
			handlePeriodsCommand(ctx, s, msg, b)
		} else {
			handleStartCommand(ctx, s, msg, b)
		}
		return
	}

	// ✅ Ожидаемые действия
	switch s.PendingAction {
	case "awaiting_edit_index":
		handleAwaitingEditIndex(ctx, msg, s, b)
		return
	case "awaiting_date":
		handleAwaitingDate(ctx, msg, s, b)
		return
	case "awaiting_new_in":
		handleAwaitingNewIn(ctx, msg, s, b)
		return
	case "awaiting_new_out":
		handleAwaitingNewOut(ctx, msg, s, b)
		return
	case "awaiting_new_country":
		handleAwaitingNewCountry(ctx, msg, s, b)
		return
	case "awaiting_add_out":
		handleAwaitingAddOut(ctx, msg, s, b)
		return
	case "awaiting_add_country":
		handleAwaitingAddCountry(ctx, msg, s, b)
		return
	case "awaiting_add_open_country":
		handleAddOpenCountry(ctx, msg, s, b)
		return
	case "awaiting_tail_out":
		handleAwaitingTailOut(ctx, msg, s, b)
		return
	case "awaiting_tail_country":
		handleAwaitingTailCountry(ctx, msg, s, b)
		return
	case "awaiting_head_in":
		handleAwaitingHeadIn(ctx, msg, s, b)
		return
	case "awaiting_head_country":
		handleAwaitingHeadCountry(ctx, msg, s, b)
		return
	case "awaiting_delete_index":
		handleAwaitingDeleteIndex(ctx, msg, s, b)
		return
	case "awaiting_add_in":
		handleAddin(ctx, msg, s, b)
		return
	}

	if strings.HasPrefix(text, "{") {
		handleJSONInput(ctx, msg, s, b)
	} else {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "❓ Неизвестная команда. Введите /help, чтобы посмотреть список."})
	}
}

func handleAwaitingEditIndex(ctx context.Context, msg *models.Message, s *model.Session, b *bot.Bot) {
	index, err := strconv.Atoi(strings.TrimSpace(msg.Text))

	if err != nil || index < 1 || index > len(s.Data.Periods) {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Введите корректный номер периода."})
		return
	}

	s.EditingIndex = index - 1
	s.PendingAction = "awaiting_edit_field"
	s.SaveSession()

	from := s.Data.Periods[s.EditingIndex].In
	till := s.Data.Periods[s.EditingIndex].Out
	msgText := fmt.Sprintf("Выбран период с %s по %s. Что изменить?", from, till)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      msg.Chat.ID,
		Text:        msgText,
		ReplyMarkup: keyboard.BuildEditFieldMenu(),
	})
}

func handleAwaitingDate(ctx context.Context, msg *models.Message, s *model.Session, b *bot.Bot) {
	date, err := utils.ParseDate(msg.Text)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Неверный формат даты. Используйте ДД.ММ.ГГГГ."})
		return
	}
	s.Data.Current = date.Format("02.01.2006")
	s.PendingAction = ""
	s.SaveSession()
	report := reportbuilder.BuildReport(s.Data)
	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: fmt.Sprintf("✅ Дата расчета установлена: %s\n\n%s", s.Data.Current, report)})
}

func handleAwaitingNewIn(ctx context.Context, msg *models.Message, s *model.Session, b *bot.Bot) {
	newDate, err := utils.ParseDate(msg.Text)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Неверный формат даты."})
		return
	}

	index := s.EditingIndex
	if index < 0 || index >= len(s.Data.Periods) {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⚠️ Ошибка: индекс периода вне допустимого диапазона."})
		return
	}

	curr := s.Data.Periods[index]
	oldDate, _ := utils.ParseDate(curr.In)
	if newDate.Equal(oldDate) {
		s.PendingAction = ""
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "ℹ️ Дата въезда не изменилась."})
		handlePeriodsCommand(ctx, s, msg, b)
		return
	}

	// Проверка конфликта с предыдущим периодом
	if index > 0 {
		prev := s.Data.Periods[index-1]
		prevOut, err := utils.ParseDate(prev.Out)
		if err == nil {
			switch {
			case newDate.Before(prevOut):
				s.TempEditedIn = newDate.Format("02.01.2006")
				s.PendingAction = "resolve_in_conflict"
				s.SaveSession()

				text := fmt.Sprintf("⚠️ Новая дата въезда пересекается с предыдущим периодом (%s). Что сделать?",
					utils.FormatDate(prevOut))
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:      msg.Chat.ID,
					Text:        text,
					ReplyMarkup: keyboard.BuildResolveOptions("📌 Подвинуть предыдущий период"),
				})
				return

			case newDate.After(prevOut.AddDate(0, 0, 1)):
				s.TempEditedIn = newDate.Format("02.01.2006")
				s.PendingAction = "resolve_in_gap"
				s.SaveSession()

				text := fmt.Sprintf("⚠️ Между %s и %s обнаружен разрыв. Что сделать?",
					utils.FormatDate(prevOut.AddDate(0, 0, 1)), utils.FormatDate(newDate))
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:      msg.Chat.ID,
					Text:        text,
					ReplyMarkup: keyboard.BuildResolveOptions("📌 Подвинуть предыдущий период"),
				})
				return
			}
		}
	}

	// Всё в порядке, обновляем
	s.Data.Periods[index].In = newDate.Format("02.01.2006")
	s.PendingAction = ""
	s.SaveSession()
	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "✅ Дата въезда обновлена."})
	handlePeriodsCommand(ctx, s, msg, b)
}

func handleAwaitingNewOut(ctx context.Context, msg *models.Message, s *model.Session, b *bot.Bot) {
	newDate, err := utils.ParseDate(msg.Text)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Неверный формат даты."})
		return
	}

	index := s.EditingIndex
	if index < 0 || index >= len(s.Data.Periods) {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⚠️ Ошибка: индекс периода вне допустимого диапазона."})
		return
	}

	curr := s.Data.Periods[index]
	oldDate, _ := utils.ParseDate(curr.Out)
	if newDate.Equal(oldDate) {
		s.PendingAction = ""
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "ℹ️ Дата выезда не изменилась."})
		handlePeriodsCommand(ctx, s, msg, b)
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
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:      msg.Chat.ID,
					Text:        text,
					ReplyMarkup: keyboard.BuildResolveOptions("📌 Подвинуть следующий период"),
				})
				return

			case newDate.Before(nextIn.AddDate(0, 0, -1)):
				s.TempEditedOut = newDate.Format("02.01.2006")
				s.PendingAction = "resolve_out_gap"
				s.SaveSession()

				text := fmt.Sprintf("⚠️ Между %s и %s образовался разрыв. Что сделать?",
					utils.FormatDate(newDate.AddDate(0, 0, 1)), utils.FormatDate(nextIn))
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:      msg.Chat.ID,
					Text:        text,
					ReplyMarkup: keyboard.BuildResolveOptions("📌 Подвинуть следующий период"),
				})
				return
			}
		}
	}

	// Всё в порядке, обновляем
	s.Data.Periods[index].Out = newDate.Format("02.01.2006")
	s.PendingAction = ""
	s.SaveSession()
	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "✅ Дата выезда обновлена."})
	handlePeriodsCommand(ctx, s, msg, b)
}

func handleAwaitingNewCountry(ctx context.Context, msg *models.Message, s *model.Session, b *bot.Bot) {
	newCountry := strings.TrimSpace(msg.Text)
	if newCountry == "" {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Название страны не может быть пустым."})
		return
	}
	s.Data.Periods[s.EditingIndex].Country = newCountry
	s.PendingAction = ""
	s.SaveSession()
	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "✅ Страна обновлена."})
	handlePeriodsCommand(ctx, s, msg, b)
}

func handleAwaitingAddOut(ctx context.Context, msg *models.Message, s *model.Session, b *bot.Bot) {
	if len(s.Temp) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⚠️ Внутренняя ошибка. Начните добавление заново."})
		s.PendingAction = ""
		return
	}
	date, err := utils.ParseDate(msg.Text)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Неверный формат даты. Используйте ДД.ММ.ГГГГ."})
		return
	}
	inDate, err := utils.ParseDate(s.Temp[0].In)
	if err != nil || date.Before(inDate) {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Дата выезда не может быть раньше даты въезда."})
		return
	}
	s.Temp[0].Out = date.Format("02.01.2006")
	s.PendingAction = "awaiting_add_country"
	s.SaveSession()
	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "🌍 Укажите название страны:"})
}

func handleAwaitingAddCountry(ctx context.Context, msg *models.Message, s *model.Session, b *bot.Bot) {
	country := strings.TrimSpace(msg.Text)
	if country == "" {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Страна не может быть пустой."})
		return
	}
	if len(s.Temp) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⚠️ Внутренняя ошибка: временный буфер пуст."})
		s.PendingAction = ""
		return
	}
	period := s.Temp[0]
	period.Country = country

	// Проверка хронологического порядка
	newIn, errIn := utils.ParseDate(period.In)
	if errIn != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Некорректная дата въезда."})
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
			b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Невозможно добавить период: нарушен хронологический порядок."})
			s.PendingAction = ""
			s.Temp = nil
			return
		}
	}

	s.Data.Periods = append(s.Data.Periods, period)
	s.Temp = nil
	s.PendingAction = ""
	s.SaveSession()

	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "✅ Новый период добавлен."})
	handlePeriodsCommand(ctx, s, msg, b)
}

func handleAddOpenCountry(ctx context.Context, msg *models.Message, s *model.Session, b *bot.Bot) {
	country := strings.TrimSpace(msg.Text)
	if country == "" {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Название страны не может быть пустым"})
		return
	}
	s.Data.Periods = append(s.Data.Periods, model.Period{
		In:      s.Data.Current,
		Out:     "",
		Country: country,
	})
	s.PendingAction = ""
	s.SaveSession()

	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "✅ Новый период добавлен."})
	handlePeriodsCommand(ctx, s, msg, b)
}

func handleAwaitingTailOut(ctx context.Context, msg *models.Message, s *model.Session, b *bot.Bot) {
	date, err := utils.ParseDate(msg.Text)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Неверный формат даты. Используйте ДД.ММ.ГГГГ."})
		return
	}

	s.Temp = []model.Period{{Out: date.Format("02.01.2006")}}
	s.PendingAction = "awaiting_tail_country"
	s.SaveSession()

	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "🌍 Укажите название страны:"})
}

func handleAwaitingTailCountry(ctx context.Context, msg *models.Message, s *model.Session, b *bot.Bot) {
	country := strings.TrimSpace(msg.Text)
	if country == "" {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Страна не может быть пустой."})
		return
	}
	if len(s.Temp) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⚠️ Внутренняя ошибка: начните добавление заново."})
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
				b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Дата выезда не может быть после начала первого периода."})
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

	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "✅ Новый период добавлен."})
	handlePeriodsCommand(ctx, s, msg, b)
}

func handleAwaitingHeadIn(ctx context.Context, msg *models.Message, s *model.Session, b *bot.Bot) {
	date, err := utils.ParseDate(msg.Text)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Неверный формат даты. Используйте ДД.ММ.ГГГГ."})
		return
	}

	s.Temp = []model.Period{{In: date.Format("02.01.2006")}}
	s.PendingAction = "awaiting_head_country"
	s.SaveSession()

	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "🌍 Укажите название страны:"})
}

func handleAwaitingHeadCountry(ctx context.Context, msg *models.Message, s *model.Session, b *bot.Bot) {
	country := strings.TrimSpace(msg.Text)
	if country == "" {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Страна не может быть пустой."})
		return
	}
	if len(s.Temp) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⚠️ Внутренняя ошибка: временный буфер пуст."})
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
			b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Невозможно добавить период: нарушен хронологический порядок."})
			s.PendingAction = ""
			s.Temp = nil
			return
		}
	}

	s.Data.Periods = append(s.Data.Periods, period)
	s.PendingAction = ""
	s.Temp = nil
	s.SaveSession()

	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "✅ Новый период добавлен."})
	handlePeriodsCommand(ctx, s, msg, b)
}

func handleAddin(ctx context.Context, msg *models.Message, s *model.Session, b *bot.Bot) {
	text := strings.TrimSpace(msg.Text)
	_, err := utils.ParseDate(text)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Неверный формат даты. Введите ДД.ММ.ГГГГ."})
		return
	}
	s.Temp = []model.Period{{In: text}}
	s.PendingAction = "awaiting_add_out"
	s.SaveSession()
	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "📆 Введите дату выезда (ДД.ММ.ГГГГ):"})
}

func handleJSONInput(ctx context.Context, msg *models.Message, s *model.Session, b *bot.Bot) {
	s.BackupSession()
	err := json.Unmarshal([]byte(msg.Text), &s.Data)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Ошибка в формате JSON."})
		return
	}
	if s.Data.Current == "" {
		s.Data.Current = time.Now().Format("02.01.2006")
	}
	s.SaveSession()
	report := reportbuilder.BuildReport(s.Data)
	b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: report})
}

func handleInputFile(ctx context.Context, msg *models.Message, s *model.Session, b *bot.Bot) {
	fileID := msg.Document.FileID
	file, err := b.GetFile(ctx, &bot.GetFileParams{FileID: fileID})
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Не удалось получить файл."})
		return
	}
	url := b.FileDownloadLink(file)
	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Get(url)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: "⛔ Не удалось загрузить файл."})
		return
	}

	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	msg.Text = string(body)
	s.PendingAction = ""
	handleJSONInput(ctx, msg, s, b)
}
