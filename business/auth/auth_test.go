package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/so-heil/wishlist/business/keystore"
	"go.uber.org/zap"
)

type cleanUpFunc func()

func newAuth(t *testing.T) (*Auth, cleanUpFunc) {
	shutdown := make(chan os.Signal)
	l, err := zap.NewProduction()
	if err != nil {
		t.Fatalf("create logger: %s", err)
	}

	ks, err := keystore.New(500*time.Millisecond, time.Second, shutdown, l.Sugar())
	if err != nil {
		t.Fatalf("create keystore: %s", err)
	}

	return New(ks), func() {
		shutdown <- syscall.SIGKILL
	}
}

func TestAuth(t *testing.T) {
	a, cleanUp := newAuth(t)
	defer cleanUp()

	email := "test@test.com"
	evc := NewEmailVerifiedClaims(email, 3*time.Second)
	tk, err := a.Token(evc)
	if err != nil {
		t.Fatalf("email verfied token: %s", err)
	}

	var fromToken EmailVerifiedClaims
	if err := a.Parse(tk, &fromToken); err != nil {
		t.Errorf("parse token: %s", err)
	}
	if fromToken.Email != email {
		t.Errorf("concrete token email should be: %s, is: %s", email, fromToken.Email)
	}

	bearerToken := fmt.Sprintf("Bearer %s", tk)
	var fromBearerToken EmailVerifiedClaims
	if err := a.ParseFromBearer(bearerToken, &fromBearerToken); err != nil {
		t.Errorf("parse from bearer token: %s", err)
	}
	if fromBearerToken.Email != email {
		t.Errorf("concrete token email should be: %s, is: %s", email, fromToken.Email)
	}

	invalidToken := tk[:len(tk)-2]
	if err := a.Parse(tk, &fromToken); err != nil {
		if !errors.Is(err, ErrInvalidToken) {
			t.Errorf("should yield %s but got: %s", ErrInvalidToken, err)
		}
	}

	invalidBearer := fmt.Sprintf("Baerer %s", invalidToken)
	if err := a.ParseFromBearer(invalidBearer, &fromBearerToken); err != nil {
		if !errors.Is(err, ErrMalformedToken) {
			t.Errorf("should yield %s but got: %s", ErrMalformedToken, err)
		}
	}

	time.Sleep(2 * time.Second)
	if err := a.Parse(tk, &fromToken); err == nil {
		t.Errorf("parse should not succeed with expired token kid")
	}

	userID := 10
	uc := NewUserClaims(userID, 800*time.Millisecond)
	utk, err := a.Token(uc)
	if err != nil {
		t.Fatalf("user claims token: %s", utk)
	}

	var ucFromToken UserClaims
	if err := a.Parse(utk, &ucFromToken); err != nil {
		t.Errorf("parse user claims token: %s", err)
	}
	if userID != ucFromToken.ID {
		t.Errorf("user claims id must match user id, want %d got %d", userID, ucFromToken.ID)
	}

	// wait till claims are expired but kid is not
	time.Sleep(900 * time.Millisecond)
	if err := a.Parse(utk, &ucFromToken); err == nil {
		t.Error("parse expired token should not succeed")
	}

	ctx := context.Background()
	ctx = SetUserID(ctx, userID)
	idFromCtx, err := GetUserID(ctx)
	if err != nil {
		t.Errorf("should get id form context: %s", err)
	}
	if idFromCtx != userID {
		t.Errorf("id from ctx should match initial id: want: %d got: %d", userID, idFromCtx)
	}
}
