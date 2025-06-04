package reportbuilder

import (
	"fmt"
	"sort"
	"strings"
	"telegram-tax-bot/internal/model"
	"telegram-tax-bot/internal/utils"
	"time"
)

func BuildReport(data model.Data) string {
	calcDate, _ := utils.ParseDate(data.Current)
	oneYearAgo := calcDate.AddDate(-1, 0, 0).AddDate(0, 0, 1)
	countryDays := make(map[string]int)
	var previousOutDate time.Time

	for i, period := range data.Periods {
		var inDate, outDate time.Time
		if period.Out != "" {
			outDate, _ = utils.ParseDate(period.Out)
		} else {
			outDate = calcDate
		}

		if i == 0 && period.In == "" {
			inDate = oneYearAgo
			if outDate.Before(oneYearAgo) {
				continue
			}
		} else {
			inDate, _ = utils.ParseDate(period.In)
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
	builder.WriteString(fmt.Sprintf("Анализ за период: %s — %s\n\n", utils.FormatDate(oneYearAgo), utils.FormatDate(calcDate)))
	for _, s := range stats {
		if s.Country == "unknown" {
			builder.WriteString(fmt.Sprintf("🕳 Неизвестно где: %d дней\n", s.Days))
			continue
		}
		iso := utils.CountryCodeMap[s.Country]
		flag := utils.CountryToFlag(iso)
		builder.WriteString(fmt.Sprintf("%s %s: %d дней\n", flag, s.Country, s.Days))
	}

	builder.WriteString("\n")
	if stats[0].Country != "unknown" && stats[0].Days >= 183 {
		iso := utils.CountryCodeMap[stats[0].Country]
		flag := utils.CountryToFlag(iso)
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
