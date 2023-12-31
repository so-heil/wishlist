package user

import (
	"time"
)

type User struct {
	ID           int
	Username     string
	Name         string
	Email        string
	PasswordHash []byte
	CreatedAt    time.Time
}

type NewUser struct {
	Name     string
	Email    string
	Username string
	Password string
}

type UserAuthenticate struct {
	Email    string
	Password string
}
