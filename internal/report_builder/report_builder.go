package reportbuilder

import (
	"fmt"
	"sort"
	"strings"
	"telegram-tax-bot/internal/model"
	"telegram-tax-bot/internal/utils"
	"time"
)

type periodSpan struct {
	Country string
	In      time.Time
	Out     time.Time
	Days    int
}

func BuildReport(data model.Data) string {
	calcDate, err := utils.ParseDate(data.Current)
	if err != nil {
		return "⛔ Ошибка: не задана или некорректна дата расчёта. Установите дату через меню."
	}
	oneYearAgo := calcDate.AddDate(-1, 0, 0).AddDate(0, 0, 1)

	countryDays := make(map[string]int)
	countrySpans := make(map[string][]periodSpan)
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
				if gapEnd.After(calcDate) {
					gapEnd = calcDate
				}
				if gapStart.Before(oneYearAgo) {
					gapStart = oneYearAgo
				}
				if !gapStart.After(gapEnd) {
					days := int(gapEnd.Sub(gapStart).Hours()/24) + 1
					countryDays["unknown"] += days
					countrySpans["unknown"] = append(countrySpans["unknown"], periodSpan{
						Country: "unknown", In: gapStart, Out: gapEnd, Days: days,
					})
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
		countrySpans[period.Country] = append(countrySpans[period.Country], periodSpan{
			Country: period.Country, In: inDate, Out: outDate, Days: days,
		})
	}

	if len(countryDays) == 0 {
		return "Нет данных для анализа за указанный период."
	}

	type stat struct {
		Country string
		Days    int
	}
	var stats []stat
	totalDays := 0
	for c, d := range countryDays {
		stats = append(stats, stat{c, d})
		totalDays += d
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Days > stats[j].Days
	})

	builder := strings.Builder{}
	fmt.Fprintf(&builder, "📊 Отчёт о налоговом резидентстве\nПериод анализа: %s — %s (%s)\n\n",
		utils.FormatDate(oneYearAgo), utils.FormatDate(calcDate), utils.PluralDays(totalDays))

	for _, s := range stats {
		pct := float64(s.Days) / float64(totalDays) * 100
		if s.Country == "unknown" {
			fmt.Fprintf(&builder, "🕳 Не учтено — %s (%.1f%%)\n", utils.PluralDays(s.Days), pct)
		} else {
			iso := utils.CountryCodeMap[s.Country]
			flag := utils.CountryToFlag(iso)
			fmt.Fprintf(&builder, "%s %s — %s (%.1f%%)\n", flag, s.Country, utils.PluralDays(s.Days), pct)
		}
		// Show individual period spans
		spans := countrySpans[s.Country]
		if len(spans) > 1 || (len(spans) == 1 && s.Country != "unknown") {
			for _, sp := range spans {
				fmt.Fprintf(&builder, "   • %s — %s (%s)\n",
					utils.FormatShortDate(sp.In), utils.FormatShortDate(sp.Out), utils.PluralDays(sp.Days))
			}
		}
	}

	builder.WriteString("\n━━━━━━━━━━━━━━━━━━━━\n")
	if stats[0].Country != "unknown" && stats[0].Days >= 183 {
		iso := utils.CountryCodeMap[stats[0].Country]
		flag := utils.CountryToFlag(iso)
		fmt.Fprintf(&builder, "✅ Налоговый резидент: %s %s (%s)\n", flag, stats[0].Country, utils.PluralDays(stats[0].Days))
		builder.WriteString("Страна с наибольшим пребыванием (≥183 дней).\n")
	} else {
		for _, s := range stats {
			if s.Country != "unknown" {
				fmt.Fprintf(&builder, "⚠️ Нет страны с ≥183 днями. Больше всего в: %s (%s)\n", s.Country, utils.PluralDays(s.Days))
				break
			}
		}
	}

	return builder.String()
}
