package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
)

var shutdown = make(chan os.Signal, 1)

func runApp(t *testing.T, mw ...Middleware) (*App, string, func()) {
	log, err := zap.NewProduction()
	if err != nil {
		t.Fatalf("should be able to create a logger: %s", err)
	}
	l := log.Sugar()
	mux := http.NewServeMux()

	app := NewApp(l, mux, mw, shutdown, nil)
	srv := httptest.NewServer(app)
	return app, srv.URL, srv.Close
}

func TestHandle(t *testing.T) {
	app, url, close := runApp(t)
	defer close()

	type data struct {
		OK bool `json:"ok"`
	}

	h := func(w http.ResponseWriter, r *http.Request, ctx context.Context) error {
		d := data{OK: true}
		return Respond(w, ctx, d, http.StatusOK)
	}

	app.Handle(http.MethodGet, "testgrp", "/testpath", h, nil)

	getResp, err := http.Get(fmt.Sprintf("%s/testgrp/testpath", url))
	if err != nil {
		t.Fatalf("should be able to call handler over http: %s", err)
	}
	defer getResp.Body.Close()

	var r data
	if err := json.NewDecoder(getResp.Body).Decode(&r); err != nil {
		t.Fatalf("should be able to decode response: %s", err)
	}

	if r.OK != true {
		t.Errorf("should be same as initial data, response: %+v", r)
	}

	postResp, err := http.Post(fmt.Sprintf("%s/testgrp/testpath", url), "", nil)
	if err != nil {
		t.Fatalf("should be able to call handler over http: %s", err)
	}

	defer postResp.Body.Close()
	if postResp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("should return 405 on wrong method calls, status code: %d", postResp.StatusCode)
	}
}

func TestMiddleware(t *testing.T) {
	const key = "factor"

	// sets factor in context
	handlerMW := func(handler Handler) Handler {
		return func(w http.ResponseWriter, r *http.Request, ctx context.Context) error {
			ctx = context.WithValue(ctx, key, 0)
			return handler(w, r, ctx)
		}
	}

	// increases factor
	appMW := func(handler Handler) Handler {
		return func(w http.ResponseWriter, r *http.Request, ctx context.Context) error {
			f := ctx.Value(key).(int)
			ctx = context.WithValue(ctx, key, f+1)
			return handler(w, r, ctx)
		}
	}

	// increases factor one time
	h := func(w http.ResponseWriter, r *http.Request, ctx context.Context) error {
		f := ctx.Value(key).(int)
		if f != 1 {
			t.Errorf("context should have factor with value 1, but is: %d", f)
		}
		return Respond(w, ctx, struct{}{}, http.StatusOK)
	}

	app, url, close := runApp(t, appMW)
	defer close()

	app.Handle(http.MethodPost, "testgrp", "/testmw", h, handlerMW)

	resp, err := http.Post(fmt.Sprintf("%s/testgrp/testmw", url), "application/json", nil)
	if err != nil {
		t.Fatalf("should be able to call handler over http: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("should have 200 status code, has: %s", resp.Status)
	}
}

func TestError(t *testing.T) {
	app, url, close := runApp(t)
	defer close()

	h := func(w http.ResponseWriter, r *http.Request, ctx context.Context) error {
		return errors.New("bad things happened")
	}

	app.Handle(http.MethodPost, "testgrp", "/testerr", h, nil)

	resp, err := http.Post(fmt.Sprintf("%s/testgrp/testerr", url), "application/json", nil)
	if err != nil {
		t.Fatalf("should be able to call handler over http: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("should have 500 status code, has: %s", resp.Status)
	}

	timer := time.NewTimer(300 * time.Millisecond)
	select {
	case <-shutdown:
		t.Log("shutdown signal received")
	case <-timer.C:
		t.Fatalf("should have received shutdown signal but timeoput reached")
	}
}
