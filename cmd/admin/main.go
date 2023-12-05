package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/golang-migrate/migrate/v4"
	"github.com/so-heil/wishlist/business/database/db"
	"github.com/so-heil/wishlist/business/database/migration"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type config struct {
	DB struct {
		User              string        `env:"DB_USER" envDefault:"postgres"`
		Password          string        `env:"DB_PASSWORD" envDefault:"postgres"`
		Host              string        `env:"DB_HOST" envDefault:"postgres-svc:5432"`
		Name              string        `env:"DB_NAME" envDefault:""`
		Schema            string        `env:"DB_SCHEMA" envDefault:""`
		DisableTLS        bool          `env:"DB_DISABLE_TLS" envDefault:"true"`
		ConnectionTimeout time.Duration `env:"CONNECTION_TIMEOUT" envDefault:"120s"`
		SeedTimeout       time.Duration `env:"SEED_TIMEOUT" envDefault:"20s"`
	}
}

func run() error {
	args := os.Args[1:]

	if err := checkArgs(args); err != nil {
		return err
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("construct logger: %w", err)
	}
	l := logger.Sugar()

	switch args[0] {
	case "migrate":
		return startMigration(args, l)
	default:
		return fmt.Errorf("admin: command %s not supported", args[0])
	}
}

func checkArgs(args []string) error {
	if len(args) < 1 {
		return errors.New("not enough commands")
	}
	return nil
}

func startMigration(args []string, l *zap.SugaredLogger) error {
	args = args[1:]
	if err := checkArgs(args); err != nil {
		return err
	}

	var cfg config
	if err := env.Parse(&cfg); err != nil {
		return err
	}

	database, err := db.Open(&db.Config{
		User:         cfg.DB.User,
		Password:     cfg.DB.Password,
		Host:         cfg.DB.Host,
		Name:         cfg.DB.Name,
		Schema:       cfg.DB.Schema,
		MaxIdleConns: 0,
		MaxOpenConns: 0,
		DisableTLS:   cfg.DB.DisableTLS,
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DB.ConnectionTimeout)
	defer cancel()
	if err := db.StatusCheck(ctx, database, l); err != nil {
		return fmt.Errorf("statuscheck: %w", err)
	}

	mgrt := migration.New(cfg.DB.Name, database, l, true)

	switch args[0] {
	case "up":
		return migrateUp(mgrt, l)
	case "seed":
		ctx, cancel := context.WithTimeout(context.Background(), cfg.DB.SeedTimeout)
		defer cancel()
		return mgrt.Seed(ctx)
	default:
		return fmt.Errorf("startMigration: command %s not supported", args[0])
	}
}

func migrateUp(mgrt *migration.Migration, l *zap.SugaredLogger) error {
	migrator, merr := mgrt.Instance()
	if merr != nil {
		return merr
	}
	l.Infoln("initialized migration instance")

	l.Infoln("starting up migration")
	if err := migrator.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			l.Infoln("no change is required to be made")
			return nil
		}
		return err
	}
	l.Infoln("up migration completed successfully")

	v, d, err := migrator.Version()
	if err != nil {
		return fmt.Errorf("migration version: %w", err)
	}
	l.Infow("migration state", "version", v, "dirty", d)
	return nil
}
