package service

import (
	"testing"

	"telegram-tax-bot/internal/model"
)

type memStorage struct {
	m map[int64]model.UserData
}

func (ms *memStorage) Load(id int64) (model.UserData, error) {
	if d, ok := ms.m[id]; ok {
		return d, nil
	}
	return model.UserData{ChatID: id}, nil
}

func (ms *memStorage) Save(d model.UserData) error {
	ms.m[d.ChatID] = d
	return nil
}

func TestUserStorage(t *testing.T) {
	ms := &memStorage{m: make(map[int64]model.UserData)}
	us := NewUserStorage(ms)
	data := model.UserData{ChatID: 1, Periods: []model.Period{{Country: "RU"}}}
	if err := us.SaveUser(data); err != nil {
		t.Fatalf("save error: %v", err)
	}
	loaded, err := us.LoadUser(1)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if len(loaded.Periods) != 1 || loaded.ChatID != 1 {
		t.Fatalf("loaded wrong data: %+v", loaded)
	}
}
