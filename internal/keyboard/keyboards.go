package keyboard

import (
	"telegram-tax-bot/internal/model"

	"github.com/go-telegram/bot/models"
)

func btn(text string) models.KeyboardButton {
	return models.KeyboardButton{Text: text}
}

func row(buttons ...models.KeyboardButton) []models.KeyboardButton {
	return buttons
}

func markup(rows ...[]models.KeyboardButton) *models.ReplyKeyboardMarkup {
	return &models.ReplyKeyboardMarkup{
		Keyboard:       rows,
		ResizeKeyboard: true,
	}
}

func BuildBackToMenu() *models.ReplyKeyboardMarkup {
	return markup(row(btn("🔙 Назад в меню")))
}

// BuildBack returns a keyboard with a single "Назад" button.
func BuildBack() *models.ReplyKeyboardMarkup {
	return markup(row(btn("🔙 Назад")))
}

func BuildMainMenu(s *model.Session) *models.ReplyKeyboardMarkup {
	if s.IsEmpty() {
		return markup(
			row(btn("📎 Загрузить файл")),
			row(btn("ℹ️ Помощь")),
			row(btn("📖 Команды")),
		)
	}
	return markup(
		row(btn("📋 Показать текущие данные")),
		row(btn("📊 Отчёт")),
		row(btn("📅 Отчёт на заданную дату")),
		row(btn("📎 Загрузить новый файл")),
		row(btn("📤 Экспорт"), btn("🗑 Сбросить")),
		row(btn("ℹ️ Помощь")),
		row(btn("📖 Команды")),
	)
}

// BuildPeriodsMenu returns keyboard for period list actions.
func BuildPeriodsMenu() *models.ReplyKeyboardMarkup {
	return markup(
		row(btn("✏️ Отредактировать период")),
		row(btn("➕ Добавить период")),
		row(btn("🗑 Удалить период")),
		row(btn("📊 Отчёт")),
		row(btn("🔙 Назад в меню")),
	)
}

// BuildAddPeriodMenu returns keyboard for choosing type of period to add.
func BuildAddPeriodMenu() *models.ReplyKeyboardMarkup {
	return markup(
		row(btn("📄 Полный период (въезд + выезд)")),
		row(btn("🔚 Был в стране до... (только вые��д)")),
		row(btn("🔜 Прибыл и остаюсь (только въезд)")),
		row(btn("❌ Отменить")),
	)
}

// BuildEditFieldMenu returns keyboard with editable fields of a period.
func BuildEditFieldMenu() *models.ReplyKeyboardMarkup {
	return markup(
		row(btn("📅 Изменить дату въезда (in)")),
		row(btn("📆 Изменить дату выезда (out)")),
		row(btn("🌍 Изменить страну")),
		row(btn("🔙 Назад")),
	)
}

// BuildConfirm returns a keyboard with confirm/cancel buttons.
func BuildConfirm() *models.ReplyKeyboardMarkup {
	return markup(
		row(btn("✅ Да, подтверждаю")),
		row(btn("❌ Отменить")),
	)
}

// BuildCountrySelect returns a keyboard with previously used countries + back button.
// The user can also type a new country name as free text.
func BuildCountrySelect(countries []string) *models.ReplyKeyboardMarkup {
	var rows [][]models.KeyboardButton
	for _, c := range countries {
		rows = append(rows, row(btn(c)))
	}
	rows = append(rows, row(btn("🔙 Назад")))
	return &models.ReplyKeyboardMarkup{
		Keyboard:       rows,
		ResizeKeyboard: true,
	}
}

// BuildResolveOptions returns keyboard for conflict resolution with a move option.
func BuildResolveOptions(move string) *models.ReplyKeyboardMarkup {
	return markup(
		row(btn(move)),
		row(btn("✅ Оставить как есть")),
		row(btn("❌ Отменить")),
	)
}
