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
	"github.com/so-heil/wishlist/business/web/middlewares"
	"github.com/so-heil/wishlist/cmd/wishapi/v1/handlers/wishlist"
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
	Development     bool          `env:"DEVELOPMENT" envDefault:"true"`
	Address         string        `env:"ADDRESS" envDefault:"0.0.0.0:3000"`
	ReadTimeout     time.Duration `env:"READ_TIMEOUT" envDefault:"5s"`
	WriteTimeout    time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
	IdleTimeout     time.Duration `env:"IDLE_TIMEOUT" envDefault:"120s"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"15s"`
	CollectorURL    string        `env:"COLLECTOR_URL" envDefault:"http://zipkin.observe.svc.cluster.local:9411/api/v2/spans"`
	Probibility     float64       `env:"PROBIBILITY" envDefault:"1"`
}

func run() error {
	// *** Read config ***
	var cfg config
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	// *** Construct logger ***
	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("construct logger: %w", err)
	}
	l := logger.Sugar()

	// *** Graceful shutdown init ***
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGTERM, syscall.SIGINT)
	serverErr := make(chan error, 1)

	// *** Start tracer ***
	l.Infoln("startup: starting tracer")
	traceProvider, err := startTracing(
		"wishapi",
		cfg.CollectorURL,
		cfg.Probibility,
	)
	if err != nil {
		return fmt.Errorf("start tracer: %w", err)
	}
	defer traceProvider.Shutdown(context.Background())
	tracer := traceProvider.Tracer("webserver")

	// *** Build web service ***
	l.Infoln("startup: building web service")
	mux := http.NewServeMux()
	mw := []web.Middleware{middlewares.Log(l)}
	app := web.NewApp(l, mux, mw, shutdown, tracer)

	wl := wishlist.New(app)
	wl.HandleRoutes("v1")
	srv := http.Server{
		Addr:         cfg.Address,
		Handler:      app,
		TLSConfig:    nil,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	go func() {
		l.Infow("startup: starting wishapi web service", "address", cfg.Address)
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("web server: %w", err)
	case <-shutdown:
		l.Infoln("shutdown: starting graceful shutdown")
		defer l.Infoln("shutdown: shutdown completed")

		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			srv.Close()
			return fmt.Errorf("shutdown: %w", err)
		}
	}

	return nil
}

func startTracing(serviceName, collectorUrl string, probability float64) (*trace.TracerProvider, error) {
	exporter, err := zipkin.New(collectorUrl)
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
