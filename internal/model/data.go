package model

type Data struct {
	Periods []Period `json:"periods"`
	Current string   `json:"current"`
}
