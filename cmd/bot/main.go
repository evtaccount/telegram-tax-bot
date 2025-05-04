package main

import (
	"log"

	"github.com/evgenii-ev/go-tax-bot/internal/bot"
	"github.com/evgenii-ev/go-tax-bot/internal/config"
	"github.com/evgenii-ev/go-tax-bot/internal/handler"
	"github.com/evgenii-ev/go-tax-bot/internal/service"
	"github.com/evgenii-ev/go-tax-bot/internal/storage"
)

func main() {
	cfg := config.Load()
	strg := storage.NewFileStorage(cfg.DataDir)

	svc := service.NewCalculator(strg)

	tgBot, err := bot.New(cfg.TelegramToken)
	if err != nil {
		log.Fatalf("telegram bot init: %v", err)
	}

	handler.Register(tgBot.API, svc)

	log.Println("Bot up and running 🚀")
	tgBot.Run()
}
