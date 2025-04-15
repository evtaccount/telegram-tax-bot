// main.go
// Telegram бот для расчета налогового резидентства
// Поддержка: /start, /help, /reset, /undo, добавление/редактирование периодов, Docker secret/env, история, inline-кнопки

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
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
	UserID     int64
	Data       Data
	Backup     Data
	HistoryDir string
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
	// 1. Пробуем считать из Docker Secret (если запускается в Swarm или через файл)
	if data, err := os.ReadFile("/run/secrets/telegram_bot_token"); err == nil {
		token := strings.TrimSpace(string(data))
		if token != "" {
			log.Println("✅ Токен получен из Docker Secret (/run/secrets/telegram_bot_token)")
			return token
		}
	}

	// 2. Пробуем из переменной окружения
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token != "" {
		log.Println("✅ Токен получен из переменной окружения (TELEGRAM_BOT_TOKEN)")
		return token
	}

	// 3. Если ничего не найдено — критическая ошибка
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

func undoSession(s *Session) string {
	if len(s.Backup.Periods) == 0 {
		return "Нет изменений для отката."
	}
	s.Data = s.Backup
	saveSession(s)
	return "Последнее изменение отменено."
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

func handleSetDateCommand(message *tgbotapi.Message, s *Session, bot *tgbotapi.BotAPI) {
	text := strings.TrimSpace(strings.TrimPrefix(message.Text, "/setdate"))
	if text == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Укажите дату в формате ДД.ММ.ГГГГ, например: /setdate 15.04.2025")
		bot.Send(msg)
		return
	}
	parsed, err := parseDate(text)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, "⛔ Неверный формат даты. Используйте формат: ДД.ММ.ГГГГ")
		bot.Send(msg)
		return
	}
	s.Data.Current = parsed.Format("02.01.2006")
	saveSession(s)
	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("✅ Дата расчета установлена: %s", s.Data.Current))
	bot.Send(msg)
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
		if update.Message == nil {
			continue
		}
		msg := update.Message
		userID := msg.From.ID
		s, ok := sessions[userID]
		if !ok {
			s = &Session{UserID: userID}
			s.HistoryDir = ensureDirs(userID)
			loadUserData(s)
			sessions[userID] = s
		}

		text := msg.Text
		switch {
		case strings.HasPrefix(text, "/start"):
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "👋 Привет! Отправьте JSON с периодами или используйте /help"))
		case strings.HasPrefix(text, "/help"):
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "ℹ️ Доступные команды:\n/start — начало\n/help — помощь\n/reset — сброс данных\n/undo — отменить последнее изменение\n/setdate ДД.ММ.ГГГГ — установить дату расчета"))
		case strings.HasPrefix(text, "/reset"):
			s.Data = Data{}
			s.Backup = Data{}
			saveSession(s)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Данные сброшены."))
		case strings.HasPrefix(text, "/undo"):
			response := undoSession(s)
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, response))
		case strings.HasPrefix(text, "/setdate"):
			handleSetDateCommand(msg, s, bot)
		default:
			if strings.HasPrefix(text, "{") {
				handleJSONInput(msg, s, bot)
			} else {
				bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❓ Неизвестная команда. Введите /help для списка."))
			}
		}
	}
}

func loadUserData(s *Session) {
	path := fmt.Sprintf("%s/data.json", s.HistoryDir)
	b, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(b, &s.Data)
	}
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

func sendReport(bot *tgbotapi.BotAPI, chatID int64, report string) {
	msg := tgbotapi.NewMessage(chatID, report)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Повторить расчёт", "repeat"),
			tgbotapi.NewInlineKeyboardButtonData("Сбросить данные", "reset"),
		),
	)
	bot.Send(msg)
}
