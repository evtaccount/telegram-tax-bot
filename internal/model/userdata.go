package model

type UserData struct {
	ChatID  int64      `json:"-"`
	Periods []Period   `json:"periods"`
	History [][]Period `json:"history,omitempty"`
}
