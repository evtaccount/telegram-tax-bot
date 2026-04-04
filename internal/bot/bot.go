package bot

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// New creates a bot instance with the given token and default handler, registers
// the command menu, and returns it ready to Start.
func New(token string, handler bot.HandlerFunc) (*bot.Bot, error) {
	b, err := bot.New(token, bot.WithDefaultHandler(handler))
	if err != nil {
		return nil, err
	}

	b.SetMyCommands(context.Background(), &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{Command: "start", Description: "главное меню"},
			{Command: "help", Description: "справка"},
			{Command: "upload_report", Description: "загрузить данные"},
			{Command: "periods", Description: "показать периоды"},
			{Command: "reset", Description: "сбросить данные"},
			{Command: "commands", Description: "список команд"},
		},
	})

	return b, nil
}
