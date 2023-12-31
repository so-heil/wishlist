package auth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// EmailVerifiedClaims verifies that anyone with this claim has an email verified in our system
type EmailVerifiedClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func NewEmailVerifiedClaims(email string, expireDur time.Duration) *EmailVerifiedClaims {
	return &EmailVerifiedClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   email,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireDur)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
}

// UserClaims verifies that anyone with this claim is authenticated into system
type UserClaims struct {
	ID int `json:"id"`
	jwt.RegisteredClaims
}

func NewUserClaims(id int, expireDur time.Duration) *UserClaims {
	return &UserClaims{
		ID: id,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireDur)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
}

// SetUserID sets the user id into the context value which is then accessible by a call to GetUserID
func SetUserID(ctx context.Context, id int) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// GetUserID returns the user id attached to the context, if any
func GetUserID(ctx context.Context) (int, error) {
	id, ok := ctx.Value(userIDKey).(int)
	if !ok {
		return 0, ErrNoUserInContext
	}
	return id, nil
}
