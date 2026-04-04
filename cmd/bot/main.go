package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	b, err := bot.New(cfg.TelegramToken, handler.NewHandler(sst))
	if err != nil {
		log.Fatalf("telegram bot init: %v", err)
	}

	log.Println("Bot up and running 🚀")
	b.Start(ctx)
}
