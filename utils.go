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
	b, _ := json.MarshalIndent(s, "", "  ")
	_ = os.WriteFile(fmt.Sprintf("%s/session.json", s.HistoryDir), b, 0644)
	t := time.Now().Format("2006-01-02_15-04-05")
	_ = os.WriteFile(fmt.Sprintf("%s/report_%s.txt", s.HistoryDir, t), []byte(buildReport(s.Data)), 0644)
}

func loadUserData(s *Session) {
	path := fmt.Sprintf("%s/session.json", s.HistoryDir)
	b, err := os.ReadFile(path)

	if err == nil {
		var restored Session
		if err := json.Unmarshal(b, &restored); err == nil {
			*s = restored
		}
	}
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

		// проверка хронологии
		if i > 0 && inDate.Before(previousOutDate) {
			return fmt.Sprintf("Ошибка: периоды не в хронологическом порядке (период %d)", i+1)
		}

		// обработка разрыва между предыдущим и текущим
		if i > 0 && !previousOutDate.IsZero() {
			gapStart := previousOutDate.AddDate(0, 0, 1)
			if gapStart.Before(inDate) {
				gapEnd := inDate.AddDate(0, 0, -1)
				// обрезаем до окна
				if gapEnd.After(calcDate) {
					gapEnd = calcDate
				}
				if gapStart.Before(oneYearAgo) {
					gapStart = oneYearAgo
				}
				if !gapStart.After(gapEnd) {
					days := int(gapEnd.Sub(gapStart).Hours()/24) + 1
					countryDays["unknown"] += days
				}
			}
		}

		previousOutDate = outDate

		// обрезка до окна
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
		if s.Country == "unknown" {
			builder.WriteString(fmt.Sprintf("🕳 Неизвестно где: %d дней\n", s.Days))
			continue
		}
		iso := countryCodeMap[s.Country]
		flag := countryToFlag(iso)
		builder.WriteString(fmt.Sprintf("%s %s: %d дней\n", flag, s.Country, s.Days))
	}

	builder.WriteString("\n")
	if stats[0].Country != "unknown" && stats[0].Days >= 183 {
		iso := countryCodeMap[stats[0].Country]
		flag := countryToFlag(iso)
		builder.WriteString(fmt.Sprintf("✅ Налоговый резидент: %s %s (%d дней)\n", flag, stats[0].Country, stats[0].Days))
	} else {
		for _, s := range stats {
			if s.Country != "unknown" {
				builder.WriteString(fmt.Sprintf("⚠️ Нет страны с >=183 днями. Больше всего в: %s (%d дней)\n", s.Country, s.Days))
				break
			}
		}
	}

	return builder.String()
}

func formatPeriodList(periods []Period, current string) string {
	var builder strings.Builder
	builder.WriteString("📋 Список периодов:\n\n")
	for i, p := range periods {
		in := p.In
		if in == "" {
			in = "—"
		}
		out := p.Out
		if out == "" {
			out = "по " + current
		}
		builder.WriteString(fmt.Sprintf("%d. %s — %s (%s)\n", i+1, in, out, p.Country))
	}
	return builder.String()
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

func buildBackToMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад в меню", "start"),
		),
	)
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
