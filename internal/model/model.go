
package model

import "time"

type Period struct {
    Country string     `json:"country"`
    In      *time.Time `json:"in,omitempty"`
    Out     time.Time  `json:"out"`
}

type UserData struct {
    ChatID  int64     `json:"-"`
    Periods []Period  `json:"periods"`
    History [][]Period `json:"history,omitempty"`
}
