package users

import (
	"loyalty-system/internal/models/money"
	"time"
)

type User struct {
	Id               string
	Login            string
	HashPassword     string
	RegistrationDate time.Time
}

type Balance struct {
	Current   money.Money `json:"current"`
	Withdrawn money.Money `json:"withdrawn"`
}
