package model

import (
	"os"
	"strings"
	"testing"
)

func TestIsEmpty(t *testing.T) {
	s := &Session{}
	if !s.IsEmpty() {
		t.Fatal("expected empty session")
	}
	s.Data.Periods = []Period{{Country: "RU"}}
	if s.IsEmpty() {
		t.Fatal("expected non-empty session")
	}
}

func TestBuildPeriodsList(t *testing.T) {
	s := &Session{Data: Data{Current: "01.01.2024", Periods: []Period{{In: "01.01.2024", Country: "Россия"}}}}
	expected := "📋 Ваши периоды:\n\n1. 🇷🇺 Россия: 01.01.2024 — н.в. (1 день)\n"
	if list := s.BuildPeriodsList(); list != expected {
		t.Fatalf("unexpected list: %s", list)
	}
}

func TestBuildPeriodsListWithGap(t *testing.T) {
	s := &Session{Data: Data{Current: "01.03.2024", Periods: []Period{
		{In: "01.01.2024", Out: "10.01.2024", Country: "Россия"},
		{In: "15.01.2024", Out: "01.03.2024", Country: "Казахстан"},
	}}}
	list := s.BuildPeriodsList()
	if !strings.Contains(list, "⚠️ Разрыв") {
		t.Fatalf("expected gap indicator in list: %s", list)
	}
	if !strings.Contains(list, "10 дней") {
		t.Fatalf("expected day count in list: %s", list)
	}
}

func TestUsedCountries(t *testing.T) {
	s := &Session{Data: Data{Periods: []Period{
		{Country: "Россия"},
		{Country: "unknown"},
		{Country: "Россия"},
		{Country: "Казахстан"},
	}}}
	countries := s.UsedCountries()
	if len(countries) != 2 {
		t.Fatalf("expected 2 countries, got %d: %v", len(countries), countries)
	}
}

func TestSaveAndLoadSession(t *testing.T) {
	dir := t.TempDir()
	s := &Session{UserID: 1, HistoryDir: dir, Data: Data{Current: "01.01.2024", Periods: []Period{{In: "01.01.2024", Out: "02.01.2024", Country: "RU"}}}}
	s.SaveSession()
	var restored Session
	restored.HistoryDir = dir
	restored.LoadUserData()
	if len(restored.Data.Periods) != 1 || restored.Data.Current != "01.01.2024" {
		t.Fatal("session not restored correctly")
	}
	if _, err := os.Stat(dir + "/session.json"); err != nil {
		t.Fatalf("session file not found: %v", err)
	}
}
