package utils

import (
	"strings"
	"unicode"
)

// NormalizeCountry applies Title Case to a country name.
func NormalizeCountry(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}
	// Title case: first letter of each word uppercase, rest lowercase.
	runes := []rune(strings.ToLower(name))
	capitalize := true
	for i, r := range runes {
		if capitalize && unicode.IsLetter(r) {
			runes[i] = unicode.ToUpper(r)
			capitalize = false
		}
		if unicode.IsSpace(r) || r == '-' {
			capitalize = true
		}
	}
	return string(runes)
}

func CountryToFlag(isoCode string) string {
	isoCode = strings.ToUpper(isoCode)
	if len(isoCode) != 2 {
		return ""
	}
	return string(rune(0x1F1E6+int(isoCode[0]-'A'))) + string(rune(0x1F1E6+int(isoCode[1]-'A')))
}

var CountryCodeMap = map[string]string{
	"Австралия":            "AU",
	"Австрия":              "AT",
	"Азербайджан":          "AZ",
	"Албания":              "AL",
	"Алжир":                "DZ",
	"Американское Самоа":   "AS",
	"Ангола":               "AO",
	"Андорра":              "AD",
	"Аргентина":            "AR",
	"Армения":              "AM",
	"Афганистан":           "AF",
	"Бангладеш":            "BD",
	"Беларусь":             "BY",
	"Бельгия":              "BE",
	"Болгария":             "BG",
	"Боливия":              "BO",
	"Босния и Герцеговина": "BA",
	"Бразилия":             "BR",
	"Великобритания":       "GB",
	"Венгрия":              "HU",
	"Венесуэла":            "VE",
	"Вьетнам":              "VN",
	"Германия":             "DE",
	"Гонконг":              "HK",
	"Греция":               "GR",
	"Грузия":               "GE",
	"Дания":                "DK",
	"Египет":               "EG",
	"Израиль":              "IL",
	"Индия":                "IN",
	"Индонезия":            "ID",
	"Иордания":             "JO",
	"Ирак":                 "IQ",
	"Иран":                 "IR",
	"Ирландия":             "IE",
	"Исландия":             "IS",
	"Испания":              "ES",
	"Италия":               "IT",
	"Казахстан":            "KZ",
	"Камбоджа":             "KH",
	"Канада":               "CA",
	"Катар":                "QA",
	"Кения":                "KE",
	"Кипр":                 "CY",
	"Китай":                "CN",
	"Колумбия":             "CO",
	"Коста-Рика":           "CR",
	"Куба":                 "CU",
	"Кыргызстан":           "KG",
	"Латвия":               "LV",
	"Ливан":                "LB",
	"Литва":                "LT",
	"Люксембург":           "LU",
	"Малайзия":             "MY",
	"Мальта":               "MT",
	"Марокко":              "MA",
	"Мексика":              "MX",
	"Молдова":              "MD",
	"Монголия":             "MN",
	"Нидерланды":           "NL",
	"Новая Зеландия":       "NZ",
	"Норвегия":             "NO",
	"ОАЭ":                  "AE",
	"Пакистан":             "PK",
	"Панама":               "PA",
	"Перу":                 "PE",
	"Польша":               "PL",
	"Португалия":           "PT",
	"Россия":               "RU",
	"Румыния":              "RO",
	"Саудовская Аравия":    "SA",
	"Сербия":               "RS",
	"Сингапур":             "SG",
	"Словакия":             "SK",
	"Словения":             "SI",
	"США":                  "US",
	"Таджикистан":          "TJ",
	"Таиланд":              "TH",
	"Тунис":                "TN",
	"Туркменистан":         "TM",
	"Турция":               "TR",
	"Узбекистан":           "UZ",
	"Украина":              "UA",
	"Филиппины":            "PH",
	"Финляндия":            "FI",
	"Франция":              "FR",
	"Хорватия":             "HR",
	"Чехия":                "CZ",
	"Чили":                 "CL",
	"Швейцария":            "CH",
	"Швеция":               "SE",
	"Шри-Ланка":            "LK",
	"Эстония":              "EE",
	"ЮАР":                  "ZA",
	"Южная Корея":          "KR",
	"Япония":               "JP",
}
