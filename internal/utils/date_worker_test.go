package utils

import "testing"

func TestParseAndFormatDate(t *testing.T) {
	d, err := ParseDate("01.02.2023")
	if err != nil {
		t.Fatalf("ParseDate returned error: %v", err)
	}
	if FormatDate(d) != "01.02.2023" {
		t.Fatalf("expected 01.02.2023, got %s", FormatDate(d))
	}
}

func TestParseDateInvalid(t *testing.T) {
	if _, err := ParseDate("invalid"); err == nil {
		t.Fatal("expected error for invalid date")
	}
}
