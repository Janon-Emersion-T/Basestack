package functions

import (
	"context"
	"errors"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/auth"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/rbac"
	"net/http/httptest"
	"strings"
	"testing"
)

type testSessions struct{}

func (testSessions) SignIn(context.Context, string, string) (auth.Session, error) {
	return auth.Session{}, auth.ErrUnavailable
}
func (testSessions) Logout(context.Context, string) error { return nil }
func (testSessions) Current(_ context.Context, token string) (auth.PublicUser, error) {
	if token != "valid" {
		return auth.PublicUser{}, auth.ErrInvalidCredentials
	}
	return auth.PublicUser{ID: "user"}, nil
}

type checkFunc func(context.Context, string, string) error

func (f checkFunc) Check(ctx context.Context, id, permission string) error {
	return f(ctx, id, permission)
}
func TestRegistryAndProtectedExecution(t *testing.T) {
	called := false
	defs := []Definition{{Metadata: Metadata{Name: "private", Permission: "reports.read"}, Handle: func(ctx context.Context, r Request) (any, error) {
		called = true
		v, _ := r.Environment("VALUE")
		return map[string]string{"value": v, "id": r.User.ID}, nil
	}}}
	r, err := New(defs)
	if err != nil {
		t.Fatal(err)
	}
	if r.List()[0].Name != "private" {
		t.Fatal("discovery failed")
	}
	if _, ok := r.Inspect("absent"); ok {
		t.Fatal("unknown function exists")
	}
	for _, bad := range [][]Definition{append(defs, defs[0]), {{Metadata: Metadata{Name: "../bad", Public: true}, Handle: defs[0].Handle}}, {{Metadata: Metadata{Name: "bad"}, Handle: defs[0].Handle}}, {{Metadata: Metadata{Name: "bad", Public: true, Permission: "permission"}, Handle: defs[0].Handle}}} {
		if _, err := New(bad); err == nil {
			t.Fatal("invalid definition accepted")
		}
	}
	for _, tc := range []struct {
		token      string
		permission error
		want       int
	}{{"", nil, 401}, {"valid", rbac.ErrDenied, 403}, {"valid", errors.New("PRIVATE_ERROR"), 503}, {"valid", nil, 200}} {
		called = false
		handler := r.HTTP(testSessions{}, checkFunc(func(_ context.Context, id, p string) error {
			if id != "user" || p != "reports.read" {
				t.Fatal("wrong permission context")
			}
			return tc.permission
		}), map[string]string{"VALUE": "hello"})
		req := httptest.NewRequest("POST", "/api/functions/private", strings.NewReader(`{}`))
		req.Header.Set("Authorization", "Bearer "+tc.token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != tc.want || called != (tc.want == 200) || strings.Contains(w.Body.String(), "PRIVATE_ERROR") {
			t.Fatal("authorization boundary failed", w.Code)
		}
	}
}
func TestFunctionFailuresAndBodyLimits(t *testing.T) {
	r, _ := New([]Definition{{Metadata: Metadata{Name: "failure", Public: true}, Handle: func(context.Context, Request) (any, error) { return nil, errors.New("PRIVATE_ERROR") }}, {Metadata: Metadata{Name: "panic", Public: true}, Handle: func(context.Context, Request) (any, error) { panic("PRIVATE_PANIC") }}})
	for _, tc := range []struct {
		method, path, body string
		want               int
	}{{"POST", "failure", "{}", 500}, {"POST", "panic", "{}", 500}, {"POST", "unknown", "{}", 404}, {"GET", "failure", "", 405}, {"POST", "failure", "broken", 400}, {"POST", "failure", strings.Repeat("x", (1<<20)+1), 413}} {
		w := httptest.NewRecorder()
		r.HTTP(nil, nil, nil).ServeHTTP(w, httptest.NewRequest(tc.method, "/api/functions/"+tc.path, strings.NewReader(tc.body)))
		if w.Code != tc.want || strings.Contains(w.Body.String(), "PRIVATE") {
			t.Fatal("unsafe function failure", w.Code)
		}
	}
}
