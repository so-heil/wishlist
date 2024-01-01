package apitest

import (
	"context"
	"fmt"
	"time"

	"github.com/so-heil/wishlist/business/database/db"
	"github.com/so-heil/wishlist/business/database/migration"
	"github.com/so-heil/wishlist/foundation/compose"
	"go.uber.org/zap"
)

type Database struct {
	Dbase *db.DB
	cmps  *compose.Compose
}

type DatabaseConfig struct {
	ShouldMigrate  bool
	ShouldSeed     bool
	ConnectTimeout time.Duration
}

var DefaultDatabaseConfig = DatabaseConfig{
	ShouldMigrate:  true,
	ShouldSeed:     true,
	ConnectTimeout: 10 * time.Second,
}

func NewDatabase(config DatabaseConfig, l *zap.SugaredLogger) (*Database, error) {
	const composeYaml = `
services:
  db:
    image: postgres
    ports:
      - 55432:5432
    environment:
      POSTGRES_PASSWORD: postgres
    healthcheck:
      test: ["CMD-SHELL", "pg_isready", "-d", "db_prod"]
      interval: 300ms
      timeout: 60s
      retries: 5
      start_period: 80s  
`

	cmps, err := compose.New(composeYaml)
	if err != nil {
		return nil, fmt.Errorf("db new compose: %w", err)
	}

	containers, err := cmps.Up()
	if err != nil {
		return nil, fmt.Errorf("compose up: %s", err)
	}

	dbContainer := containers["db"]
	dbase, err := db.Open(&db.Config{
		User:       "postgres",
		Password:   "postgres",
		Host:       dbContainer.Host,
		DisableTLS: true,
	}, l)
	if err != nil {
		return nil, fmt.Errorf("open test db: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
	defer cancel()
	if err := dbase.StatusCheck(ctx); err != nil {
		return nil, fmt.Errorf("check db status up: %w", err)
	}

	var mgrt *migration.Migration
	if config.ShouldMigrate {
		mgrt, err = migration.New("", dbase.DB, l, true)
		if err != nil {
			return nil, fmt.Errorf("create migrator: %w", err)
		}

		if err := mgrt.Up(); err != nil {
			return nil, fmt.Errorf("migrate up: %w", err)
		}
	}

	if config.ShouldMigrate && config.ShouldSeed {
		if err := mgrt.Seed(context.Background()); err != nil {
			return nil, fmt.Errorf("seed db: %s", err)
		}
	}

	return &Database{
		Dbase: dbase,
		cmps:  cmps,
	}, nil
}

func (tdb *Database) Close() error {
	var res error
	if err := tdb.Dbase.Close(); err != nil {
		res = err
	}

	if err := tdb.cmps.Close(); err != nil {
		res = fmt.Errorf("%s\n%s", res.Error(), err)
	}

	return res
}
