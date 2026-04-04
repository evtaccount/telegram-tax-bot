package manager

import (
	"sync"

	storage "telegram-tax-bot/internal/file_storage"
	"telegram-tax-bot/internal/model"
)

var (
	mu       sync.Mutex
	sessions = map[int64]*model.Session{}
)

func GetSession(userID int64) *model.Session {
	mu.Lock()
	defer mu.Unlock()

	s, ok := sessions[userID]

	if !ok {
		s = &model.Session{UserID: userID}
		s.HistoryDir = storage.EnsureDirs(userID)
		s.LoadUserData()
		sessions[userID] = s
	}

	return s
}
