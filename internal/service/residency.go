package service

import (
	"telegram-tax-bot/internal/model"
	"telegram-tax-bot/internal/storage"
)

// Storage is the minimal persistence interface the service depends on.
type Storage interface {
	Load(int64) (model.UserData, error)
	Save(model.UserData) error
}

type SessionStorate struct {
	st Storage
}

func NewSessionStorage(st Storage) *SessionStorate {
	return &SessionStorate{st: st}
}

// --- Persistence helpers --------------------------------------------------

func (ss *SessionStorate) LoadUser(chatID int64) (model.UserData, error) {
	return ss.st.Load(chatID)
}

func (ss *SessionStorate) SaveUser(data model.UserData) error {
	return ss.st.Save(data)
}

var sessions = map[int64]*model.Session{}

func GetSession(userID int64) *model.Session {
	s, ok := sessions[userID]

	if !ok {
		s = &model.Session{UserID: userID}
		s.HistoryDir = storage.EnsureDirs(userID)
		s.LoadUserData()
		sessions[userID] = s
	}

	return s
}
