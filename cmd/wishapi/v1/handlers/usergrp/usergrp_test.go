package usergrp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/so-heil/wishlist/business/email"
	"github.com/so-heil/wishlist/foundation/apitest"
)

func TestUserGroup(t *testing.T) {
	t.Parallel()
	l, err := apitest.Logger(true)
	if err != nil {
		t.Fatalf("create logger: %s", err)
	}

	srv, err := apitest.NewAPIServer(apitest.DefaultAPIServerConfig, l)
	if err != nil {
		t.Fatalf("start api server: %s", err)
	}
	defer srv.Close()

	database, err := apitest.NewDatabase(apitest.DefaultDatabaseConfig, l)
	if err != nil {
		t.Fatalf("create database: %s", err)
	}
	defer database.Close()

	const group = "ug"
	mailClient := newEmailClient()
	ug, err := New(Config{
		EmailVerifyExp:           time.Second,
		UserSessExp:              time.Second,
		MailTimeout:              time.Second,
		EmailVerificationSubject: "Email Verification",
		CacheSize:                100_000,
		OTPLength:                6,
		OTPTimeout:               10 * time.Second,
	}, mailClient, srv.App, srv.Auth, database.Dbase, l, "{{.}}")
	if err != nil {
		t.Fatalf("create usergroup: %s", err)
	}
	ug.Routes(group)

	verifyEmail := &apitest.Group{
		Name:   "verifyEmail",
		URL:    fmt.Sprintf("%s/%s%s", srv.URL, group, "/verify-email"),
		Method: http.MethodPost,
		Tests: []apitest.EndpointTest{
			{
				Name:       "validEmail",
				ReqBody:    `{"email": "test@test.com"}`,
				StatusCode: http.StatusNoContent,
			},
			{
				Name:       "invalidEmail",
				ReqBody:    `{"email": "test*test.com"}`,
				StatusCode: http.StatusBadRequest,
			},
			{
				Name:       "emptyBody",
				StatusCode: http.StatusBadRequest,
			},
			{
				Name:       "invalidField",
				ReqBody:    `{"id": "test@test.com"}`,
				StatusCode: http.StatusBadRequest,
			},
			{
				Name:       "invalidBody",
				ReqBody:    `"email": "test@test.com"}`,
				StatusCode: http.StatusBadRequest,
			},
			{
				Name:       "sentRecently",
				ReqBody:    `{"email": "test@test.com"}`,
				StatusCode: http.StatusTooEarly,
			},
			{
				Name:       "registeredUser",
				ReqBody:    `{"email": "parisa@gmail.com"}`,
				StatusCode: http.StatusBadRequest,
			},
		},
	}
	verifyEmail.Run(t)

	mailClient.shouldErr = true
	verifyUnavailable := apitest.Group{
		Name:   "mailClientUnavailable",
		URL:    fmt.Sprintf("%s/%s%s", srv.URL, group, "/verify-email"),
		Method: http.MethodPost,
		Tests: []apitest.EndpointTest{
			{
				Name:       "mailClientUnavailable",
				ReqBody:    `{"email": "testtest@test.com"}`,
				StatusCode: http.StatusServiceUnavailable,
			},
		},
	}
	verifyUnavailable.Run(t)
	mailClient.shouldErr = false

	type tokenResponse struct {
		Token string `json:"token"`
	}
	var emailVerifiedToken tokenResponse

	verifyOTP := apitest.Group{
		Name:   "verifyOTP",
		URL:    fmt.Sprintf("%s/%s%s", srv.URL, group, "/verify-otp"),
		Method: http.MethodPost,
		Tests: []apitest.EndpointTest{
			{
				Name:       "invalidOTP",
				ReqBody:    `{"email": "test@test.com", "otp": "000000"}`,
				StatusCode: http.StatusUnauthorized,
			},
			{
				Name:       "unknownEmail",
				ReqBody:    `{"email": "test1@test.com", "otp": "000000"}`,
				StatusCode: http.StatusUnauthorized,
			},
			{
				Name:       "invalidCode",
				ReqBody:    `{"email": "test*test.com", "otp": "0a0a0a"}`,
				StatusCode: http.StatusBadRequest,
			},
			// receive the code sent in previous test group and send to verify-otp
			{
				Name:       "validOTP",
				ReqBody:    fmt.Sprintf(`{"email": "test@test.com", "otp": "%s"}`, <-mailClient.transport),
				StatusCode: http.StatusOK,
				RespDst:    &emailVerifiedToken,
				Validate: func() error {
					if emailVerifiedToken.Token == "" {
						return fmt.Errorf("token should not be empty")
					}
					return nil
				},
			},
		},
	}
	verifyOTP.Run(t)

	register := apitest.Group{
		Name:   "register",
		URL:    fmt.Sprintf("%s/%s%s", srv.URL, group, "/register"),
		Method: http.MethodPost,
		Tests: []apitest.EndpointTest{
			{
				Name:       "validRegister",
				ReqBody:    `{"name": "test", "username": "test", "password": "test_testA1"}`,
				StatusCode: http.StatusCreated,
				Headers:    map[string]string{"Authorization": fmt.Sprintf("Bearer %s", emailVerifiedToken.Token)},
			},
			{
				Name:       "validRegisterInvalidToken",
				ReqBody:    `{"name": "test", "username": "test", "password": "test_testA1"}`,
				StatusCode: http.StatusUnauthorized,
				Headers:    map[string]string{"Authorization": fmt.Sprintf("Bearer* %s", emailVerifiedToken.Token)},
			},
			{
				Name:       "validRegisterMissingToken",
				ReqBody:    `{"name": "test", "username": "test", "password": "test_testA1"}`,
				StatusCode: http.StatusUnauthorized,
			},
			{
				Name:       "invalidBody",
				ReqBody:    `{"name": "", "username": "@8_", "password": "password"}`,
				StatusCode: http.StatusBadRequest,
			},
		},
	}
	register.Run(t)

	var userToken tokenResponse
	login := apitest.Group{
		Name:   "login",
		URL:    fmt.Sprintf("%s/%s%s", srv.URL, group, "/login"),
		Method: http.MethodPost,
		Tests: []apitest.EndpointTest{
			{
				Name:       "validAuth",
				RespDst:    &userToken,
				ReqBody:    `{"email": "test@test.com", "password": "test_testA1"}`,
				StatusCode: http.StatusOK,
				Validate: func() error {
					if userToken.Token == "" {
						return fmt.Errorf("user token should not be empty")
					}
					return nil
				},
			},
			{
				Name:       "invalidBody",
				ReqBody:    `{"email": "test@test.com"}`,
				StatusCode: http.StatusBadRequest,
			},
			{
				Name:       "notRegistered",
				ReqBody:    `{"email": "unknown@email.com"}`,
				StatusCode: http.StatusBadRequest,
			},
			{
				Name:       "wrongPassword",
				ReqBody:    `{"email": "test@test.com", "password": "password"}`,
				StatusCode: http.StatusUnauthorized,
			},
			{
				Name:       "invalidBody",
				ReqBody:    `{"name": "", "username": "@8_", "password": "password"}`,
				StatusCode: http.StatusBadRequest,
			},
		},
	}
	login.Run(t)
}

type emailClient struct {
	transport chan string
	shouldErr bool
}

func newEmailClient() *emailClient {
	transport := make(chan string, 10)
	return &emailClient{
		transport: transport,
		shouldErr: false,
	}
}

func (ec *emailClient) Send(_ context.Context, mail email.Mail) error {
	if ec.shouldErr {
		return errors.New("fake error")
	}
	fmt.Printf("mocking mail for code: %s\n", mail.Body)
	ec.transport <- mail.Body
	return nil
}
