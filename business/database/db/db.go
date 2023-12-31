package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
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

type DB struct {
	*sqlx.DB
	log *zap.SugaredLogger
}

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

func Open(cfg *Config, log *zap.SugaredLogger) (*DB, error) {
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

	return &DB{DB: database, log: log}, nil
}

func (dbase *DB) StatusCheck(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Second)
		defer cancel()
	}

	var pingError error
	for attempts := 1; ; attempts++ {
		pingError = dbase.Ping()
		if pingError == nil {
			dbase.log.Infow("ping successful", "on attempt", attempts)
			break
		}
		wait := time.Duration(attempts) * 100 * time.Millisecond
		dbase.log.Infow("ping error", "error", pingError.Error(), "on attempt", attempts, "waiting", wait.String())
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
	return dbase.QueryRowContext(ctx, q).Scan(&tmp)
}

// NamedExecContext executes a named query against the provided data
func (dbase *DB) NamedExecContext(ctx context.Context, query string, data any) (err error) {
	ctx, span := web.AddSpan(ctx, "business.database.exec", attribute.String("query", query))
	defer span.End()

	if _, err := sqlx.NamedExecContext(ctx, dbase, query, data); err != nil {
		if pqerr, ok := err.(*pgconn.PgError); ok {
			switch pqerr.Code {
			case undefinedTable:
				return ErrUndefinedTable
			case uniqueViolation:
				return ErrDBDuplicatedEntry
			}
		}
		return fmt.Errorf("db.NamedExecContext: %w", err)
	}

	return nil
}

// NamedQueryStruct sends the query and tries to retrieve a single row into the dest struct
func (dbase *DB) NamedQueryStruct(ctx context.Context, query string, data, dest any) error {
	return dbase.namedQueryStruct(ctx, query, data, dest)
}

// NamedQueryStructUpdate sends the query against the data and updates the data itself
func (dbase *DB) NamedQueryStructUpdate(ctx context.Context, query string, data any) error {
	return dbase.namedQueryStruct(ctx, query, data, data)
}

func (dbase *DB) namedQueryStruct(ctx context.Context, query string, data, dest any) (err error) {
	q := queryString(query, data)

	defer func() {
		if err != nil {
			dbase.log.Infow("database.NamedQuerySlice", "query", q, "ERROR", err)
		}
	}()

	ctx, span := web.AddSpan(ctx, "business.database.query", attribute.String("query", q))
	defer span.End()

	rows, err := sqlx.NamedQueryContext(ctx, dbase, query, data)
	if err != nil {
		if pqerr, ok := err.(*pgconn.PgError); ok && pqerr.Code == undefinedTable {
			return ErrUndefinedTable
		}
		return fmt.Errorf("NamedQueryContext: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return ErrDBNotFound
	}

	if err := rows.StructScan(dest); err != nil {
		return fmt.Errorf("struct scan: %w", err)
	}

	return nil
}

func queryString(query string, args any) string {
	query, params, err := sqlx.Named(query, args)
	if err != nil {
		return err.Error()
	}

	for _, param := range params {
		var value string
		switch v := param.(type) {
		case string:
			value = fmt.Sprintf("'%s'", v)
		case []byte:
			value = fmt.Sprintf("'%s'", string(v))
		default:
			value = fmt.Sprintf("%v", v)
		}
		query = strings.Replace(query, "?", value, 1)
	}

	query = strings.ReplaceAll(query, "\t", "")
	query = strings.ReplaceAll(query, "\n", " ")

	return strings.Trim(query, " ")
}
