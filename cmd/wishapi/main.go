package main

import (
	"fmt"
	"github.com/caarlos0/env/v10"
	"go.uber.org/zap"
	"log"
	"net/http"
	"time"
)

func testHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello http"))
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("shutting down: %s", err)
	}
}

type config struct {
	Development  bool          `env:"DEVELOPMENT" envDefault:"true"`
	Address      string        `env:"ADDRESS" envDefault:"0.0.0.0:3000"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT" envDefault:"5s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
	IdleTimeout  time.Duration `env:"IDLE_TIMEOUT" envDefault:"120s"`
}

func run() error {
	// *** Read config ***
	var cfg config
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// *** Construct logger ***
	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("construct logger: %w", err)
	}
	l := logger.Sugar()

	// *** Building web service ***
	srv := http.Server{
		Addr:         cfg.Address,
		Handler:      http.HandlerFunc(testHandler),
		TLSConfig:    nil,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	l.Infow("starting wishapi web service", "address", cfg.Address)
	if err := srv.ListenAndServe(); err != nil {
		return fmt.Errorf("wishapi web server: %w", err)
	}

	return nil
}
