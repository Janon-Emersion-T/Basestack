// Package app wires the modular runtime without global mutable application state.
package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/auth"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/config"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/functions"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/migrations"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/rbac"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/server"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func Run(ctx context.Context, dir string, args []string, out io.Writer) error {
	return RunWithDelivery(ctx, dir, args, out, nil)
}

// Options injects application-owned providers and compiled function definitions.
type Options struct {
	Secrets   config.SecretStore
	Delivery  auth.Delivery
	Database  migrations.Provider
	Storage   storage.Store
	Functions []functions.Definition
}

func RunWithDelivery(ctx context.Context, dir string, args []string, out io.Writer, delivery auth.Delivery) error {
	return RunWithOptions(ctx, dir, args, out, Options{Delivery: delivery})
}
func RunWithOptions(ctx context.Context, dir string, args []string, out io.Writer, options Options) error {
	if len(args) == 0 {
		args = []string{"serve"}
	}
	command := args[0]
	valid := len(args) == 1 && (command == "serve" || command == "migrate" || command == "status" || command == "rollback")
	if command == "storage" {
		valid = storage.ValidCommand(args[1:])
	}
	if command == "functions" {
		valid = functions.ValidCommand(args[1:])
	}
	if command == "roles" {
		valid = rbac.ValidCommand(args[1:])
	}
	if !valid {
		return fmt.Errorf("invalid runtime command; use serve, migrate, status, rollback, storage, functions or roles")
	}
	delivery := options.Delivery
	c, err := config.LoadWithSecrets(dir, options.Secrets)
	if err != nil {
		return err
	}
	if command == "serve" && !config.Enabled(c.Services.API.Enabled) {
		return fmt.Errorf("API is disabled in basestack/services.json")
	}
	if command == "roles" && (c.Services.Authorization == nil || !config.Enabled(c.Services.Authorization.Enabled)) {
		return fmt.Errorf("authorization is disabled")
	}
	if (command == "migrate" || command == "status" || command == "rollback" || command == "roles") && !config.Enabled(c.Services.Database.Enabled) {
		return fmt.Errorf("database is disabled in basestack/services.json")
	}

	var store storage.Store
	if command == "storage" || command == "serve" {
		if c.Services.Storage != nil && config.Enabled(c.Services.Storage.Enabled) {
			store = options.Storage
			if store == nil {
				store, err = storage.NewLocal(dir, c.Services.Storage.MaxObjectBytes)
				if err != nil {
					return err
				}
			}
		}
		if command == "storage" {
			if store == nil {
				return fmt.Errorf("storage is disabled")
			}
			return storage.Command(ctx, store, args[1:], os.Stdin, out)
		}
	}
	definitions := options.Functions
	if definitions == nil {
		definitions = functions.Examples()
	}
	registry, err := functions.New(definitions)
	if err != nil {
		return err
	}
	if command == "functions" {
		if c.Services.Functions == nil || !config.Enabled(c.Services.Functions.Enabled) {
			return fmt.Errorf("functions are disabled")
		}
		if args[1] != "run" {
			return registry.Describe(args[1:], out)
		}
		if _, ok := registry.Inspect(args[2]); !ok {
			return fmt.Errorf("function not found")
		}
	}
	var pool *pgxpool.Pool
	var db migrations.Service
	if config.Enabled(c.Services.Database.Enabled) {
		provider := options.Database
		if provider == nil {
			provider = migrations.PostgreSQL{}
		}
		db, err = provider.Open(ctx, c.DatabaseURL)
		if err != nil {
			return err
		}
		defer db.Close()
		if pg, ok := db.(*migrations.PostgresService); ok {
			pool = pg.Pool
		}
	}
	if command == "migrate" || command == "status" || command == "rollback" {
		deadline, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		var states []migrations.State
		path := filepath.Join(dir, migrations.Directory)
		switch command {
		case "migrate":
			states, err = db.Migrate(deadline, path)
		case "rollback":
			states, err = db.Rollback(deadline, path)
		default:
			states, err = db.Status(deadline, path)
		}
		if err != nil {
			return err
		}
		fmt.Fprintln(out, "Database connected.")
		for _, state := range states {
			status := "pending"
			if state.Applied {
				status = "applied"
			}
			fmt.Fprintf(out, "%s  %s\n", status, state.File.Name)
		}
		return nil
	}
	var permissions rbac.Authorizer
	if c.Services.Authorization != nil && config.Enabled(c.Services.Authorization.Enabled) {
		if pool == nil {
			return fmt.Errorf("built-in authorization requires PostgreSQL")
		}
		provider := rbac.Postgres{Pool: pool}
		if err := provider.Ready(ctx); err != nil {
			return fmt.Errorf("authorization tables are unavailable; run basestack db migrate")
		}
		permissions = provider
	}
	if command == "roles" {
		if permissions == nil {
			return fmt.Errorf("authorization is disabled")
		}
		deadline, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := rbac.Command(deadline, permissions.(rbac.Manager), args[1:]); err != nil {
			return err
		}
		fmt.Fprintln(out, "Role operation completed.")
		return nil
	}
	var sessions auth.Sessions
	var authHandler http.Handler
	if c.Services.Auth != nil && config.Enabled(c.Services.Auth.Enabled) {
		if pool == nil {
			return fmt.Errorf("built-in Auth requires PostgreSQL")
		}
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
		limiter := auth.NewLimiter()
		authHandler = auth.Handler(service, limiter)
		if c.Services.SchemaVersion >= 3 {
			var ready bool
			if err := pool.QueryRow(ctx, "SELECT to_regclass('basestack_auth.sessions') IS NOT NULL").Scan(&ready); err != nil || !ready {
				return fmt.Errorf("session tables are unavailable; run basestack db migrate")
			}
			sessions = auth.PostgresSessions{Service: service}
			authHandler = auth.SessionHandler(authHandler, sessions, limiter)
		}
	}

	values, err := config.EnvironmentWithStore(dir, options.Secrets)
	if err != nil {
		return err
	}
	var functionHandler http.Handler
	if c.Services.Functions != nil && config.Enabled(c.Services.Functions.Enabled) {
		functionHandler = registry.HTTP(sessions, permissions, values)
	}
	if command == "functions" {
		var input io.Reader = http.NoBody
		if info, err := os.Stdin.Stat(); err == nil && info.Mode()&os.ModeCharDevice == 0 {
			input = io.LimitReader(os.Stdin, (1<<20)+1)
		}
		req := httptest.NewRequest("POST", "/api/functions/"+args[2], input).WithContext(ctx)
		if token := values["BASESTACK_FUNCTION_TOKEN"]; token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		response := httptest.NewRecorder()
		functionHandler.ServeHTTP(response, req)
		if response.Code >= 400 {
			return errors.New("function execution failed; check input, session and permission")
		}
		_, err = out.Write(response.Body.Bytes())
		return err
	}
	var storageHandler http.Handler
	if store != nil {
		storageHandler = storage.HTTP(store, sessions, permissions, c.Services.Storage.MaxObjectBytes)
	}
	addr := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("cannot listen on API host/port; check BASESTACK_API_HOST and BASESTACK_API_PORT")
	}
	var pinger server.Pinger
	if db != nil {
		pinger = db
	}
	srv := server.New(addr, server.Handler(server.Options{Database: pinger, Origins: c.Origins, Auth: authHandler, Functions: functionHandler, Storage: storageHandler}))
	fmt.Fprintf(out, "BaseStack API listening at http://%s (Ctrl+C to stop).\n", listener.Addr())
	return server.Serve(ctx, srv, listener)
}
