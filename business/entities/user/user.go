package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUniqueEmail      = errors.New("email is not unique")
	ErrUserNotFound     = errors.New("user not found")
	ErrWrongCredentials = errors.New("wrong credentials")
	ErrEmailVerifySoon  = errors.New("a code has been generated lately")
	ErrInvalidOTP       = errors.New("OTP is not valid")
)

type Storage interface {
	Create(context.Context, *User) error
	LookUpEmail(context.Context, string) (User, error)
}

type BookKeeper struct {
	storage Storage
}

func NewBookKeeper(storer Storage) *BookKeeper {
	return &BookKeeper{storage: storer}
}

func (bk *BookKeeper) Create(ctx context.Context, nu NewUser) (User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(nu.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("generate hash from password: %w", err)
	}

	usr := User{
		Username:     nu.Username,
		Name:         nu.Name,
		Email:        nu.Email,
		PasswordHash: hash,
		CreatedAt:    time.Now(),
	}

	if err := bk.storage.Create(ctx, &usr); err != nil {
		return User{}, fmt.Errorf("store user: %w", err)
	}

	return usr, nil
}

func (bk *BookKeeper) Authenticate(ctx context.Context, ua UserAuthenticate) (User, error) {
	usr, err := bk.storage.LookUpEmail(ctx, ua.Email)
	if err != nil {
		return User{}, err
	}

	if err := bcrypt.CompareHashAndPassword(usr.PasswordHash, []byte(ua.Password)); err != nil {
		return User{}, ErrWrongCredentials
	}

	return usr, nil
}

func (bk *BookKeeper) LookUpEmail(ctx context.Context, email string) (User, error) {
	usr, err := bk.storage.LookUpEmail(ctx, email)
	if err != nil {
		return User{}, err
	}

	return usr, nil
}
