package manager

import (
	storage "telegram-tax-bot/internal/file_storage"
	"telegram-tax-bot/internal/model"
)

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
