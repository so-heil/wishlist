package usergrp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"text/template"
	"time"

	"github.com/so-heil/wishlist/business/auth"
	"github.com/so-heil/wishlist/business/database/db"
	"github.com/so-heil/wishlist/business/email"
	"github.com/so-heil/wishlist/business/entities/user"
	"github.com/so-heil/wishlist/business/otp"
	"github.com/so-heil/wishlist/business/storage/keyvalue/kvstores"
	"github.com/so-heil/wishlist/business/storage/postgres/userdb"
	"github.com/so-heil/wishlist/foundation/web"
	"go.uber.org/zap"
)

type Config struct {
	EmailVerifyExp           time.Duration
	UserSessExp              time.Duration
	MailTimeout              time.Duration
	EmailVerificationSubject string
	CacheSize                int
	OTPLength                int
	OTPTimeout               time.Duration
}

type UserGroup struct {
	bookKeeper  *user.BookKeeper
	app         *web.App
	otpClient   *otp.OTP
	a           *auth.Auth
	emailClient email.Client
	cfg         Config
}

func New(
	cfg Config,
	emailClient email.Client,
	app *web.App,
	a *auth.Auth,
	dbase *db.DB,
	l *zap.SugaredLogger,
	otpTemplate string,
) (*UserGroup, error) {
	otpTempl, err := template.New("otp").Parse(otpTemplate)
	if err != nil {
		return nil, fmt.Errorf("create otp template: %w", err)
	}

	otpClient := otp.New(
		kvstores.NewFreeCache(cfg.CacheSize),
		cfg.OTPLength,
		cfg.OTPTimeout,
		otpTempl,
	)

	return &UserGroup{
		bookKeeper:  user.NewBookKeeper(userdb.New(dbase, l)),
		app:         app,
		otpClient:   otpClient,
		a:           a,
		emailClient: emailClient,
		cfg:         cfg,
	}, nil
}

func (ug *UserGroup) verifyEmail(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var aev APIEmailVerification
	if err := web.DecodeBody(r.Body, &aev); err != nil {
		return err
	}

	if _, err := ug.bookKeeper.LookUpEmail(ctx, aev.Email); err != nil {
		if !errors.Is(err, user.ErrUserNotFound) {
			return fmt.Errorf("lookup email: %w", err)
		}
	} else {
		return web.EUEFromError(user.ErrUniqueEmail, http.StatusBadRequest)
	}

	exists, err := ug.otpClient.Exists(aev.Email)
	if err != nil {
		return fmt.Errorf("check code exists: %w", err)
	}

	if exists {
		return web.EndUserError{
			Message: user.ErrEmailVerifySoon.Error(),
			Status:  http.StatusTooEarly,
		}
	}

	code, err := ug.otpClient.GenCode()
	if err != nil {
		return fmt.Errorf("generate otp code: %w", err)
	}

	message, err := ug.otpClient.Message(code)
	if err != nil {
		return fmt.Errorf("message for otp: %w", err)
	}

	mailCtx, cancel := context.WithTimeout(context.Background(), ug.cfg.MailTimeout)
	defer cancel()

	if err := ug.emailClient.Send(mailCtx, email.Mail{
		Body:    message,
		Subject: ug.cfg.EmailVerificationSubject,
		To:      aev.Email,
	}); err != nil {
		return web.ExternalError{
			Err: fmt.Errorf("send email verification mail: %w", err),
		}
	}

	if err := ug.otpClient.Save(aev.Email, code); err != nil {
		return fmt.Errorf("save otp code: %w", err)
	}

	return web.Respond(w, ctx, nil, http.StatusNoContent)
}

func (ug *UserGroup) verifyOTP(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var aov APIOTPVerfication
	if err := web.DecodeBody(r.Body, &aov); err != nil {
		return err
	}

	if err := ug.otpClient.Check(aov.Email, aov.OTP); err != nil {
		return web.EUEFromError(otp.ErrInvalidCode, http.StatusUnauthorized)
	}

	tk, err := ug.a.Token(auth.NewEmailVerifiedClaims(aov.Email, ug.cfg.EmailVerifyExp))
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
	err := ug.a.ParseFromBearer(r.Header.Get("Authorization"), &evc)
	if err != nil {
		return web.EUEFromError(auth.ErrInvalidToken, http.StatusUnauthorized)
	}

	if _, err := ug.bookKeeper.Create(ctx, user.NewUser{
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

	usr, err := ug.bookKeeper.Authenticate(ctx, user.UserAuthenticate{
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

	tk, err := ug.a.Token(auth.NewUserClaims(usr.ID, ug.cfg.UserSessExp))
	if err != nil {
		return fmt.Errorf("gen token for authenticated user: %w", err)
	}

	return web.Respond(w, ctx, token{Token: tk}, http.StatusOK)
}

func (ug *UserGroup) Routes(group string) {
	ug.app.Handle(http.MethodPost, group, "/verify-email", ug.verifyEmail)
	ug.app.Handle(http.MethodPost, group, "/verify-otp", ug.verifyOTP)
	ug.app.Handle(http.MethodPost, group, "/register", ug.register)
	ug.app.Handle(http.MethodPost, group, "/login", ug.authenticate)
}
