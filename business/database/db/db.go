package db

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/so-heil/wishlist/foundation/web"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

const (
	uniqueViolation = "23505"
	undefinedTable  = "42P01"
)

var (
	ErrDBNotFound        = sql.ErrNoRows
	ErrDBDuplicatedEntry = errors.New("duplicated entry")
	ErrUndefinedTable    = errors.New("undefined table")
)

type Config struct {
	User         string
	Password     string
	Host         string
	Name         string
	Schema       string
	MaxIdleConns int
	MaxOpenConns int
	DisableTLS   bool
}

func Open(cfg *Config) (*sqlx.DB, error) {
	sslMode := "require"
	if cfg.DisableTLS {
		sslMode = "disable"
	}

	q := make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")
	if cfg.Schema != "" {
		q.Set("search_path", cfg.Schema)
	}

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     cfg.Host,
		Path:     cfg.Name,
		RawQuery: q.Encode(),
	}

	database, err := sqlx.Open("postgres", u.String())
	if err != nil {
		return nil, err
	}

	database.SetMaxIdleConns(cfg.MaxIdleConns)
	database.SetMaxOpenConns(cfg.MaxOpenConns)

	return database, nil
}

func StatusCheck(ctx context.Context, db *sqlx.DB, l *zap.SugaredLogger) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Second)
		defer cancel()
	}

	var pingError error
	for attempts := 1; ; attempts++ {
		pingError = db.Ping()
		if pingError == nil {
			l.Infow("ping successful", "on attempt", attempts)
			break
		}
		wait := time.Duration(attempts) * 100 * time.Millisecond
		l.Infow("ping error", "error", pingError.Error(), "on attempt", attempts, "waiting", wait.String())
		time.Sleep(wait)
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	const q = `SELECT true`
	var tmp bool
	return db.QueryRowContext(ctx, q).Scan(&tmp)
}

func NamedExecContext(ctx context.Context, l *zap.SugaredLogger, db sqlx.ExtContext, query string, data any) (err error) {
	ctx, span := web.AddSpan(ctx, "business.database.exec", attribute.String("query", query))
	defer span.End()

	if _, err := sqlx.NamedExecContext(ctx, db, query, data); err != nil {
		l.Infow("database.NamedExecContext", "query", query, "ERROR", err)
		if pqerr, ok := err.(*pgconn.PgError); ok {
			switch pqerr.Code {
			case undefinedTable:
				return ErrUndefinedTable
			case uniqueViolation:
				return ErrDBDuplicatedEntry
			}
		}
		return err
	}

	return nil
}
