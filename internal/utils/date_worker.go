package utils

import "time"

func ParseDate(dateStr string) (time.Time, error) {
	return time.Parse("02.01.2006", dateStr)
}

func FormatDate(d time.Time) string {
	return d.Format("02.01.2006")
}
