package handler

import (
	"log"

	"github.com/evgenii-ev/go-tax-bot/internal/keyboard"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (r *Registry) handleCommand(m *tgbotapi.Message) {
	switch m.Command() {
	case "start":
		r.cmdStart(m)
	case "help":
		r.cmdHelp(m)
	case "reset":
		r.cmdReset(m)
	case "add_period":
		r.cmdAddPeriod(m)
	default:
		r.reply(m.Chat.ID, "Неизвестная команда. Напишите /help")
	}
}

func (r *Registry) cmdStart(m *tgbotapi.Message) {
	r.reply(m.Chat.ID, "Привет! Я бот для подсчёта налогового резидентства."+"Загрузите JSON с периодами или используйте меню.")
}

func (r *Registry) cmdHelp(m *tgbotapi.Message) {
	help := `/start — приветствие
/help  — это сообщение
/reset — удалить все сохранённые данные
/add_period — добавить период вручную

Загрузите JSON‑файл с периодами или пользуйтесь кнопками.`
	msg := tgbotapi.NewMessage(m.Chat.ID, help)
	msg.ParseMode = "Markdown"
	if _, err := r.bot.Send(msg); err != nil {
		log.Print("send help:", err)
	}
}

func (r *Registry) cmdReset(m *tgbotapi.Message) {
	data, err := r.svc.LoadUser(m.Chat.ID)
	if err != nil {
		r.reply(m.Chat.ID, "Ошибка чтения данных: "+err.Error())
		return
	}
	data.Periods = nil
	if err := r.svc.SaveUser(data); err != nil {
		r.reply(m.Chat.ID, "Ошибка сохранения: "+err.Error())
		return
	}
	r.reply(m.Chat.ID, "Данные сброшены.")
}

func (r *Registry) cmdAddPeriod(m *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(m.Chat.ID, "➕ Что добавить?")
	msg.ReplyMarkup = keyboard.AddPeriodMenu()
	if _, err := r.bot.Send(msg); err != nil {
		log.Print("send add_period:", err)
	}
}

func (r *Registry) reply(chatID int64, text string) {
	if _, err := r.bot.Send(tgbotapi.NewMessage(chatID, text)); err != nil {
		log.Print("reply:", err)
	}
}
