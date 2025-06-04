package model

import (
	"os"
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
	expected := "📋 Список периодов:\n\n1. 01.01.2024 — по 01.01.2024 (Россия)\n"
	if list := s.BuildPeriodsList(); list != expected {
		t.Fatalf("unexpected list: %s", list)
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
