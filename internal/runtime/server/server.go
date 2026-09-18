// Package server is the HTTP boundary of the application runtime.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const Version = "0.3.1"

type Pinger interface{ Ping(context.Context) error }
type Options struct {
	Auth     http.Handler
	Origins  []string
	Database Pinger
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func failure(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func Handler(options Options) http.Handler {
	origins := map[string]bool{}
	for _, origin := range options.Origins {
		origins[origin] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		w.Header().Add("Vary", "Origin")
		defer func() {
			if recover() != nil {
				failure(w, 500, "internal_error", "Internal server error")
			}
		}()
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !origins[origin] {
				failure(w, 403, "origin_denied", "Origin is not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		isAuth := options.Auth != nil && strings.HasPrefix(r.URL.Path, "/api/auth/")
		if r.URL.Path != "/api/health" && !isAuth {
			failure(w, 404, "not_found", "Route not found")
			return
		}
		expectedMethod := "GET"
		if isAuth {
			expectedMethod = "POST"
		}
		if r.Method == http.MethodOptions {
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")
			if origin == "" || r.Header.Get("Access-Control-Request-Method") != expectedMethod || (r.Header.Get("Access-Control-Request-Headers") != "" && (!isAuth || strings.ToLower(strings.TrimSpace(r.Header.Get("Access-Control-Request-Headers"))) != "content-type")) {
				failure(w, 403, "preflight_denied", "Preflight is not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", expectedMethod)
			if isAuth {
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if isAuth {
			options.Auth.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET, OPTIONS")
			failure(w, 405, "method_not_allowed", "Method not allowed")
			return
		}
		status, db, httpStatus := "ok", "disabled", http.StatusOK
		if options.Database != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := options.Database.Ping(ctx); err != nil {
				status, db, httpStatus = "error", "unavailable", http.StatusServiceUnavailable
			} else {
				db = "connected"
			}
		}
		writeJSON(w, httpStatus, struct {
			Status   string `json:"status"`
			Service  string `json:"service"`
			Version  string `json:"version"`
			Database string `json:"database"`
		}{status, "basestack", Version, db})
	})
}
func New(addr string, handler http.Handler) *http.Server {
	return &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10, ErrorLog: log.New(io.Discard, "", 0)}
}

// Serve accepts an owned listener for deterministic tests and supports bounded graceful shutdown.
func Serve(ctx context.Context, srv *http.Server, listener net.Listener) error {
	srv.BaseContext = func(net.Listener) context.Context { return ctx }
	done := make(chan error, 1)
	go func() { done <- srv.Serve(listener) }()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("HTTP server failed")
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdown); err != nil {
			_ = srv.Close()
			<-done
			return fmt.Errorf("HTTP shutdown timed out")
		}
		err := <-done
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server failed")
		}
		return nil
	}
}
