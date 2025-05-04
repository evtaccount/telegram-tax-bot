package main

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	bot, err := tgbotapi.NewBotAPI(getBotToken())
	if err != nil {
		log.Panic(err)
	}
	log.Printf("🟢 Бот запущен как @%s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {

		// === 📌 Обработка callback кнопок ===
		if update.CallbackQuery != nil {
			callback := update.CallbackQuery
			userID := callback.From.ID
			s := getSession(userID)
			data := callback.Data
			chatID := callback.Message.Chat.ID

			switch data {
			case "start":
				handleStartCommand(s, callback.Message, bot)
			case "help":
				handleHelpCommand(callback.Message, bot)
			case "reset":
				handleResetCommand(s, callback.Message, bot)
			case "set_date":
				handleSetDateCommand(s, callback.Message, bot)
			case "upload_report", "upload_file":
				handleUploadCommand(s, callback.Message, bot)
			case "periods":
				handlePeriodsCommand(s, callback.Message, bot)
			case "add_gap_period":
				handleAddGapPeriod(s, callback, bot)
			case "adjust_next_in":
				handleAdjustNextIn(s, callback.Message, bot)
			case "keep_conflict":
				handleKeepConflict(callback.Message, s, bot)
			case "cancel_edit":
				handleCancelEdit(s, callback.Message, bot)
			case "show_report":
				report := buildReport(s.Data)
				msg := tgbotapi.NewMessage(chatID, report)
				msg.ReplyMarkup = buildBackToMenu()
				bot.Send(msg)
			case "add_period":
				// меню выбора варианта добавления
				reply := tgbotapi.NewMessage(chatID, "➕ Что добавить?")
				reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
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
				bot.Send(reply)

			case "add_tail":
				s.PendingAction = "awaiting_tail_out"
				saveSession(s)

				replay := tgbotapi.NewMessage(chatID, "📆 Введите дату выезда (ДД.MM.YYYY):")
				replay.ReplyMarkup = buildBackToMenu()
				bot.Send(replay)

			case "add_head":
				s.PendingAction = "awaiting_head_in"
				saveSession(s)

				replay := tgbotapi.NewMessage(chatID, "📆 Введите дату въезда (ДД.MM.YYYY):")
				replay.ReplyMarkup = buildBackToMenu()
				bot.Send(replay)

			case "add_full":
				s.PendingAction = "awaiting_full_in"
				saveSession(s)

				replay := tgbotapi.NewMessage(chatID, "📆 Введите дату въезда (ДД.MM.YYYY):")
				replay.ReplyMarkup = buildBackToMenu()
				bot.Send(replay)
				if isEmpty(s) {
					bot.Send(tgbotapi.NewMessage(chatID, "📭 Нет сохранённых периодов для удаления."))
				} else {
					s.PendingAction = "awaiting_delete_index"
					saveSession(s)
					bot.Send(tgbotapi.NewMessage(chatID, "❌ Укажите номер периода, который нужно удалить:"))
				}
			case "edit_period":
				if isEmpty(s) {
					bot.Send(tgbotapi.NewMessage(chatID, "📭 Нет сохранённых периодов для редактирования."))
				} else {
					s.PendingAction = "awaiting_edit_index"
					saveSession(s)
					bot.Send(tgbotapi.NewMessage(chatID, "✏️ Введите номер периода, который хотите отредактировать:"))
				}
			case "adjust_prev_out":
				newIn, _ := parseDate(s.TempEditedIn)
				s.Data.Periods[s.EditingIndex-1].Out = newIn.Format("02.01.2006")
				s.Data.Periods[s.EditingIndex].In = newIn.Format("02.01.2006")
				s.PendingAction = ""
				s.TempEditedIn = ""
				saveSession(s)
				bot.Send(tgbotapi.NewMessage(chatID, "📌 Предыдущий период подвинут. Дата въезда обновлена."))
				handlePeriodsCommand(s, callback.Message, bot)
			case "edit_in":
				s.PendingAction = "awaiting_new_in"
				saveSession(s)
				curr := s.Data.Periods[s.EditingIndex].In
				bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✏️ Текущая дата %s. Введите новую дату въезда:", curr)))
			case "edit_out":
				s.PendingAction = "awaiting_new_out"
				saveSession(s)
				curr := s.Data.Periods[s.EditingIndex].Out
				bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✏️ Текущая дата %s. Введите новую дату выезда:", curr)))
			case "edit_country":
				s.PendingAction = "awaiting_new_country"
				saveSession(s)
				bot.Send(tgbotapi.NewMessage(chatID, "🌍 Введите новое название страны:"))
			default:
				bot.Send(tgbotapi.NewMessage(chatID, "❓ Неизвестная кнопка."))
			}

			bot.Request(tgbotapi.NewCallback(callback.ID, ""))
			continue
		}

		// Обработка текстовых сообщений
		if update.Message != nil {
			msg := update.Message
			userID := msg.From.ID
			s := getSession(userID)
			text := msg.Text

			switch s.PendingAction {
			case "awaiting_edit_index":
				handleAwaitingEditIndex(msg, s, bot)
				return
			case "awaiting_date":
				handleAwaitingDate(msg, s, bot)
				return
			case "awaiting_new_in":
				handleAwaitingNewIn(msg, s, bot)
				return
			case "awaiting_new_out":
				handleAwaitingNewOut(msg, s, bot)
				return
			case "awaiting_new_country":
				handleAwaitingNewCountry(msg, s, bot)
				return
			case "awaiting_add_out":
				handleAwaitingAddOut(msg, s, bot)
				return
			case "awaiting_add_country":
				handleAwaitingAddCountry(msg, s, bot)
				return
			case "awaiting_add_open_country":
				handleAddOpenCountry(msg, s, bot)
			case "awaiting_add_in":
				text := strings.TrimSpace(msg.Text)
				_, err := parseDate(text)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Неверный формат даты. Введите ДД.ММ.ГГГГ"))
					return
				}
				s.Temp = []Period{{In: text}} // сохраняем только дату in во временное хранилище
				s.PendingAction = "awaiting_add_out"
				saveSession(s)
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📆 Введите дату выезда (ДД.ММ.ГГГГ):"))
			}

			// ✅ Загрузка JSON-файла
			if msg.Document != nil && s.Data.Current == "upload_pending" {
				handleInputFile(msg, s, bot)
				continue
			}

			// ✅ Команды
			switch {
			case strings.HasPrefix(text, "/start"):
				handleStartCommand(s, msg, bot)
			case strings.HasPrefix(text, "/help"):
				handleHelpCommand(msg, bot)
			default:
				if strings.HasPrefix(text, "{") {
					handleJSONInput(msg, s, bot)
				} else {
					bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❓ Неизвестная команда. Введите /help для списка."))
				}
			}
		}
	}
}

var sessions = map[int64]*Session{}

var countryCodeMap = map[string]string{
	"Австралия":            "AU",
	"Австрия":              "AT",
	"Азербайджан":          "AZ",
	"Албания":              "AL",
	"Алжир":                "DZ",
	"Американское Самоа":   "AS",
	"Ангола":               "AO",
	"Андорра":              "AD",
	"Аргентина":            "AR",
	"Армения":              "AM",
	"Афганистан":           "AF",
	"Бангладеш":            "BD",
	"Беларусь":             "BY",
	"Бельгия":              "BE",
	"Болгария":             "BG",
	"Боливия":              "BO",
	"Босния и Герцеговина": "BA",
	"Бразилия":             "BR",
	"Великобритания":       "GB",
	"Венгрия":              "HU",
	"Венесуэла":            "VE",
	"Вьетнам":              "VN",
	"Германия":             "DE",
	"Гонконг":              "HK",
	"Греция":               "GR",
	"Грузия":               "GE",
	"Дания":                "DK",
	"Египет":               "EG",
	"Израиль":              "IL",
	"Индия":                "IN",
	"Индонезия":            "ID",
	"Иордания":             "JO",
	"Ирак":                 "IQ",
	"Иран":                 "IR",
	"Ирландия":             "IE",
	"Исландия":             "IS",
	"Испания":              "ES",
	"Италия":               "IT",
	"Казахстан":            "KZ",
	"Камбоджа":             "KH",
	"Канада":               "CA",
	"Катар":                "QA",
	"Кения":                "KE",
	"Кипр":                 "CY",
	"Китай":                "CN",
	"Колумбия":             "CO",
	"Коста-Рика":           "CR",
	"Куба":                 "CU",
	"Кыргызстан":           "KG",
	"Латвия":               "LV",
	"Ливан":                "LB",
	"Литва":                "LT",
	"Люксембург":           "LU",
	"Малайзия":             "MY",
	"Мальта":               "MT",
	"Марокко":              "MA",
	"Мексика":              "MX",
	"Молдова":              "MD",
	"Монголия":             "MN",
	"Нидерланды":           "NL",
	"Новая Зеландия":       "NZ",
	"Норвегия":             "NO",
	"ОАЭ":                  "AE",
	"Пакистан":             "PK",
	"Панама":               "PA",
	"Перу":                 "PE",
	"Польша":               "PL",
	"Португалия":           "PT",
	"Россия":               "RU",
	"Румыния":              "RO",
	"Саудовская Аравия":    "SA",
	"Сербия":               "RS",
	"Сингапур":             "SG",
	"Словакия":             "SK",
	"Словения":             "SI",
	"США":                  "US",
	"Таджикистан":          "TJ",
	"Таиланд":              "TH",
	"Тунис":                "TN",
	"Туркменистан":         "TM",
	"Турция":               "TR",
	"Узбекистан":           "UZ",
	"Украина":              "UA",
	"Филиппины":            "PH",
	"Финляндия":            "FI",
	"Франция":              "FR",
	"Хорватия":             "HR",
	"Чехия":                "CZ",
	"Чили":                 "CL",
	"Швейцария":            "CH",
	"Швеция":               "SE",
	"Шри-Ланка":            "LK",
	"Эстония":              "EE",
	"ЮАР":                  "ZA",
	"Южная Корея":          "KR",
	"Япония":               "JP",
}

const (
	dataDir = "./data"
	logDir  = "./logs"
)
