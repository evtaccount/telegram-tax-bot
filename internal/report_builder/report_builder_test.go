package reportbuilder

import (
	"strings"
	"testing"

	"telegram-tax-bot/internal/model"
)

func TestBuildReportSimple(t *testing.T) {
	data := model.Data{
		Current: "31.12.2023",
		Periods: []model.Period{{In: "01.01.2023", Out: "31.12.2023", Country: "Россия"}},
	}
	got := BuildReport(data)

	checks := []string{
		"📊 Отчёт о налоговом резидентстве",
		"01.01.2023 — 31.12.2023 (365 дней)",
		"🇷🇺 Россия — 365 дней (100.0%)",
		"✅ Налоговый резидент: 🇷🇺 Россия (365 дней)",
		"Страна с наибольшим пребыванием (≥183 дней).",
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Fatalf("report missing %q:\n%s", check, got)
		}
	}
}

func TestBuildReportMultipleCountries(t *testing.T) {
	data := model.Data{
		Current: "31.12.2023",
		Periods: []model.Period{
			{In: "01.01.2023", Out: "01.07.2023", Country: "Россия"},
			{In: "02.07.2023", Out: "30.12.2023", Country: "Казахстан"},
		},
	}
	got := BuildReport(data)
	if !strings.Contains(got, "⚠️ Нет страны с ≥183 днями") {
		t.Fatalf("expected no-residency warning:\n%s", got)
	}
	if !strings.Contains(got, "•") {
		t.Fatalf("expected period breakdown:\n%s", got)
	}
}

func TestBuildReportPluralDays(t *testing.T) {
	data := model.Data{
		Current: "02.01.2023",
		Periods: []model.Period{{In: "01.01.2023", Out: "02.01.2023", Country: "Россия"}},
	}
	got := BuildReport(data)
	if !strings.Contains(got, "2 дня") {
		t.Fatalf("expected '2 дня' plural form:\n%s", got)
	}
}
