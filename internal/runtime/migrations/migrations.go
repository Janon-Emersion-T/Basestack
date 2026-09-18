package migrations

import (
	"context"
	"fmt"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

const lockID int64 = 0x4261736553746163
const metadata = `CREATE SCHEMA IF NOT EXISTS basestack_internal;
CREATE TABLE IF NOT EXISTS basestack_internal.schema_migrations (
 version bigint PRIMARY KEY,
 name text NOT NULL UNIQUE,
 checksum text NOT NULL CHECK (length(checksum) = 64),
 applied_at timestamptz NOT NULL DEFAULT now(),
 down_checksum text NOT NULL DEFAULT ''
);
ALTER TABLE basestack_internal.schema_migrations ADD COLUMN IF NOT EXISTS down_checksum text NOT NULL DEFAULT ''`

type State struct {
	File    File
	Applied bool
}

// Run serializes runners with a PostgreSQL transaction-scoped advisory lock.
// The entire pending batch and its metadata commit atomically; status never creates metadata.
func Run(ctx context.Context, pool *pgxpool.Pool, dir string, apply bool) ([]State, error) {
	return run(ctx, pool, dir, apply, false)
}
func Rollback(ctx context.Context, pool *pgxpool.Pool, dir string) ([]State, error) {
	return run(ctx, pool, dir, false, true)
}
func run(ctx context.Context, pool *pgxpool.Pool, dir string, apply, rollback bool) ([]State, error) {
	files, err := Discover(dir)
	if err != nil {
		return nil, err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, database.SafeError("begin migrations", err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockID); err != nil {
		return nil, database.SafeError("lock migrations", err)
	}
	if apply {
		if _, err = tx.Exec(ctx, metadata); err != nil {
			return nil, database.SafeError("initialize migration metadata", err)
		}
	}
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT to_regclass('basestack_internal.schema_migrations') IS NOT NULL").Scan(&exists); err != nil {
		return nil, database.SafeError("inspect migration metadata", err)
	}
	applied := []File{}
	if exists {
		rows, err := tx.Query(ctx, "SELECT version,name,checksum,COALESCE(to_jsonb(m)->>'down_checksum','') FROM basestack_internal.schema_migrations m ORDER BY version")
		if err != nil {
			return nil, database.SafeError("read migration metadata", err)
		}
		for rows.Next() {
			var f File
			if err := rows.Scan(&f.Version, &f.Name, &f.Checksum, &f.DownChecksum); err != nil {
				rows.Close()
				return nil, database.SafeError("decode migration metadata", err)
			}
			applied = append(applied, f)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, database.SafeError("read migration metadata", err)
		}
	}
	if err := Integrity(files, applied); err != nil {
		return nil, err
	}
	if rollback {
		if len(applied) == 0 {
			return nil, fmt.Errorf("no applied migrations to roll back")
		}
		file := files[len(applied)-1]
		if file.DownSQL == "" {
			return nil, fmt.Errorf("latest migration has no recorded down migration; create a forward migration instead")
		}
		if _, err := tx.Exec(ctx, file.DownSQL, pgx.QueryExecModeSimpleProtocol); err != nil {
			return nil, database.SafeError("roll back migration", err)
		}
		if tx.Conn().PgConn().TxStatus() != 'T' {
			return nil, fmt.Errorf("down migration changed transaction state")
		}
		if _, err := tx.Exec(ctx, "DELETE FROM basestack_internal.schema_migrations WHERE version=$1", file.Version); err != nil {
			return nil, database.SafeError("record rollback", err)
		}
		applied = applied[:len(applied)-1]
	}
	states := make([]State, 0, len(files))
	for i, file := range files {
		done := i < len(applied)
		if apply && !done {
			if _, err := tx.Exec(ctx, file.SQL, pgx.QueryExecModeSimpleProtocol); err != nil {
				return nil, database.SafeError("apply migration "+file.Name, err)
			}
			if tx.Conn().PgConn().TxStatus() != 'T' {
				return nil, fmt.Errorf("migration %s changed transaction state", file.Name)
			}
			if _, err := tx.Exec(ctx, "INSERT INTO basestack_internal.schema_migrations(version,name,checksum,down_checksum) VALUES($1,$2,$3,$4)", file.Version, file.Name, file.Checksum, file.DownChecksum); err != nil {
				return nil, database.SafeError("record migration "+file.Name, err)
			}
			done = true
		}
		states = append(states, State{File: file, Applied: done})
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, database.SafeError("commit migrations", err)
	}
	return states, nil
}
func Integrity(files, applied []File) error {
	if len(applied) > len(files) {
		return fmt.Errorf("migration integrity failure: an applied migration is missing locally")
	}
	for i, record := range applied {
		if record.Version != files[i].Version || record.Name != files[i].Name || record.Checksum != files[i].Checksum || record.DownChecksum != files[i].DownChecksum {
			return fmt.Errorf("migration integrity failure at version %06d: history was changed, removed or reordered", record.Version)
		}
	}
	return nil
}
