package handler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Registry struct{}

// NewHandler returns a bot.HandlerFunc that dispatches messages and callbacks.
func NewHandler() bot.HandlerFunc {
	r := &Registry{}
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		switch {
		case update.Message != nil:
			r.handleMessage(ctx, b, update.Message)
		case update.CallbackQuery != nil:
			r.handleCallback(ctx, b, update.CallbackQuery)
		}
	}
}
