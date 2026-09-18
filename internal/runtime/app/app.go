// Package app wires the modular runtime without global mutable application state.
package app

import (
	"context"
	"fmt"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/auth"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/config"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/database"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/migrations"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/server"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"time"
)

func Run(ctx context.Context, dir string, args []string, out io.Writer) error {
	return RunWithDelivery(ctx, dir, args, out, nil)
}

// Production applications inject their reviewed email delivery provider here.
func RunWithDelivery(ctx context.Context, dir string, args []string, out io.Writer, delivery auth.Delivery) error {
	if len(args) == 0 {
		args = []string{"serve"}
	}
	if len(args) != 1 || (args[0] != "serve" && args[0] != "migrate" && args[0] != "status") {
		return fmt.Errorf("usage: server [serve|migrate|status]")
	}
	c, err := config.Load(dir)
	if err != nil {
		return err
	}
	if args[0] == "serve" && !config.Enabled(c.Services.API.Enabled) {
		return fmt.Errorf("API is disabled in basestack/services.json")
	}
	if args[0] != "serve" && !config.Enabled(c.Services.Database.Enabled) {
		return fmt.Errorf("database is disabled in basestack/services.json")
	}
	var pool *pgxpool.Pool
	if config.Enabled(c.Services.Database.Enabled) {
		pool, err = database.Open(ctx, c.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
	}
	if args[0] != "serve" {
		deadline, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		states, err := migrations.Run(deadline, pool, filepath.Join(dir, migrations.Directory), args[0] == "migrate")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, "PostgreSQL connected.")
		for _, state := range states {
			status := "pending"
			if state.Applied {
				status = "applied"
			}
			fmt.Fprintf(out, "%s  %s\n", status, state.File.Name)
		}
		if args[0] == "migrate" {
			fmt.Fprintln(out, "Migrations are up to date.")
		}
		return nil
	}
	var authHandler http.Handler
	if c.Services.Auth != nil && config.Enabled(c.Services.Auth.Enabled) {
		if delivery == nil && c.AuthDelivery == "local" {
			delivery, err = auth.NewLocalDelivery(dir, c.Environment)
			if err != nil {
				return fmt.Errorf("cannot initialize development Auth outbox; ensure private directory permissions")
			}
		}
		if delivery == nil {
			return fmt.Errorf("Auth requires a delivery provider; inject an external provider or explicitly select development/local delivery")
		}
		if c.Environment != "development" && delivery.DevelopmentOnly() {
			return fmt.Errorf("development Auth delivery is forbidden in production")
		}
		repo := auth.NewRepository(pool)
		if err := repo.Ready(ctx); err != nil {
			return fmt.Errorf("Auth tables are unavailable; run basestack db migrate")
		}
		service, err := auth.NewService(repo, auth.Options{RequireVerification: config.Enabled(c.Services.Auth.RequireEmailVerification), Environment: c.Environment, Parameters: c.AuthParameters, VerificationTTL: c.VerificationTTL, ResetTTL: c.ResetTTL}, delivery)
		if err != nil {
			return fmt.Errorf("cannot initialize Auth safely")
		}
		authHandler = auth.Handler(service, auth.NewLimiter())
	}
	addr := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("cannot listen on API host/port; check BASESTACK_API_HOST and BASESTACK_API_PORT")
	}
	var pinger server.Pinger
	if pool != nil {
		pinger = pool
	}
	srv := server.New(addr, server.Handler(server.Options{Database: pinger, Origins: c.Origins, Auth: authHandler}))
	fmt.Fprintf(out, "BaseStack API listening at http://%s (Ctrl+C to stop).\n", listener.Addr())
	return server.Serve(ctx, srv, listener)
}
