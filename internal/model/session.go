package model

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"telegram-tax-bot/internal/utils"
	"time"
)

type Session struct {
	Mu            sync.Mutex `json:"-"`
	UserID        int64
	Data          Data
	Backup        Data
	HistoryDir    string
	Temp          []Period
	EditingIndex  int
	PendingAction string
	TempEditedIn  string
	TempEditedOut string
}

func (s *Session) BackupSession() {
	bytes, err := json.MarshalIndent(s.Data, "", "  ")
	if err != nil {
		log.Printf("backup marshal error (user %d): %v", s.UserID, err)
		return
	}
	if err := os.WriteFile(fmt.Sprintf("%s/backup.json", s.HistoryDir), bytes, 0644); err != nil {
		log.Printf("backup write error (user %d): %v", s.UserID, err)
	}
	s.Backup = s.Data
}

func (s *Session) SaveSession() {
	bytes, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Printf("session marshal error (user %d): %v", s.UserID, err)
		return
	}
	if err := os.WriteFile(fmt.Sprintf("%s/session.json", s.HistoryDir), bytes, 0644); err != nil {
		log.Printf("session write error (user %d): %v", s.UserID, err)
	}
}

func (s *Session) LoadUserData() {
	path := fmt.Sprintf("%s/session.json", s.HistoryDir)
	raw, err := os.ReadFile(path)

	if err == nil {
		var restored Session
		if err := json.Unmarshal(raw, &restored); err == nil {
			s.UserID = restored.UserID
			s.Data = restored.Data
			s.Backup = restored.Backup
			s.HistoryDir = restored.HistoryDir
			s.Temp = restored.Temp
			s.EditingIndex = restored.EditingIndex
			s.PendingAction = restored.PendingAction
			s.TempEditedIn = restored.TempEditedIn
			s.TempEditedOut = restored.TempEditedOut
		}
	}
}

func (s *Session) IsEmpty() bool {
	return len(s.Data.Periods) == 0
}

// UsedCountries returns a deduplicated list of country names from periods (excluding "unknown").
func (s *Session) UsedCountries() []string {
	seen := make(map[string]bool)
	var result []string
	for _, p := range s.Data.Periods {
		if p.Country != "" && p.Country != "unknown" && !seen[p.Country] {
			seen[p.Country] = true
			result = append(result, p.Country)
		}
	}
	return result
}

func (s *Session) BuildPeriodsList() string {
	builder := strings.Builder{}
	builder.WriteString("📋 Ваши периоды:\n\n")
	for i, p := range s.Data.Periods {
		in := p.In
		if in == "" {
			in = "—"
		}
		out := p.Out
		openEnded := false
		if out == "" {
			out = "н.в."
			openEnded = true
		}

		countryName := p.Country
		if countryName == "unknown" {
			countryName = "Не указано"
		}

		flag := ""
		if p.Country == "unknown" {
			flag = "🕳 "
		} else if code, ok := utils.CountryCodeMap[p.Country]; ok {
			flag = utils.CountryToFlag(code) + " "
		}

		// Calculate days in period
		daysSuffix := ""
		inDate, errIn := utils.ParseDate(p.In)
		var outDate time.Time
		var errOut error
		if openEnded && s.Data.Current != "" {
			outDate, errOut = utils.ParseDate(s.Data.Current)
		} else {
			outDate, errOut = utils.ParseDate(p.Out)
		}
		if errIn == nil && errOut == nil && !outDate.Before(inDate) {
			days := utils.DaysBetween(inDate, outDate)
			daysSuffix = fmt.Sprintf(" (%s)", utils.PluralDays(days))
		}

		builder.WriteString(fmt.Sprintf("%d. %s%s: %s — %s%s\n", i+1, flag, countryName, in, out, daysSuffix))

		// Show gap to next period
		if i < len(s.Data.Periods)-1 {
			next := s.Data.Periods[i+1]
			if p.Out != "" && next.In != "" {
				curOut, err1 := utils.ParseDate(p.Out)
				nextIn, err2 := utils.ParseDate(next.In)
				if err1 == nil && err2 == nil {
					gapStart := curOut.AddDate(0, 0, 1)
					if gapStart.Before(nextIn) {
						gapDays := utils.DaysBetween(gapStart, nextIn.AddDate(0, 0, -1))
						builder.WriteString(fmt.Sprintf("   ⚠️ Разрыв %s (%s — %s)\n",
							utils.PluralDays(gapDays),
							utils.FormatShortDate(gapStart),
							utils.FormatShortDate(nextIn.AddDate(0, 0, -1))))
					}
				}
			}
		}
	}
	return builder.String()
}
