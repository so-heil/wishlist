package keystore_test

import (
	"errors"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/so-heil/wishlist/business/keystore"
	"go.uber.org/zap"
)

func TestKeyStore(t *testing.T) {
	shutdown := make(chan os.Signal, 1)
	logger, err := zap.NewProduction()
	if err != nil {
		t.Fatal(err)
	}
	l := logger.Sugar()
	rotationPeriod := 100 * time.Millisecond
	expirationPeriod := 2 * time.Second
	tolerance := 50 * time.Millisecond
	ks, err := keystore.New(rotationPeriod, expirationPeriod, shutdown, l)
	if err != nil {
		t.Fatal(err)
	}

	activeId, key, err := ks.Active()
	if err != nil {
		t.Fatalf("keystore should return an active key after init: %s", err)
	}

	type TestClaims struct {
		ID string `json:"ID,omitempty"`
		jwt.RegisteredClaims
	}
	initId := "test_id"
	tk := jwt.NewWithClaims(jwt.SigningMethodEdDSA, TestClaims{
		ID: initId,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "test",
			Subject:   "test",
			ExpiresAt: jwt.NewNumericDate(key.Expire),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	},
	)
	tk.Header["kid"] = activeId

	signedStr, err := tk.SignedString(key.Signer)
	if err != nil {
		t.Fatalf("should be able to sign claims: %s", err)
	}

	testClaims := func() (string, error) {
		var resClaims TestClaims
		_, err = jwt.ParseWithClaims(signedStr, &resClaims, func(token *jwt.Token) (interface{}, error) {
			kid := token.Header["kid"].(string)
			sig, err := ks.Signer(kid)
			if err != nil {
				return nil, err
			}
			return sig.Signer.Public(), nil
		})
		if err != nil {
			return "", err
		}
		return resClaims.ID, nil
	}

	cId, err := testClaims()
	if err != nil {
		t.Fatalf("claims should be evaluated: %s", err)
	}
	if cId != initId {
		t.Errorf("claim id want %s got %s", initId, cId)
	}

	// Wait for a rotation
	time.Sleep(rotationPeriod + tolerance)
	rActiveId, _, _ := ks.Active()
	if rActiveId == activeId {
		t.Errorf("active key should have been rotated by now")
	}

	cIdAfterRot, err := testClaims()
	if err != nil {
		t.Fatalf("claims should be evaluated: %s", err)
	}
	if cIdAfterRot != initId {
		t.Errorf("claim id want %s got %s", initId, cId)
	}

	// Wait for an expiration
	time.Sleep(expirationPeriod)
	_, err = testClaims()
	if err != nil {
		if !errors.Is(err, keystore.ErrInvalidKey) {
			t.Errorf("should yield invalid key error, but errored: %s", err)
		}
	} else {
		t.Error("should yield invalid key error, but returned no error")
	}

	beforeSt, _, err := ks.Active()
	if err != nil {
		t.Fatal("should get active key: %w", err)
	}
	shutdown <- syscall.SIGTERM
	// Wait for another rotation period
	time.Sleep(rotationPeriod + tolerance)
	afterSt, _, err := ks.Active()
	if err != nil {
		t.Fatal("should get active key: %w", err)
	}

	if beforeSt != afterSt {
		t.Errorf("should have stopped rotation")
	}
}
