package model

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"telegram-tax-bot/internal/utils"
)

type Session struct {
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
	b, err := os.ReadFile(path)

	if err == nil {
		var restored Session
		if err := json.Unmarshal(b, &restored); err == nil {
			*s = restored
		}
	}
}

func (s *Session) IsEmpty() bool {
	return len(s.Data.Periods) == 0
}

func (s *Session) BuildPeriodsList() string {
	builder := strings.Builder{}
	builder.WriteString("📋 Список периодов:\n\n")
	for i, p := range s.Data.Periods {
		in := p.In
		if in == "" {
			in = "—"
		}
		out := p.Out
		if out == "" {
			out = "по " + s.Data.Current
		}
		flag := ""
		if p.Country == "unknown" {
			flag = "🕳 "
		} else if code, ok := utils.CountryCodeMap[p.Country]; ok {
			flag = utils.CountryToFlag(code) + " "
		}
		builder.WriteString(fmt.Sprintf("%d. %s%s (%s — %s)\n", i+1, flag, p.Country, in, out))
	}
	return builder.String()
}
