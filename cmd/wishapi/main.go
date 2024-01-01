package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/so-heil/wishlist/business/auth"
	"github.com/so-heil/wishlist/business/database/db"
	"github.com/so-heil/wishlist/business/email"
	"github.com/so-heil/wishlist/business/keystore"
	"github.com/so-heil/wishlist/business/validate"
	"github.com/so-heil/wishlist/business/web/middlewares"
	"github.com/so-heil/wishlist/cmd/wishapi/v1/handlers/probes"
	"github.com/so-heil/wishlist/cmd/wishapi/v1/handlers/usergrp"
	"github.com/so-heil/wishlist/foundation/web"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.9.0"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("shutting down: %s", err)
	}
}

type config struct {
	Web struct {
		Address         string        `env:"ADDRESS" envDefault:"0.0.0.0:3000"`
		ReadTimeout     time.Duration `env:"READ_TIMEOUT" envDefault:"5s"`
		WriteTimeout    time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
		IdleTimeout     time.Duration `env:"IDLE_TIMEOUT" envDefault:"120s"`
		ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"15s"`
	}
	App struct {
		Users struct {
			OTPLength                int           `env:"OTP_LENGTH" envDefault:"6"`
			OTPTimeout               time.Duration `env:"OTP_TIMEOUT" envDefault:"90s"`
			OTPTemplate              string        `env:"OTP_TEMPLATE" envDefault:"Your email verification code is {{.}}."`
			EmailVerifiedExpiration  time.Duration `env:"EMAIL_VERIFIED_EXPIRATION" envDefault:"30m"`
			UserSessionExpiration    time.Duration `env:"USER_SESSION_EXPIRATION" envDefault:"36h"`
			EmailVerificationSubject string        `env:"EMAIL_VERIFICATION_SUBJECT" envDefault:"Email Verification Code"`
			SendMailContextTimeout   time.Duration `env:"SEND_MAIL_CONTEXT_TIMEOUT" envDefault:"10s"`
			CourierAPIKey            string        `env:"COURIER_API_KEY"`
		}
		CacheSize           int           `env:"CACHE_SIZE" envDefault:"100000"`
		KeyRotationPeriod   time.Duration `env:"KEY_ROTATION_PERIOD" envDefault:"24h"`
		KeyExpirationPeriod time.Duration `env:"KEY_EXPIRATION_PERIOD" envDefault:"48h"`
	}
	DB struct {
		User       string `env:"DB_USER" envDefault:"postgres"`
		Password   string `env:"DB_PASSWORD" envDefault:"postgres"`
		Host       string `env:"DB_HOST" envDefault:"postgres-svc:5432"`
		Name       string `env:"DB_NAME" envDefault:""`
		Schema     string `env:"DB_SCHEMA" envDefault:""`
		DisableTLS bool   `env:"DB_DISABLE_TLS" envDefault:"true"`
	}
	Debug struct {
		CollectorURL string  `env:"COLLECTOR_URL" envDefault:"http://zipkin.observe.svc.cluster.local:9411/api/v2/spans"`
		Probibility  float64 `env:"PROBIBILITY" envDefault:"1"`
	}
	Mail struct {
		Password string `env:"GMAIL_PASSWORD"`
		Host     string `env:"GMAIL_HOST"`
		From     string `env:"GMAIL_FROM"`
		Port     string `env:"GMAIL_PORT" envDefault:"587"`
	}
}

func run() error {
	// *** Read config ***
	var cfg config
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	// *** Construct logger ***
	logger, lerr := zap.NewProduction()
	if lerr != nil {
		return fmt.Errorf("construct logger: %w", lerr)
	}
	l := logger.Sugar()

	// *** Init graceful shutdown ***
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGTERM, syscall.SIGINT)
	serverErr := make(chan error, 1)

	// *** Start tracer ***
	l.Infoln("startup: starting tracer")
	traceProvider, terr := startTracing(
		"wishapi",
		cfg.Debug.CollectorURL,
		cfg.Debug.Probibility,
	)
	if terr != nil {
		return fmt.Errorf("start tracer: %w", terr)
	}
	defer func(traceProvider *trace.TracerProvider, ctx context.Context) {
		if terr := traceProvider.Shutdown(ctx); terr != nil {
			fmt.Printf("trace shutdown: %s", terr)
		}
	}(traceProvider, context.Background())
	tracer := traceProvider.Tracer("webserver")

	// *** Init database connection ***
	l.Infoln("startup: connecting to database")
	database, cerr := db.Open(&db.Config{
		User:         cfg.DB.User,
		Password:     cfg.DB.Password,
		Host:         cfg.DB.Host,
		Name:         cfg.DB.Name,
		Schema:       cfg.DB.Schema,
		MaxIdleConns: 0,
		MaxOpenConns: 0,
		DisableTLS:   cfg.DB.DisableTLS,
	}, l)
	if cerr != nil {
		return fmt.Errorf("open database connection: %w", cerr)
	}

	// *** Init keystore and auth ***
	l.Infoln("startup: initializing keystore and auth")
	ks, err := keystore.New(cfg.App.KeyRotationPeriod, cfg.App.KeyExpirationPeriod, shutdown, l)
	if err != nil {
		return fmt.Errorf("init keystore: %w", err)
	}
	a := auth.New(ks)

	// *** Init web.App ***
	l.Infoln("startup: init web app")
	app := web.NewApp(
		l,
		http.NewServeMux(),
		[]web.Middleware{middlewares.Log(l), middlewares.Errors(l)},
		shutdown,
		tracer,
	)

	// *** Init validator ***
	if err := validate.Init(); err != nil {
		return fmt.Errorf("init validator: %w", err)
	}

	// *** Build handler groups and register routes to app ***
	emailClient := email.NewCourierClient(cfg.App.Users.CourierAPIKey)
	userGroup, err := usergrp.New(usergrp.Config{
		EmailVerifyExp:           cfg.App.Users.EmailVerifiedExpiration,
		UserSessExp:              cfg.App.Users.UserSessionExpiration,
		MailTimeout:              cfg.App.Users.SendMailContextTimeout,
		EmailVerificationSubject: cfg.App.Users.EmailVerificationSubject,
		CacheSize:                cfg.App.CacheSize,
		OTPLength:                cfg.App.Users.OTPLength,
		OTPTimeout:               cfg.App.Users.OTPTimeout,
	}, emailClient, app, a, database, l, cfg.App.Users.OTPTemplate)
	if err != nil {
		return fmt.Errorf("create usergroup: %w", err)
	}

	handlerGroups{
		"debug": probes.New(l, app),
		"users": userGroup,
	}.handleAll()

	// *** Start server ***
	srv := http.Server{
		Addr:         cfg.Web.Address,
		Handler:      app,
		TLSConfig:    nil,
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		IdleTimeout:  cfg.Web.IdleTimeout,
	}
	go func() {
		l.Infow("startup: starting wishapi web service", "address", cfg.Web.Address)
		serverErr <- srv.ListenAndServe()
	}()

	// *** Listen for shutdown signal ***
	select {
	case err := <-serverErr:
		return fmt.Errorf("web server: %w", err)
	case <-shutdown:
		l.Infoln("shutdown: starting graceful shutdown")
		defer l.Infoln("shutdown: shutdown completed")

		ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			srv.Close()
			return fmt.Errorf("shutdown: %w", err)
		}
	}

	return nil
}

type handlerGroup interface {
	Routes(group string)
}

type handlerGroups map[string]handlerGroup

func (g handlerGroups) handleAll() {
	for name, group := range g {
		group.Routes(name)
	}
}

func startTracing(serviceName, collectURL string, probability float64) (*trace.TracerProvider, error) {
	exporter, err := zipkin.New(collectURL)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithSampler(trace.TraceIDRatioBased(probability)),
		trace.WithBatcher(exporter, trace.WithBatchTimeout(time.Second)),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(serviceName),
			)),
	)
	otel.SetTracerProvider(traceProvider)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return traceProvider, nil
}
