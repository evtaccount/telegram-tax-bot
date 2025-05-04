package handler

import (
	"log"
	"time"

	"github.com/evgenii-ev/go-tax-bot/internal/keyboard"
	"github.com/evgenii-ev/go-tax-bot/internal/model"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (r *Registry) handleCallback(c *tgbotapi.CallbackQuery) {
	data := c.Data
	switch data {
	case "add_period":
		r.handleAddPeriodMenu(c)
	default:
		r.handleUnknownCallback(c)
	}
}

func (r *Registry) handleAddPeriodMenu(c *tgbotapi.CallbackQuery) {
	msg := tgbotapi.NewEditMessageTextAndMarkup(
		c.Message.Chat.ID,
		c.Message.MessageID,
		"➕ Что добавить?",
		keyboard.AddPeriodMenu(),
	)
	if _, err := r.bot.Send(msg); err != nil {
		log.Println("edit add_period menu:", err)
	}
}

func (r *Registry) handleUnknownCallback(c *tgbotapi.CallbackQuery) {
	cb := tgbotapi.NewCallback(c.ID, "Функция ещё не реализована")
	cb.ShowAlert = false

	if _, err := r.bot.Request(cb); err != nil {
		log.Println("callback answer:", err)
	}
}

// A very naive full period append for demo
func (r *Registry) appendFullPeriod(chatID int64, country string, in, out time.Time) error {
	data, err := r.svc.LoadUser(chatID)
	if err != nil {
		return err
	}
	data.Periods = append(data.Periods, model.Period{
		Country: country,
		In:      &in,
		Out:     out,
	})
	return r.svc.SaveUser(data)
}
