package reportbuilder

import (
	"testing"

	"telegram-tax-bot/internal/model"
)

func TestBuildReportSimple(t *testing.T) {
	data := model.Data{
		Current: "31.12.2023",
		Periods: []model.Period{{In: "01.01.2023", Out: "31.12.2023", Country: "Россия"}},
	}
	got := BuildReport(data)
	expected := "Анализ за период: 01.01.2023 — 31.12.2023\n\n🇷🇺 Россия: 365 дней\n\n✅ Налоговый резидент: 🇷🇺 Россия (365 дней)\n"
	if got != expected {
		t.Fatalf("unexpected report:\n%s", got)
	}
}

func TestCountryToFlag(t *testing.T) {
	if countryToFlag("RU") != "🇷🇺" {
		t.Fatalf("wrong flag")
	}
}
