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
