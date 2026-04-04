package utils

import "testing"

func TestCountryToFlag(t *testing.T) {
	if CountryToFlag("RU") != "🇷🇺" {
		t.Fatalf("wrong flag")
	}
}

func TestCountryCodeMap(t *testing.T) {
	if CountryCodeMap["Россия"] != "RU" {
		t.Fatalf("wrong code")
	}
}

func TestNormalizeCountry(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"россия", "Россия"},
		{"РОССИЯ", "Россия"},
		{"южная корея", "Южная Корея"},
		{"шри-ланка", "Шри-Ланка"},
		{"  германия  ", "Германия"},
		{"", ""},
	}
	for _, tc := range cases {
		got := NormalizeCountry(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeCountry(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
