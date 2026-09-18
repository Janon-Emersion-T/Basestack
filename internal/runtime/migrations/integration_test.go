package migrations

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/server"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// The opt-in URL must point at a disposable PostgreSQL instance with CREATEDB permission.
// Every test creates and drops only its own randomly named database.
func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("BASESTACK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set BASESTACK_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal("cannot connect to integration PostgreSQL (details omitted)")
	}
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		t.Fatal(err)
	}
	name := "basestack_test_" + hex.EncodeToString(random)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		admin.Close(ctx)
		t.Fatal("cannot create isolated integration database; test role requires CREATEDB")
	}
	var pool *pgxpool.Pool
	t.Cleanup(func() {
		if pool != nil {
			pool.Close()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
			t.Error("cannot remove owned integration database")
		}
		admin.Close(ctx)
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid integration DSN")
	}
	cfg.ConnConfig.Database = name
	pool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal("cannot create integration pool")
	}
	if pool.Ping(ctx) != nil {
		t.Fatal("cannot ping integration database")
	}
	return pool
}
func TestPostgreSQLMigrationLifecycle(t *testing.T) {
	pool := integrationPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dir := t.TempDir()
	first := "CREATE TABLE example (id integer PRIMARY KEY); INSERT INTO example VALUES (1);"
	sqlFile(t, dir, "000001_first.sql", first)
	states, err := Run(ctx, pool, dir, false)
	if err != nil || len(states) != 1 || states[0].Applied {
		t.Fatal("initial status failed", err)
	}
	var exists bool
	pool.QueryRow(ctx, "SELECT to_regclass('basestack_internal.schema_migrations') IS NOT NULL").Scan(&exists)
	if exists {
		t.Fatal("status mutated metadata")
	}
	for i := 0; i < 2; i++ {
		states, err = Run(ctx, pool, dir, true)
		if err != nil || !states[0].Applied {
			t.Fatal("apply/idempotence failed", err)
		}
	}
	count := func(query string, want int) {
		t.Helper()
		var got int
		if err := pool.QueryRow(ctx, query).Scan(&got); err != nil {
			t.Fatal("count query failed")
		}
		if got != want {
			t.Fatalf("count got %d want %d", got, want)
		}
	}
	count("SELECT count(*) FROM example", 1)
	count("SELECT count(*) FROM basestack_internal.schema_migrations", 1)
	sqlFile(t, dir, "000002_second.sql", "INSERT INTO example VALUES (2);")
	sqlFile(t, dir, "000003_third.sql", "INSERT INTO example VALUES (1);")
	if _, err := Run(ctx, pool, dir, true); err == nil {
		t.Fatal("failed SQL accepted")
	}
	count("SELECT count(*) FROM example", 1)
	count("SELECT count(*) FROM basestack_internal.schema_migrations", 1)
	sqlFile(t, dir, "000003_third.sql", "INSERT INTO example VALUES (3);")
	if _, err := Run(ctx, pool, dir, true); err != nil {
		t.Fatal(err)
	}
	count("SELECT count(*) FROM example", 3)
	sqlFile(t, dir, "000001_first.sql", first+" -- modified")
	if _, err := Run(ctx, pool, dir, false); err == nil {
		t.Fatal("modified history accepted")
	}
	sqlFile(t, dir, "000001_first.sql", first)
	second, err := os.ReadFile(filepath.Join(dir, "000002_second.sql"))
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Join(dir, "000002_second.sql"))
	if _, err := Run(ctx, pool, dir, true); err == nil {
		t.Fatal("missing history accepted")
	}
	sqlFile(t, dir, "000002_second.sql", string(second))
	sqlFile(t, dir, "000004_fourth.sql", "INSERT INTO example VALUES (4);")
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := Run(ctx, pool, dir, true); results <- err }()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	count("SELECT count(*) FROM example", 4)
	// Lock waits honor context cancellation and do not leave a session lock behind.
	lock, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal("lock test begin failed")
	}
	if _, err := lock.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockID); err != nil {
		t.Fatal("lock setup failed")
	}
	short, stop := context.WithTimeout(ctx, 50*time.Millisecond)
	_, err = Run(short, pool, dir, false)
	stop()
	lock.Rollback(ctx)
	if err == nil {
		t.Fatal("lock wait ignored cancellation")
	}
	if _, err := Run(ctx, pool, dir, false); err != nil {
		t.Fatal("lock not released", err)
	}
	w := httptest.NewRecorder()
	server.Handler(server.Options{Database: pool}).ServeHTTP(w, httptest.NewRequest("GET", "/api/health", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"database":"connected"`) {
		t.Fatal("health against real PostgreSQL failed")
	}
}
