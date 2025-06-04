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

// BuildPeriodsMenu returns keyboard for period list actions.
func BuildPeriodsMenu() tgbotapi.ReplyKeyboardMarkup {
	markup := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("✏️ Отредактировать период")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("➕ Добавить период")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("🗑 Удалить период")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("📊 Отчёт")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("🔙 Назад в меню")),
	)
	markup.ResizeKeyboard = true
	return markup
}

// BuildAddPeriodMenu returns keyboard for choosing type of period to add.
func BuildAddPeriodMenu() tgbotapi.ReplyKeyboardMarkup {
	markup := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("🗓 Хвостовой (только выезд)")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("⏮ Начальный (только въезд)")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("📄 Полный (въезд+выезд)")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("❌ Отменить")),
	)
	markup.ResizeKeyboard = true
	return markup
}

// BuildEditFieldMenu returns keyboard with editable fields of a period.
func BuildEditFieldMenu() tgbotapi.ReplyKeyboardMarkup {
	markup := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("📅 Изменить дату въезда (in)")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("📆 Изменить дату выезда (out)")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("🌍 Изменить страну")),
	)
	markup.ResizeKeyboard = true
	return markup
}

// BuildResolveOptions returns keyboard for conflict resolution with a move option.
func BuildResolveOptions(move string) tgbotapi.ReplyKeyboardMarkup {
	markup := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(move)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("✅ Оставить как есть")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("❌ Отменить")),
	)
	markup.ResizeKeyboard = true
	return markup
}
