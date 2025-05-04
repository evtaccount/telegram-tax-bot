package main

import (
	"log"
	"telegram-tax-bot/internal/bot"
	"telegram-tax-bot/internal/config"
	"telegram-tax-bot/internal/handler"
	"telegram-tax-bot/internal/service"
	"telegram-tax-bot/internal/storage"
)

func main() {
	cfg := config.Load()
	strg := storage.NewFileStorage(cfg.DataDir)
	sst := service.NewSessionStorage(strg)

	tgBot, err := bot.New(cfg.TelegramToken)
	if err != nil {
		log.Fatalf("telegram bot init: %v", err)
	}

	handler.Register(tgBot.API, sst)

	log.Println("Bot up and running 🚀")
	tgBot.Run()
}
