package usergrp

import "github.com/so-heil/wishlist/business/validate"

type APINewUser struct {
	Name     string `json:"name" validate:"required"`
	Username string `json:"username" validate:"required,lowercase,alphanum,max=20"`
	Password string `json:"password" validate:"required,password"`
}

func (anu *APINewUser) Validate() error {
	return validate.Check(anu)
}

type APIUserAuthentication struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=32"`
}

func (aua *APIUserAuthentication) Validate() error {
	return validate.Check(aua)
}

type APIEmailVerification struct {
	Email string `json:"email" validate:"required,email"`
}

func (aev *APIEmailVerification) Validate() error {
	return validate.Check(aev)
}

type APIOTPVerfication struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required,len=6,numeric"`
}

func (aov *APIOTPVerfication) Validate() error {
	return validate.Check(aov)
}

type token struct {
	Token string `json:"token"`
}
