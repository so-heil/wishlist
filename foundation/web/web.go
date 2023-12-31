// Package web describes a small web-framework
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

type Middleware func(Handler) Handler

type App struct {
	log      *zap.SugaredLogger
	mux      *http.ServeMux
	mw       []Middleware
	shutdown chan os.Signal
	tracer   trace.Tracer
}

func NewApp(log *zap.SugaredLogger, mux *http.ServeMux, mw []Middleware, shutdown chan os.Signal, tracer trace.Tracer) *App {
	return &App{
		log:      log,
		mux:      mux,
		mw:       mw,
		shutdown: shutdown,
		tracer:   tracer,
	}
}

func applyMiddlewares(handler Handler, mw []Middleware) Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		mid := mw[i]
		if mid != nil {
			handler = mid(handler)
		}
	}

	return handler
}

func (app *App) shutServerDown() {
	app.shutdown <- syscall.SIGTERM
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.mux.ServeHTTP(w, r)
}

func (app *App) Handle(method, group, path string, handler Handler, mw ...Middleware) {
	handler = applyMiddlewares(handler, mw)
	handler = applyMiddlewares(handler, app.mw)

	h := func(w http.ResponseWriter, r *http.Request) {
		ctx, span := app.startSpan(w, r)
		defer span.End()

		v := Values{
			TraceID: span.SpanContext().TraceID().String(),
			Tracer:  app.tracer,
			Now:     time.Now().UTC(),
		}
		ctx = setValues(ctx, &v)

		if r.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if err := handler(ctx, w, r); err != nil {
			// lost integrity, shut down the app
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			if _, err := w.Write([]byte("something went really wrong")); err != nil {
				app.shutServerDown()
				return
			}
			app.shutServerDown()
			return
		}
	}

	finalPath := path
	if group != "" {
		finalPath = "/" + group + path
	}

	app.mux.HandleFunc(finalPath, h)
}

func Respond(w http.ResponseWriter, ctx context.Context, data any, statusCode int) error {
	SetStatusCode(ctx, statusCode)
	if statusCode == http.StatusNoContent || data == nil {
		w.WriteHeader(statusCode)
		return nil
	}

	jsn, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if _, err := w.Write(jsn); err != nil {
		return err
	}
	return nil
}

func (app *App) startSpan(w http.ResponseWriter, r *http.Request) (context.Context, trace.Span) {
	ctx := r.Context()

	span := trace.SpanFromContext(ctx)

	if app.tracer != nil {
		ctx, span = app.tracer.Start(ctx, "pkg.web.handle")
		span.SetAttributes(attribute.String("endpoint", r.RequestURI))
	}

	// Inject the trace information into the response.
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(w.Header()))

	return ctx, span
}

type validator interface {
	Validate() error
}

func DecodeBody(body io.ReadCloser, dst any) error {
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return EndUserError{
			Message: "request body is malformed",
			Status:  http.StatusBadRequest,
		}
	}

	v, ok := dst.(validator)
	if ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("validation: %w", err)
		}
	}

	return nil
}
