package storage

import (
	"os"
	"testing"

	"telegram-tax-bot/internal/model"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStorage(dir)
	data := model.UserData{ChatID: 42, Periods: []model.Period{{Country: "RU"}}}
	if err := fs.Save(data); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := fs.Load(42)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if len(loaded.Periods) != 1 || loaded.ChatID != 42 {
		t.Fatalf("loaded data mismatch: %+v", loaded)
	}

	if _, err := os.Stat(fs.file(42)); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func TestLoadMissing(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStorage(dir)
	loaded, err := fs.Load(99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.ChatID != 99 || len(loaded.Periods) != 0 {
		t.Fatalf("unexpected data: %+v", loaded)
	}
}
