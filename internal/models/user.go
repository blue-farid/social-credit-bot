package models

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Balance  int    `json:"balance"`
}
