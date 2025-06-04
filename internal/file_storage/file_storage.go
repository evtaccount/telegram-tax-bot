package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"telegram-tax-bot/internal/config"
	"telegram-tax-bot/internal/model"
)

type FileStorage struct {
	dir string
}

func NewFileStorage(dir string) *FileStorage {
	_ = os.MkdirAll(dir, 0o755)
	return &FileStorage{dir: dir}
}

func (fs *FileStorage) file(chatID int64) string {
	return filepath.Join(fs.dir, fmt.Sprintf("%d.json", chatID))
}

func (fs *FileStorage) Load(chatID int64) (model.UserData, error) {
	var data model.UserData
	f, err := os.Open(fs.file(chatID))
	if err != nil {
		if os.IsNotExist(err) {
			return model.UserData{ChatID: chatID}, nil
		}
		return data, err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return data, err
	}
	data.ChatID = chatID
	return data, nil
}

func (fs *FileStorage) Save(data model.UserData) error {
	f, err := os.Create(fs.file(data.ChatID))
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func EnsureDirs(userID int64) string {
	path := fmt.Sprintf("%s/%d", config.DataDir, userID)
	os.MkdirAll(path, 0755)
	return path
}
