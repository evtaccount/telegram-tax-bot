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

func TestPluralDays(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{1, "1 день"},
		{2, "2 дня"},
		{3, "3 дня"},
		{4, "4 дня"},
		{5, "5 дней"},
		{10, "10 дней"},
		{11, "11 дней"},
		{12, "12 дней"},
		{21, "21 день"},
		{22, "22 дня"},
		{25, "25 дней"},
		{111, "111 дней"},
		{121, "121 день"},
		{183, "183 дня"},
		{365, "365 дней"},
	}
	for _, tc := range cases {
		got := PluralDays(tc.n)
		if got != tc.want {
			t.Errorf("PluralDays(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

func TestDaysBetween(t *testing.T) {
	from, _ := ParseDate("01.01.2024")
	to, _ := ParseDate("10.01.2024")
	if days := DaysBetween(from, to); days != 10 {
		t.Errorf("DaysBetween = %d, want 10", days)
	}
}

func TestFormatShortDate(t *testing.T) {
	d, _ := ParseDate("15.03.2024")
	if got := FormatShortDate(d); got != "15.03" {
		t.Errorf("FormatShortDate = %q, want %q", got, "15.03")
	}
}
