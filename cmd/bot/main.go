package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"telegram-tax-bot/internal/bot"
	"telegram-tax-bot/internal/config"
	"telegram-tax-bot/internal/handler"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	b, err := bot.New(cfg.TelegramToken, handler.NewHandler())
	if err != nil {
		log.Fatalf("telegram bot init: %v", err)
	}

	log.Println("Bot up and running 🚀")
	b.Start(ctx)
}
