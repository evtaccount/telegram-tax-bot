package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot is a thin wrapper that exposes underlying API plus a blocking Run().
type Bot struct {
	API *tgbotapi.BotAPI
}

func New(token string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		return nil, err
	}

	api.Debug = false
	return &Bot{API: api}, nil
}

// Run blocks forever; all handling goroutines are launched elsewhere.
func (b *Bot) Run() {
	select {}
}
