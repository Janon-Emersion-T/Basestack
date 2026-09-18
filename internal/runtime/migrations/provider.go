package migrations

import (
	"context"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Provider owns database connectivity and migration semantics. Another database
// supplies its own implementation; it need not emulate PostgreSQL SQL or pgx.
type Provider interface {
	Open(context.Context, string) (Service, error)
}
type Service interface {
	Ping(context.Context) error
	Migrate(context.Context, string) ([]State, error)
	Status(context.Context, string) ([]State, error)
	Rollback(context.Context, string) ([]State, error)
	Close()
}
type PostgreSQL struct{}
type PostgresService struct{ Pool *pgxpool.Pool }

func (PostgreSQL) Open(ctx context.Context, dsn string) (Service, error) {
	p, e := database.Open(ctx, dsn)
	if e != nil {
		return nil, e
	}
	return &PostgresService{p}, nil
}
func (p *PostgresService) Ping(ctx context.Context) error { return p.Pool.Ping(ctx) }
func (p *PostgresService) Close()                         { p.Pool.Close() }
func (p *PostgresService) Migrate(ctx context.Context, dir string) ([]State, error) {
	return Run(ctx, p.Pool, dir, true)
}
func (p *PostgresService) Status(ctx context.Context, dir string) ([]State, error) {
	return Run(ctx, p.Pool, dir, false)
}
func (p *PostgresService) Rollback(ctx context.Context, dir string) ([]State, error) {
	return Rollback(ctx, p.Pool, dir)
}
