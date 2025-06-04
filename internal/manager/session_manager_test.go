package manager

import (
	"os"
	"path/filepath"
	"testing"

	"telegram-tax-bot/internal/config"
)

func TestGetSession(t *testing.T) {
	defer os.RemoveAll(config.DataDir)
	s1 := GetSession(5)
	if s1 == nil {
		t.Fatal("expected session")
	}
	s2 := GetSession(5)
	if s1 != s2 {
		t.Fatal("session not cached")
	}
	expected := filepath.Join(config.DataDir, "5")
	if s1.HistoryDir != expected {
		t.Fatalf("dir mismatch: %s", s1.HistoryDir)
	}
}
