package otp

import (
	"errors"
	"testing"
	"time"

	"github.com/so-heil/wishlist/business/storage/keyvalue/kvstores"
)

func TestOTP(t *testing.T) {
	otp := New(kvstores.NewFreeCache(1024*100), 8, 5*time.Second)

	code, err := otp.GenCode()
	if err != nil {
		t.Fatal("code should be generated", err)
	}

	identity := "some_user"
	if err := otp.Save(identity, code); err != nil {
		t.Fatal("code should be saved", err)
	}

	exists, err := otp.Exists(identity)
	if err != nil {
		t.Fatal("should report if identity exists", err)
	}
	if !exists {
		t.Fatal("code should exist in otp store")
	}

	time.Sleep(6 * time.Second)
	stillExists, err := otp.Exists(identity)
	if err != nil {
		t.Fatal("should report if identity exists", err)
	}
	if stillExists {
		t.Fatal("code should be expired")
	}

	newCode, err := otp.GenCode()
	if err != nil {
		t.Fatal("code should be generated", err)
	}

	if newCode == code {
		t.Error("should generate different codes")
	}

	if err := otp.Save(identity, newCode); err != nil {
		t.Fatal("code should be saved", err)
	}

	if err := otp.Check(identity, "123112"); err != nil {
		if !errors.Is(err, ErrInvalidCode) {
			t.Error("should yield invalid code")
		}
	}

	if err := otp.Check(identity, newCode); err != nil {
		t.Error("should validate correct code")
	}
}
