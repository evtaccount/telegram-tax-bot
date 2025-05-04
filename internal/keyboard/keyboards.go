
package keyboard

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func AddPeriodMenu() tgbotapi.InlineKeyboardMarkup {
    return tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("🗓 Хвостовой (только выезд)", "add_tail"),
        ),
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("⏮ Начальный (только въезд)", "add_head"),
        ),
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("📄 Полный (въезд+выезд)", "add_full"),
        ),
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("❌ Отменить", "start"),
        ),
    )
}
