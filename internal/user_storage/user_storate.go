package service

import (
	"telegram-tax-bot/internal/model"
)

// Storage is the minimal persistence interface the service depends on.
type Storage interface {
	Load(int64) (model.UserData, error)
	Save(model.UserData) error
}

type UserStorate struct {
	st Storage
}

func NewUserStorage(st Storage) *UserStorate {
	return &UserStorate{st: st}
}

func (ss *UserStorate) LoadUser(chatID int64) (model.UserData, error) {
	return ss.st.Load(chatID)
}

func (ss *UserStorate) SaveUser(data model.UserData) error {
	return ss.st.Save(data)
}
