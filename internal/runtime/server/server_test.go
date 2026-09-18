package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type pingFunc func(context.Context) error

func (f pingFunc) Ping(ctx context.Context) error { return f(ctx) }
func TestHealth(t *testing.T) {
	for _, tc := range []struct {
		name   string
		db     Pinger
		status int
		value  string
	}{{"connected", pingFunc(func(context.Context) error { return nil }), 200, "connected"}, {"unavailable", pingFunc(func(context.Context) error { return errors.New("postgres://SECRET") }), 503, "unavailable"}, {"disabled", nil, 200, "disabled"}} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Handler(Options{Database: tc.db}).ServeHTTP(w, httptest.NewRequest("GET", "/api/health", nil))
			if w.Code != tc.status || strings.Contains(w.Body.String(), "SECRET") {
				t.Fatalf("bad health: %s", w.Body.String())
			}
			var body map[string]string
			json.Unmarshal(w.Body.Bytes(), &body)
			if body["database"] != tc.value || body["service"] != "basestack" || body["version"] != Version {
				t.Fatal(body)
			}
			if w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("security headers missing")
			}
		})
	}
}
func TestCORSAndErrors(t *testing.T) {
	handler := Handler(Options{Origins: []string{"http://localhost:5173"}})
	for _, tc := range []struct {
		method, path, origin, preflight string
		code                            int
	}{{"GET", "/api/health", "http://localhost:5173", "", 200}, {"GET", "/api/health", "https://evil.test", "", 403}, {"OPTIONS", "/api/health", "http://localhost:5173", "GET", 204}, {"OPTIONS", "/api/health", "http://localhost:5173", "POST", 403}, {"POST", "/api/health", "", "", 405}, {"GET", "/other", "", "", 404}} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Origin", tc.origin)
		req.Header.Set("Access-Control-Request-Method", tc.preflight)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != tc.code {
			t.Fatalf("%+v got %d", tc, w.Code)
		}
		if w.Header().Get("Access-Control-Allow-Credentials") != "" || w.Header().Get("Access-Control-Allow-Origin") == "*" {
			t.Fatal("unsafe CORS")
		}
	}
}
func TestServeAndShutdown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := New(listener.Addr().String(), Handler(Options{}))
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, srv, listener) }()
	client := http.Client{Timeout: time.Second}
	resp, err := client.Get("http://" + listener.Addr().String() + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	client.CloseIdleConnections()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown hung")
	}
	if conn, err := net.DialTimeout("tcp", listener.Addr().String(), 100*time.Millisecond); err == nil {
		conn.Close()
		t.Fatal("listener still open")
	}
	if srv.ReadHeaderTimeout == 0 || srv.IdleTimeout == 0 || srv.MaxHeaderBytes == 0 {
		t.Fatal("HTTP limits missing")
	}
}
func TestCancellationReachesQuery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	handler := Handler(Options{Database: pingFunc(func(ctx context.Context) error { called = true; return ctx.Err() })})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/health", nil).WithContext(ctx))
	if !called || w.Code != 503 {
		t.Fatal("request context not passed to query")
	}
}

func TestAuthCORSRouting(t *testing.T) {
	called := false
	handler := Handler(Options{Origins: []string{"http://localhost:5173"}, Auth: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(201) })})
	req := httptest.NewRequest("OPTIONS", "/api/auth/signup", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 204 || w.Header().Get("Access-Control-Allow-Headers") != "Content-Type" || called {
		t.Fatal("Auth preflight failed")
	}
	req = httptest.NewRequest("POST", "/api/auth/signup", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 201 || !called {
		t.Fatal("Auth route not delegated")
	}
	w = httptest.NewRecorder()
	Handler(Options{}).ServeHTTP(w, req)
	if w.Code != 404 {
		t.Fatal("disabled Auth exposed routes")
	}
}

func TestApplicationCORSAndDisabledRoutes(t *testing.T) {
	for _, path := range []string{"/api/functions/hello", "/api/storage/files", "/api/auth/me"} {
		w := httptest.NewRecorder()
		Handler(Options{}).ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 404 {
			t.Fatal("disabled route exposed")
		}
	}
	noop := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	h := Handler(Options{Origins: []string{"http://localhost:5173"}, Functions: noop, Storage: noop, Auth: noop})
	for _, tc := range []struct {
		path, method, headers string
		status                int
	}{{"/api/functions/hello", "POST", "authorization, content-type", 204}, {"/api/storage/files", "DELETE", "authorization", 204}, {"/api/storage/files", "PATCH", "authorization", 403}, {"/api/auth/me", "GET", "authorization", 204}, {"/api/functions/hello", "POST", "x-user-id", 403}} {
		r := httptest.NewRequest("OPTIONS", tc.path, nil)
		r.Header.Set("Origin", "http://localhost:5173")
		r.Header.Set("Access-Control-Request-Method", tc.method)
		r.Header.Set("Access-Control-Request-Headers", tc.headers)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("preflight status %d want %d", w.Code, tc.status)
		}
	}
}
