package userdb

import (
	"time"

	"github.com/so-heil/wishlist/business/entities/user"
)

type dbUser struct {
	ID           int       `db:"id"`
	Username     string    `db:"username"`
	Name         string    `db:"name"`
	Email        string    `db:"email"`
	PasswordHash []byte    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
}

func toDBUser(usr *user.User) dbUser {
	return dbUser{
		ID:           usr.ID,
		Username:     usr.Username,
		Name:         usr.Name,
		Email:        usr.Email,
		PasswordHash: usr.PasswordHash,
		CreatedAt:    usr.CreatedAt,
	}
}

func (du *dbUser) toUser() user.User {
	return user.User{
		ID:           du.ID,
		Username:     du.Username,
		Name:         du.Name,
		Email:        du.Email,
		PasswordHash: du.PasswordHash,
		CreatedAt:    du.CreatedAt,
	}
}
