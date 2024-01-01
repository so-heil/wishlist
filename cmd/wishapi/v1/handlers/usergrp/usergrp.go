package usergrp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/so-heil/wishlist/business/auth"
	"github.com/so-heil/wishlist/business/email"
	"github.com/so-heil/wishlist/business/entities/user"
	"github.com/so-heil/wishlist/business/otp"
	"github.com/so-heil/wishlist/foundation/web"
)

type Config struct {
	EmailVerifyExp           time.Duration
	UserSessExp              time.Duration
	MailTimeout              time.Duration
	EmailVerificationSubject string
}

type UserGroup struct {
	BookKeeper  *user.BookKeeper
	App         *web.App
	OTP         *otp.OTP
	Auth        *auth.Auth
	EmailClient email.Client
	Config      Config
}

func (ug *UserGroup) verifyEmail(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var aev APIEmailVerification
	if err := web.DecodeBody(r.Body, &aev); err != nil {
		return err
	}

	if _, err := ug.BookKeeper.LookUpEmail(ctx, aev.Email); err != nil {
		if !errors.Is(err, user.ErrUserNotFound) {
			return fmt.Errorf("lookup email: %w", err)
		}
	} else {
		return web.EUEFromError(user.ErrUniqueEmail, http.StatusBadRequest)
	}

	exists, err := ug.OTP.Exists(aev.Email)
	if err != nil {
		return fmt.Errorf("check code exists: %w", err)
	}

	if exists {
		return web.EndUserError{
			Message: user.ErrEmailVerifySoon.Error(),
			Status:  http.StatusTooEarly,
		}
	}

	code, err := ug.OTP.GenCode()
	if err != nil {
		return fmt.Errorf("generate otp code: %w", err)
	}

	message, err := ug.OTP.Message(code)
	if err != nil {
		return fmt.Errorf("message for otp: %w", err)
	}

	mailCtx, cancel := context.WithTimeout(context.Background(), ug.Config.MailTimeout)
	defer cancel()

	if err := ug.EmailClient.Send(mailCtx, email.Mail{
		Body:    message,
		Subject: ug.Config.EmailVerificationSubject,
		To:      aev.Email,
	}); err != nil {
		return web.ExternalError{
			Err: fmt.Errorf("send email verification mail: %w", err),
		}
	}

	if err := ug.OTP.Save(aev.Email, code); err != nil {
		return fmt.Errorf("save otp code: %w", err)
	}

	return web.Respond(w, ctx, nil, http.StatusNoContent)
}

func (ug *UserGroup) verifyOTP(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var aov APIOTPVerfication
	if err := web.DecodeBody(r.Body, &aov); err != nil {
		return err
	}

	if err := ug.OTP.Check(aov.Email, aov.OTP); err != nil {
		return web.EUEFromError(otp.ErrInvalidCode, http.StatusUnauthorized)
	}

	tk, err := ug.Auth.Token(auth.NewEmailVerifiedClaims(aov.Email, ug.Config.EmailVerifyExp))
	if err != nil {
		return fmt.Errorf("gen token for email verified: %w", err)
	}

	return web.Respond(w, ctx, token{tk}, http.StatusOK)
}

func (ug *UserGroup) register(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var anu APINewUser
	if err := web.DecodeBody(r.Body, &anu); err != nil {
		return err
	}

	var evc auth.EmailVerifiedClaims
	err := ug.Auth.ParseFromBearer(r.Header.Get("Authorization"), &evc)
	if err != nil {
		return web.EUEFromError(auth.ErrInvalidToken, http.StatusUnauthorized)
	}

	if _, err := ug.BookKeeper.Create(ctx, user.NewUser{
		Name:     anu.Name,
		Email:    evc.Email,
		Username: anu.Username,
		Password: anu.Password,
	}); err != nil {
		if errors.Is(err, user.ErrUniqueEmail) {
			return web.EUEFromError(err, http.StatusBadRequest)
		}
		return fmt.Errorf("create new user: %w", err)
	}

	return web.Respond(w, ctx, nil, http.StatusCreated)
}

func (ug *UserGroup) authenticate(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var aua APIUserAuthentication
	if err := web.DecodeBody(r.Body, &aua); err != nil {
		return err
	}

	usr, err := ug.BookKeeper.Authenticate(ctx, user.UserAuthenticate{
		Email:    aua.Email,
		Password: aua.Password,
	})
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return web.EndUserError{
				Message: "this email is not registered",
				Status:  http.StatusBadRequest,
			}
		}
		if errors.Is(err, user.ErrWrongCredentials) {
			return web.EUEFromError(err, http.StatusUnauthorized)
		}
		return err
	}

	tk, err := ug.Auth.Token(auth.NewUserClaims(usr.ID, ug.Config.UserSessExp))
	if err != nil {
		return fmt.Errorf("gen token for authenticated user: %w", err)
	}

	return web.Respond(w, ctx, token{Token: tk}, http.StatusOK)
}

func (ug *UserGroup) Routes(group string) {
	ug.App.Handle(http.MethodPost, group, "/verify-email", ug.verifyEmail)
	ug.App.Handle(http.MethodPost, group, "/verify-otp", ug.verifyOTP)
	ug.App.Handle(http.MethodPost, group, "/register", ug.register)
	ug.App.Handle(http.MethodPost, group, "/login", ug.authenticate)
}
