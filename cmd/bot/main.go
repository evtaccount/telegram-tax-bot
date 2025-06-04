package main

import (
	"log"
	"telegram-tax-bot/internal/bot"
	"telegram-tax-bot/internal/config"
	file_storage "telegram-tax-bot/internal/file_storage"
	"telegram-tax-bot/internal/handler"
	service "telegram-tax-bot/internal/user_storage"
)

func main() {
	cfg := config.Load()
	strg := file_storage.NewFileStorage(cfg.DataDir)
	sst := service.NewUserStorage(strg)

	tgBot, err := bot.New(cfg.TelegramToken)
	if err != nil {
		log.Fatalf("telegram bot init: %v", err)
	}

	handler.Register(tgBot.API, sst)

	log.Println("Bot up and running 🚀")
	tgBot.Run()
}
