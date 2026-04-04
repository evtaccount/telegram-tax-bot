package utils

import (
	"fmt"
	"time"
)

func ParseDate(dateStr string) (time.Time, error) {
	return time.Parse("02.01.2006", dateStr)
}

func FormatDate(d time.Time) string {
	return d.Format("02.01.2006")
}

// FormatShortDate returns date as "02.01" (day.month without year).
func FormatShortDate(d time.Time) string {
	return d.Format("02.01")
}

// PluralDays returns the number with correct Russian plural form of "день".
func PluralDays(n int) string {
	abs := n
	if abs < 0 {
		abs = -abs
	}
	mod10 := abs % 10
	mod100 := abs % 100
	switch {
	case mod100 >= 11 && mod100 <= 19:
		return fmt.Sprintf("%d дней", n)
	case mod10 == 1:
		return fmt.Sprintf("%d день", n)
	case mod10 >= 2 && mod10 <= 4:
		return fmt.Sprintf("%d дня", n)
	default:
		return fmt.Sprintf("%d дней", n)
	}
}

// DaysBetween returns the number of days between two dates (inclusive on both ends).
func DaysBetween(from, to time.Time) int {
	return int(to.Sub(from).Hours()/24) + 1
}
