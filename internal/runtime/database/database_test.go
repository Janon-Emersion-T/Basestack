package database

import (
	"context"
	"github.com/jackc/pgx/v5/pgconn"
	"net"
	"strings"
	"testing"
	"time"
)

func TestUnavailableAndRedacted(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()
	for _, dsn := range []string{"", "postgres://private:DO_NOT_PRINT@%invalid", "postgres://private:DO_NOT_PRINT@" + addr + "/missing?sslmode=disable"} {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		pool, err := Open(ctx, dsn)
		cancel()
		if err == nil {
			pool.Close()
			t.Fatal("invalid/unavailable DB accepted")
		}
		if strings.Contains(err.Error(), "DO_NOT_PRINT") || strings.Contains(err.Error(), "private") {
			t.Fatal("connection leaked")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if pool, err := Open(ctx, "postgres://private:DO_NOT_PRINT@"+addr+"/missing"); err == nil {
		pool.Close()
		t.Fatal("cancellation ignored")
	}
	err = SafeError("query", &pgconn.PgError{Code: "23505", Message: "DO_NOT_PRINT", Detail: "private data"})
	if !strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "DO_NOT_PRINT") {
		t.Fatal("unsafe driver error")
	}
}
