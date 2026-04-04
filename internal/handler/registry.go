package handler

import (
	"context"

	user_storage "telegram-tax-bot/internal/user_storage"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Registry struct {
	ust *user_storage.UserStorage
}

// NewHandler returns a bot.HandlerFunc that dispatches messages and callbacks.
func NewHandler(ust *user_storage.UserStorage) bot.HandlerFunc {
	r := &Registry{ust: ust}
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		switch {
		case update.Message != nil:
			r.handleMessage(ctx, b, update.Message)
		case update.CallbackQuery != nil:
			r.handleCallback(ctx, b, update.CallbackQuery)
		}
	}
}
