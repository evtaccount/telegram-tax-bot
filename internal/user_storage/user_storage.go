package service

import (
	"telegram-tax-bot/internal/model"
)

// Storage is the minimal persistence interface the service depends on.
type Storage interface {
	Load(int64) (model.UserData, error)
	Save(model.UserData) error
}

type UserStorage struct {
	st Storage
}

func NewUserStorage(st Storage) *UserStorage {
	return &UserStorage{st: st}
}

func (ss *UserStorage) LoadUser(chatID int64) (model.UserData, error) {
	return ss.st.Load(chatID)
}

func (ss *UserStorage) SaveUser(data model.UserData) error {
	return ss.st.Save(data)
}
