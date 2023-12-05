// Package otp provides API to deal with user email verification codes
package otp

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/so-heil/wishlist/business/storage/keyvalue"
)

var ErrInvalidCode = errors.New("verification code is not valid")

type OTP struct {
	s          keyvalue.KeyValueStore
	codeLen    int
	expiration time.Duration
}

func New(s keyvalue.KeyValueStore, codeLen int, expiration time.Duration) *OTP {
	return &OTP{
		s:          s,
		codeLen:    codeLen,
		expiration: expiration,
	}
}

var table = [...]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}

func (v *OTP) GenCode() (string, error) {
	r := make([]byte, v.codeLen)
	_, err := rand.Read(r)
	if err != nil {
		return "", fmt.Errorf("generate random code: %w", err)
	}

	for i := 0; i < len(r); i++ {
		r[i] = table[int(r[i])%len(table)]
	}

	return string(r), nil
}

func (v *OTP) Save(identity, code string) error {
	if err := v.s.Set(identity, []byte(code), v.expiration); err != nil {
		return fmt.Errorf("set keyvalue: %w", err)
	}
	return nil
}

func (v *OTP) Check(identity, code string) error {
	cmail, err := v.s.Get(identity)
	if err != nil {
		if errors.Is(err, keyvalue.ErrNotFound) {
			return ErrInvalidCode
		}
		return err
	}

	if string(cmail) != code {
		return ErrInvalidCode
	}

	v.s.Del(identity)
	return nil
}

func (v *OTP) Exists(identity string) (bool, error) {
	_, err := v.s.Get(identity)
	if err != nil {
		if errors.Is(err, keyvalue.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
