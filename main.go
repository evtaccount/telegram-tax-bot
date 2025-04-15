package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	dataDir = "./data"
	logDir  = "./logs"
)

type Period struct {
	In      string `json:"in,omitempty"`
	Out     string `json:"out,omitempty"`
	Country string `json:"country"`
}

type Data struct {
	Periods []Period `json:"periods"`
	Current string   `json:"current"`
}

type Session struct {
	UserID        int64
	Data          Data
	Backup        Data
	HistoryDir    string
	Temp          []Period
	EditingIndex  int
	PendingAction string
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

func ensureDirs(userID int64) string {
	path := fmt.Sprintf("%s/%d", dataDir, userID)
	os.MkdirAll(path, 0755)
	return path
}

func getBotToken() string {
	if data, err := os.ReadFile("/run/secrets/telegram_bot_token"); err == nil {
		token := strings.TrimSpace(string(data))
		if token != "" {
			log.Println("✅ Токен получен из Docker Secret (/run/secrets/telegram_bot_token)")
			return token
		}
	}
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token != "" {
		log.Println("✅ Токен получен из переменной окружения (TELEGRAM_BOT_TOKEN)")
		return token
	}
	log.Fatal("❌ Токен не найден: отсутствует и Docker Secret, и переменная окружения")
	return ""
}

func countryToFlag(isoCode string) string {
	isoCode = strings.ToUpper(isoCode)
	if len(isoCode) != 2 {
		return ""
	}
	return string(rune(0x1F1E6+int(isoCode[0]-'A'))) + string(rune(0x1F1E6+int(isoCode[1]-'A')))
}

func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("02.01.2006", dateStr)
}

func formatDate(d time.Time) string {
	return d.Format("02.01.2006")
}

func backupSession(s *Session) {
	b, _ := json.MarshalIndent(s.Data, "", "  ")
	_ = os.WriteFile(fmt.Sprintf("%s/backup.json", s.HistoryDir), b, 0644)
	s.Backup = s.Data
}

func saveSession(s *Session) {
	b, _ := json.MarshalIndent(s.Data, "", "  ")
	_ = os.WriteFile(fmt.Sprintf("%s/data.json", s.HistoryDir), b, 0644)
	t := time.Now().Format("2006-01-02_15-04-05")
	_ = os.WriteFile(fmt.Sprintf("%s/report_%s.txt", s.HistoryDir, t), []byte(buildReport(s.Data)), 0644)
}

func buildReport(data Data) string {
	calcDate, _ := parseDate(data.Current)
	oneYearAgo := calcDate.AddDate(-1, 0, 0).AddDate(0, 0, 1)
	countryDays := make(map[string]int)
	var previousOutDate time.Time

	for i, period := range data.Periods {
		var inDate, outDate time.Time
		if period.Out != "" {
			outDate, _ = parseDate(period.Out)
		} else {
			outDate = calcDate
		}
		if i == 0 && period.In == "" {
			inDate = oneYearAgo
			if outDate.Before(oneYearAgo) {
				continue
			}
		} else {
			inDate, _ = parseDate(period.In)
		}
		if i > 0 && inDate.Before(previousOutDate) {
			return fmt.Sprintf("Ошибка: периоды не в хронологическом порядке (период %d)", i+1)
		}
		previousOutDate = outDate
		if outDate.Before(oneYearAgo) {
			continue
		}
		if inDate.Before(oneYearAgo) {
			inDate = oneYearAgo
		}
		if outDate.After(calcDate) {
			outDate = calcDate
		}
		if inDate.After(outDate) {
			continue
		}
		effectiveOutDate := outDate.AddDate(0, 0, 1)
		days := int(effectiveOutDate.Sub(inDate).Hours() / 24)
		countryDays[period.Country] += days
	}
	if len(countryDays) == 0 {
		return "Нет данных для анализа за указанный период."
	}
	type stat struct {
		Country string
		Days    int
	}
	var stats []stat
	for c, d := range countryDays {
		stats = append(stats, stat{c, d})
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Days > stats[j].Days
	})
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("Анализ за период: %s — %s\n\n", formatDate(oneYearAgo), formatDate(calcDate)))
	for _, s := range stats {
		iso := countryCodeMap[s.Country]
		flag := countryToFlag(iso)
		builder.WriteString(fmt.Sprintf("%s %s: %d дней\n", flag, s.Country, s.Days))
	}
	builder.WriteString("\n")
	if stats[0].Days >= 183 {
		iso := countryCodeMap[stats[0].Country]
		flag := countryToFlag(iso)
		builder.WriteString(fmt.Sprintf("%s %s: %d дней\n", flag, stats[0].Country, stats[0].Days))
	} else {
		builder.WriteString(fmt.Sprintf("Нет страны с >=183 днями. Больше всего в: %s (%d дней)", stats[0].Country, stats[0].Days))
	}
	return builder.String()
}

func handleJSONInput(msg *tgbotapi.Message, s *Session, bot *tgbotapi.BotAPI) {
	backupSession(s)
	err := json.Unmarshal([]byte(msg.Text), &s.Data)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Ошибка в формате JSON"))
		return
	}
	if s.Data.Current == "" {
		s.Data.Current = time.Now().Format("02.01.2006")
	}
	saveSession(s)
	report := buildReport(s.Data)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, report))
}

func loadUserData(s *Session) {
	path := fmt.Sprintf("%s/data.json", s.HistoryDir)
	b, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(b, &s.Data)
	}
}

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
			case "show_report":
				report := buildReport(s.Data)
				bot.Send(tgbotapi.NewMessage(chatID, report))
			case "add_period":
				s.PendingAction = "awaiting_add_in"
				saveSession(s)
				bot.Send(tgbotapi.NewMessage(chatID, "📅 Введите дату въезда (ДД.ММ.ГГГГ):"))

			case "delete_period":
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
			case "edit_in":
				s.PendingAction = "awaiting_new_in"
				saveSession(s)
				bot.Send(tgbotapi.NewMessage(chatID, "✏️ Введите новую дату въезда (ДД.ММ.ГГГГ):"))
			case "edit_out":
				s.PendingAction = "awaiting_new_out"
				saveSession(s)
				bot.Send(tgbotapi.NewMessage(chatID, "✏️ Введите новую дату выезда (ДД.ММ.ГГГГ):"))
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

			if s.PendingAction == "awaiting_add_in" {
				date, err := parseDate(msg.Text)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Неверный формат даты."))
					return
				}
				s.Temp = append(s.Temp[:0], Period{In: date.Format("02.01.2006")})
				s.PendingAction = "awaiting_add_out"
				saveSession(s)
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📆 Введите дату выезда (ДД.ММ.ГГГГ):"))
				return
			}

			if s.PendingAction == "awaiting_add_out" {
				date, err := parseDate(msg.Text)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Неверный формат даты."))
					return
				}
				s.Temp[0].Out = date.Format("02.01.2006")
				s.PendingAction = "awaiting_add_country"
				saveSession(s)
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🌍 Укажите название страны:"))
				return
			}

			if s.PendingAction == "awaiting_add_country" {
				s.Temp[0].Country = strings.TrimSpace(msg.Text)
				s.Data.Periods = append(s.Data.Periods, s.Temp[0])
				s.Temp = nil
				s.PendingAction = ""
				saveSession(s)
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Период успешно добавлен."))
				return
			}

			if s.PendingAction == "awaiting_delete_index" {
				index, err := strconv.Atoi(strings.TrimSpace(msg.Text))
				if err != nil || index < 1 || index > len(s.Data.Periods) {
					bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Введите корректный номер периода."))
					return
				}
				s.Data.Periods = append(s.Data.Periods[:index-1], s.Data.Periods[index:]...)
				s.PendingAction = ""
				saveSession(s)
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🗑 Период удалён."))
				return
			}

			// ✅ 1. Ожидание даты
			if s.PendingAction == "awaiting_date" {
				handleDateInput(msg, s, bot)
				continue
			}

			// ✅ 2. Загрузка JSON-файла
			if msg.Document != nil && s.Data.Current == "upload_pending" {
				handleInputFile(msg, s, bot)
				continue
			}

			// ✅ 3. Ожидаем номер редактируемого периода
			if s.PendingAction == "awaiting_edit_index" {
				index, err := strconv.Atoi(strings.TrimSpace(text))
				if err != nil || index < 1 || index > len(s.Data.Periods) {
					bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Введите корректный номер периода."))
					continue
				}
				s.EditingIndex = index - 1
				s.PendingAction = "awaiting_edit_field"
				saveSession(s)

				buttons := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📅 Изменить дату въезда (in)", "edit_in")),
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📆 Изменить дату выезда (out)", "edit_out")),
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🌍 Изменить страну", "edit_country")),
				)
				msg := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Период %d выбран. Что изменить?", index))
				msg.ReplyMarkup = buttons
				bot.Send(msg)

				continue
			}

			// ✅ 4. Изменяем дату in
			if s.PendingAction == "awaiting_new_in" {
				newDate, err := parseDate(text)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Неверный формат даты."))
					continue
				}
				s.Data.Periods[s.EditingIndex].In = newDate.Format("02.01.2006")
				s.PendingAction = ""
				saveSession(s)
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Дата въезда обновлена."))
				continue
			}

			// ✅ 5. Изменяем дату out
			if s.PendingAction == "awaiting_new_out" {
				newDate, err := parseDate(text)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Неверный формат даты."))
					continue
				}
				s.Data.Periods[s.EditingIndex].Out = newDate.Format("02.01.2006")
				s.PendingAction = ""
				saveSession(s)
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Дата выезда обновлена."))
				continue
			}

			// ✅ 6. Изменяем страну
			if s.PendingAction == "awaiting_new_country" {
				s.Data.Periods[s.EditingIndex].Country = strings.TrimSpace(text)
				s.PendingAction = ""
				saveSession(s)
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Страна обновлена."))
				continue
			}

			// ✅ 7. Команды
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

func handleStartCommand(s *Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, "🔘 Выберите действие:")
	reply.ReplyMarkup = buildMainMenu(s)
	bot.Send(reply)
}

func handleHelpCommand(msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	helpText := `ℹ️ Этот бот помогает определить налоговое резидентство на основе загруженных периодов пребывания в разных странах.

📎 С чего начать?
1. Сформируйте JSON-файл со списком ваших поездок (формат пример — по кнопке "Загрузить файл").
2. Отправьте файл через команду /upload_report или с помощью кнопки 📎.
3. Бот рассчитает, в какой стране вы провели больше всего времени за последний год.

📅 Как задать дату расчета?
— Выберите "📅 Задать дату" и укажите дату, на которую хотите сделать расчет (например: 15.04.2025).

📊 Что покажет отчет?
— Страну, в которой вы провели больше всего дней.
— Если есть страна с 183+ днями — вы налоговый резидент этой страны.

🔁 Другие функции:
— /reset: сбросить все данные
— /periods: показать список загруженных периодов

💬 Используйте /start для возврата в главное меню.`

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, helpText))
}

func handleResetCommand(s *Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	if isEmpty(s) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📭 У вас пока нет сохранённых периодов."))
		return
	}
	s.Data = Data{}
	s.Backup = Data{}
	_ = os.Remove(fmt.Sprintf("%s/data.json", s.HistoryDir))
	saveSession(s)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Данные сброшены."))
}

func handleSetDateCommand(s *Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	if isEmpty(s) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📭 У вас пока нет сохранённых периодов."))
		return
	}
	s.PendingAction = "awaiting_date"
	saveSession(s)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📅 Введите дату в формате ДД.ММ.ГГГГ"))
}

func handleUploadCommand(s *Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	s.Data.Current = "upload_pending"
	saveSession(s)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📎 Пожалуйста, отправьте JSON-файл как документ."))
}

func handlePeriodsCommand(s *Session, msg *tgbotapi.Message, bot *tgbotapi.BotAPI) {
	if isEmpty(s) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "📭 У вас пока нет сохранённых периодов."))
		return
	}
	msgText := buildPeriodsList(s)
	newMsg := tgbotapi.NewMessage(msg.Chat.ID, msgText)
	newMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✏️ Отредактировать период", "edit_period")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("➕ Добавить период", "add_period")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить период", "delete_period")),
	)
	bot.Send(newMsg)
}

func handleDateInput(msg *tgbotapi.Message, s *Session, bot *tgbotapi.BotAPI) {
	text := msg.Text
	parsed, err := parseDate(text)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Неверный формат даты. Используйте формат: ДД.ММ.ГГГГ"))
		return
	}
	s.Data.Current = parsed.Format("02.01.2006")
	s.PendingAction = ""
	saveSession(s)

	report := buildReport(s.Data)
	response := fmt.Sprintf("✅ Дата расчета установлена: %s\n\n%s", s.Data.Current, report)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, response))
}

func getSession(userID int64) *Session {
	s, ok := sessions[userID]
	if !ok {
		s = &Session{UserID: userID}
		s.HistoryDir = ensureDirs(userID)
		loadUserData(s)
		sessions[userID] = s
	}
	return s
}

func isEmpty(s *Session) bool {
	return len(s.Data.Periods) == 0
}

func buildMainMenu(s *Session) tgbotapi.InlineKeyboardMarkup {
	var buttons [][]tgbotapi.InlineKeyboardButton

	if isEmpty(s) {
		buttons = [][]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📎 Загрузить файл", "upload_file")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("ℹ️ Помощь", "help")),
		}
	} else {
		buttons = [][]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📋 Показать текущие данные", "periods")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📊 Отчёт", "show_report")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📅 Отчёт на заданную дату", "set_date")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📎 Загрузить новый файл", "upload_report")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🗑 Сбросить", "reset")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("ℹ️ Помощь", "help")),
		}
	}

	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

func buildPeriodsList(s *Session) string {
	builder := strings.Builder{}
	builder.WriteString("📋 Список периодов:\n\n")
	for i, p := range s.Data.Periods {
		in := p.In
		if in == "" {
			in = "—"
		}
		out := p.Out
		if out == "" {
			out = "по " + s.Data.Current
		}
		builder.WriteString(fmt.Sprintf("%d. %s — %s (%s)\n", i+1, in, out, p.Country))
	}
	return builder.String()
}

func handleInputFile(msg *tgbotapi.Message, s *Session, bot *tgbotapi.BotAPI) {
	fileID := msg.Document.FileID
	file, _ := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	url := file.Link(bot.Token)
	resp, err := http.Get(url)

	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⛔ Не удалось загрузить файл"))
		return
	}

	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	msg.Text = string(body)
	s.Data.Current = "" // сбрасываем флаг после загрузки
	handleJSONInput(msg, s, bot)
}
