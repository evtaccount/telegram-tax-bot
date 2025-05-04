
package handler

import (
    "log"

    "github.com/evgenii-ev/go-tax-bot/internal/service"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Registry struct {
    bot *tgbotapi.BotAPI
    svc *service.Calculator
}

// Register binds message/​callback handling and spawns update loop.
func Register(api *tgbotapi.BotAPI, svc *service.Calculator) {
    r := &Registry{bot: api, svc: svc}
    go r.listen() // background
}

func (r *Registry) listen() {
    u := tgbotapi.NewUpdate(0)
    u.Timeout = 30

    updates := r.bot.GetUpdatesChan(u)

    for upd := range updates {
        switch {
        case upd.Message != nil:
            r.handleMessage(upd.Message)
        case upd.CallbackQuery != nil:
            r.handleCallback(upd.CallbackQuery)
        }
    }
}

// -------------------------------------------------------------------------

func (r *Registry) handleMessage(m *tgbotapi.Message) {
    if m.IsCommand() {
        r.handleCommand(m)
        return
    }
    // Non‑command messages could be JSON files etc.
    log.Printf("got message from %d: %q", m.Chat.ID, m.Text)
}
