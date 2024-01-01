package migration

import (
	"context"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type log struct {
	l       *zap.SugaredLogger
	verbose bool
}

func (log log) Printf(format string, v ...any) {
	log.l.Infof(format, v)
}

func (log log) Verbose() bool {
	return log.verbose
}

type Migration struct {
	dbName  string
	db      *sqlx.DB
	l       *zap.SugaredLogger
	verbose bool
	*migrate.Migrate
}

func New(dbName string, db *sqlx.DB, l *zap.SugaredLogger, verbose bool) (*Migration, error) {
	ddriver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("open postgres db: %w", err)
	}

	sdriver, err := iofs.New(fsys, "sql")
	if err != nil {
		return nil, fmt.Errorf("open source file system: %w", err)
	}

	migrator, err := migrate.NewWithInstance("migration", sdriver, dbName, ddriver)
	migrator.Log = log{
		l:       l,
		verbose: verbose,
	}

	if err != nil {
		return nil, fmt.Errorf("construct migrator: %w", err)
	}

	return &Migration{
		dbName:  dbName,
		db:      db,
		l:       l,
		verbose: verbose,
		Migrate: migrator,
	}, nil
}

//go:embed sql
var fsys embed.FS

//go:embed seed.sql
var seed string

//go:embed seeded-test.sql
var seededTest string

func (m *Migration) Seed(ctx context.Context) error {
	m.l.Infoln("testing if seed has happened before")
	var seeded bool
	if err := m.db.QueryRowContext(ctx, seededTest).Scan(&seeded); err != nil {
		return fmt.Errorf("test seeded: %w", err)
	}

	if seeded {
		m.l.Infoln("seed happened before, skipping")
		return nil
	}

	m.l.Infoln("starting seed")
	res, err := m.db.ExecContext(ctx, seed)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	m.l.Infow("seed completed", "rowsAffected", n)
	return nil
}
