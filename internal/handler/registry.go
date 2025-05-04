package handler

import (
	"telegram-tax-bot/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Registry struct {
	bot *tgbotapi.BotAPI
	sst *service.SessionStorate
}

// Register binds message/​callback handling and spawns update loop.
func Register(api *tgbotapi.BotAPI, sst *service.SessionStorate) {
	r := &Registry{bot: api, sst: sst}
	go r.listen() // background
}

func (r *Registry) listen() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := r.bot.GetUpdatesChan(u)

	for upd := range updates {
		switch {
		case upd.Message != nil:
			// === 📌 Обработка текстовых сообщений ===
			r.handleMessage(upd.Message)

		case upd.CallbackQuery != nil:
			// === 📌 Обработка callback кнопок ===
			r.handleCallback(upd.CallbackQuery)
		}
	}
}
