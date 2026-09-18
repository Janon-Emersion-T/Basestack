// Package database exposes a normal pgx pool; application code can issue PostgreSQL queries directly.
package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

func Open(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, fmt.Errorf("database is not configured; run basestack services start or set BASESTACK_DATABASE_URL")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid BASESTACK_DATABASE_URL (connection details omitted)")
	}
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	cfg.MaxConns = 10
	cfg.MinConns = 0
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.MaxConnLifetime = time.Hour
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, SafeError("create database pool", err)
	}
	check, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(check); err != nil {
		pool.Close()
		return nil, SafeError("connect to PostgreSQL; check services status and BASESTACK_DATABASE_URL", err)
	}
	return pool, nil
}

// Never include driver error text: it can contain the DSN, SQL or database values.
func SafeError(operation string, err error) error {
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s: canceled", operation)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s: timed out", operation)
	}
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) {
		return fmt.Errorf("%s: PostgreSQL SQLSTATE %s (details omitted)", operation, pgerr.Code)
	}
	return fmt.Errorf("%s: database unavailable or operation failed (connection details omitted)", operation)
}
