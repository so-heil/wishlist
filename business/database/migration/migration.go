package migration

import (
	"context"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	iofs "github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type Migration struct {
	dbName  string
	db      *sqlx.DB
	l       *zap.SugaredLogger
	verbose bool
}

func New(dbName string, db *sqlx.DB, l *zap.SugaredLogger, verbose bool) *Migration {
	return &Migration{
		dbName:  dbName,
		db:      db,
		l:       l,
		verbose: verbose,
	}
}

type Log struct {
	l       *zap.SugaredLogger
	verbose bool
}

func (log Log) Printf(format string, v ...any) {
	log.l.Infof(format, v)
}

func (log Log) Verbose() bool {
	return log.verbose
}

//go:embed sql
var fsys embed.FS

func (m *Migration) Instance() (*migrate.Migrate, error) {
	ddriver, err := postgres.WithInstance(m.db.DB, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("open postgres db: %w", err)
	}

	sdriver, err := iofs.New(fsys, "sql")
	if err != nil {
		return nil, fmt.Errorf("open source file system: %w", err)
	}

	migrator, err := migrate.NewWithInstance("migration", sdriver, m.dbName, ddriver)
	migrator.Log = Log{
		l:       m.l,
		verbose: m.verbose,
	}

	if err != nil {
		return nil, fmt.Errorf("construct migrator: %w", err)
	}

	return migrator, nil
}

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
