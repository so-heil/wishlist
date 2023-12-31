// Package otp provides API to deal with user email verification codes
package otp

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"text/template"
	"time"

	"github.com/so-heil/wishlist/business/storage/keyvalue"
)

var ErrInvalidCode = errors.New("verification code is not valid")

type OTP struct {
	s          keyvalue.KeyValueStore
	codeLen    int
	expiration time.Duration
	templ      *template.Template
}

func New(s keyvalue.KeyValueStore, codeLen int, expiration time.Duration, templ *template.Template) *OTP {
	return &OTP{
		s:          s,
		codeLen:    codeLen,
		expiration: expiration,
		templ:      templ,
	}
}

func (o OTP) Message(code string) (string, error) {
	buf := new(bytes.Buffer)
	if err := o.templ.Execute(buf, code); err != nil {
		return "", fmt.Errorf("execute otp template: %w", err)
	}
	return buf.String(), nil
}

func (o *OTP) Check(identity, code string) error {
	toMatch, err := o.s.Get(identity)
	if err != nil {
		if errors.Is(err, keyvalue.ErrNotFound) {
			return ErrInvalidCode
		}
		return err
	}

	if string(toMatch) != code {
		return ErrInvalidCode
	}

	o.s.Del(identity)
	return nil
}

func (o *OTP) Exists(identity string) (bool, error) {
	_, err := o.s.Get(identity)
	if err != nil {
		if errors.Is(err, keyvalue.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

var table = [...]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}

func (o *OTP) GenCode() (string, error) {
	r := make([]byte, o.codeLen)
	_, err := rand.Read(r)
	if err != nil {
		return "", fmt.Errorf("generate random code: %w", err)
	}

	for i := 0; i < len(r); i++ {
		r[i] = table[int(r[i])%len(table)]
	}

	return string(r), nil
}

func (o *OTP) Save(identity, code string) error {
	if err := o.s.Set(identity, []byte(code), o.expiration); err != nil {
		return fmt.Errorf("set keyvalue: %w", err)
	}
	return nil
}
