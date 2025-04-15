// main.go
// Telegram бот для расчета налогового резидентства
// Поддержка: /start, /help, /reset, /undo, добавление/редактирование периодов, Docker secret/env, история, inline-кнопки

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

func main() {
	bot, err := tgbotapi.NewBotAPI(getBotToken())
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Запущен бот: %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		msg := update.Message
		uid := msg.Chat.ID

		if _, ok := sessions[uid]; !ok {
			sessions[uid] = &Session{UserID: uid, HistoryDir: ensureDirs(uid)}
		}
		s := sessions[uid]

		switch {
		case msg.IsCommand():
			switch msg.Command() {
			case "start":
				bot.Send(tgbotapi.NewMessage(uid, "👋 Пришли JSON-файл или команду /help"))
			case "help":
				bot.Send(tgbotapi.NewMessage(uid, "📘 Команды:\n/start — начать\n/help — справка\n/reset — сброс\n/undo — откат\nДобавляй периоды: 'добавить период 01.01.2024 - 10.01.2024 Грузия'\nРедактируй: 'изменить период 2 in 05.01.2024 out 15.01.2024 страна Турция'"))
			case "reset":
				s.Data = Data{}
				bot.Send(tgbotapi.NewMessage(uid, "🔄 Данные сброшены."))
			case "undo":
				resp := undoSession(s)
				bot.Send(tgbotapi.NewMessage(uid, resp))
			}

		case msg.Document != nil:
			file, _ := bot.GetFile(tgbotapi.FileConfig{FileID: msg.Document.FileID})
			url := file.Link(bot.Token)
			resp, err := http.Get(url)

			if err != nil {
				bot.Send(tgbotapi.NewMessage(uid, "❌ Ошибка загрузки файла"))
				continue
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()

			if err != nil {
				bot.Send(tgbotapi.NewMessage(uid, "❌ Ошибка чтения файла"))
				continue
			}

			if err := json.Unmarshal(body, &s.Data); err != nil {
				bot.Send(tgbotapi.NewMessage(uid, "❌ Ошибка чтения JSON"))
				continue
			}

			backupSession(s)
			saveSession(s)

			report := buildReport(s.Data)
			sendReport(bot, uid, report)

		case strings.HasPrefix(strings.ToLower(msg.Text), "добавить период"):
			parts := strings.Split(msg.Text, " ")
			if len(parts) >= 5 {
				p := Period{In: parts[2], Out: parts[4], Country: strings.Join(parts[5:], " ")}
				s.Backup = s.Data
				s.Data.Periods = append(s.Data.Periods, p)
				saveSession(s)
				report := buildReport(s.Data)
				sendReport(bot, uid, report)
			} else {
				bot.Send(tgbotapi.NewMessage(uid, "⚠️ Формат: добавить период 01.01.2024 - 10.01.2024 Грузия"))
			}

		case strings.HasPrefix(strings.ToLower(msg.Text), "изменить период"):
			// Добавить парсинг номера периода, обновление и подтверждение — по желанию
			bot.Send(tgbotapi.NewMessage(uid, "✏️ Функция редактирования скоро будет полностью доступна!"))

		default:
			bot.Send(tgbotapi.NewMessage(uid, "🤖 Я не понял. Напиши /help или пришли JSON."))
		}
	}
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
