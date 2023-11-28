// Package web describes a small web-framework
package web

import (
	"context"
	"encoding/json"
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

type Handler func(http.ResponseWriter, *http.Request, context.Context) error

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
	handler = applyMiddlewares(handler, app.mw)
	handler = applyMiddlewares(handler, mw)

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

		if err := handler(w, r, ctx); err != nil {
			// lost integrity, shut down the app
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("something went really wrong"))
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
	jsonData, err := json.Marshal(data)
	SetStatusCode(ctx, statusCode)

	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if _, err := w.Write(jsonData); err != nil {
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
