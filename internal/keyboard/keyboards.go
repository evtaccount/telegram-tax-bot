package keyboard

import (
	"telegram-tax-bot/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func BuildBackToMenu() tgbotapi.ReplyKeyboardMarkup {
	markup := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔙 Назад в меню"),
		),
	)
	markup.ResizeKeyboard = true
	return markup
}

func BuildMainMenu(s *model.Session) tgbotapi.ReplyKeyboardMarkup {
	var rows [][]tgbotapi.KeyboardButton

	if s.IsEmpty() {
		rows = [][]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("📎 Загрузить файл")),
			tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("ℹ️ Помощь")),
		}
	} else {
		rows = [][]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("📋 Показать текущие данные")),
			tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("📊 Отчёт")),
			tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("📅 Отчёт на заданную дату")),
			tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("📎 Загрузить новый файл")),
			tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("🗑 Сбросить")),
			tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("ℹ️ Помощь")),
		}
	}

	markup := tgbotapi.NewReplyKeyboard(rows...)
	markup.ResizeKeyboard = true
	return markup
}
