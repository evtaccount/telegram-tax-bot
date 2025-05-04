package model

type Period struct {
	In      string `json:"in,omitempty"`
	Out     string `json:"out,omitempty"`
	Country string `json:"country"`
}
