package userdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/so-heil/wishlist/business/database/db"
	"github.com/so-heil/wishlist/business/entities/user"
	"go.uber.org/zap"
)

type UserDB struct {
	*db.DB
	l *zap.SugaredLogger
}

func New(dbase *db.DB, l *zap.SugaredLogger) *UserDB {
	return &UserDB{
		DB: dbase,
		l:  l,
	}
}

func (udb *UserDB) Create(ctx context.Context, usr *user.User) error {
	const q = `
	INSERT INTO "user" 
			(email, username, password_hash, name, created_at)
		VALUES
			(:email, :username, :password_hash, :name, :created_at)
		RETURNING id`

	dbu := toDBUser(usr)
	err := udb.NamedQueryStructUpdate(ctx, q, &dbu)
	if err != nil {
		if errors.Is(err, db.ErrDBDuplicatedEntry) {
			return fmt.Errorf("NamedExecContext: %w", user.ErrUniqueEmail)
		}
		return err
	}
	usr.ID = dbu.ID

	return nil
}

func (udb *UserDB) LookUpEmail(ctx context.Context, email string) (user.User, error) {
	const q = `SELECT id, email, username, password_hash, name, created_at FROM "user" WHERE email = :email`

	du := dbUser{Email: email}
	err := udb.NamedQueryStructUpdate(ctx, q, &du)
	if err != nil {
		if errors.Is(err, db.ErrDBNotFound) {
			return user.User{}, user.ErrUserNotFound
		}
		return user.User{}, err
	}

	return du.toUser(), nil
}
