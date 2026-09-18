package auth

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type captureDelivery struct {
	mu       sync.Mutex
	messages []Message
}

func (d *captureDelivery) DevelopmentOnly() bool { return true }
func (d *captureDelivery) Deliver(_ context.Context, m Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = append(d.messages, m)
	return nil
}
func (d *captureDelivery) last(t *testing.T, email, purpose string) Message {
	t.Helper()
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := len(d.messages) - 1; i >= 0; i-- {
		m := d.messages[i]
		if m.Email == email && m.Purpose == purpose {
			return m
		}
	}
	t.Fatal("expected delivery missing")
	return Message{}
}
func authDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("BASESTACK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set BASESTACK_TEST_DATABASE_URL for real PostgreSQL Auth integration tests")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal("integration PostgreSQL unavailable")
	}
	id, _ := UUID()
	name := "basestack_auth_test_" + strings.ReplaceAll(id, "-", "")
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		admin.Close(ctx)
		t.Fatal("test database creation failed")
	}
	var pool *pgxpool.Pool
	t.Cleanup(func() {
		if pool != nil {
			pool.Close()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
			t.Error("test database cleanup failed")
		}
		admin.Close(ctx)
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid test DSN")
	}
	cfg.ConnConfig.Database = name
	pool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal("test pool failed")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "000001_auth.sql"), []byte(SchemaSQL), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := migrations.Run(ctx, pool, dir, true); err != nil {
		t.Fatal("Auth migration failed", err)
	}
	return pool
}
func setupAuth(t *testing.T) (*Service, *pgxpool.Pool, *captureDelivery) {
	t.Helper()
	pool := authDB(t)
	delivery := &captureDelivery{}
	service, err := NewService(NewRepository(pool), Options{RequireVerification: true, Environment: "development", VerificationTTL: time.Hour, ResetTTL: 30 * time.Minute, Parameters: smallParameters()}, delivery)
	if err != nil {
		t.Fatal(err)
	}
	return service, pool, delivery
}
func TestPostgreSQLAuthLifecycle(t *testing.T) {
	s, pool, delivery := setupAuth(t)
	ctx := context.Background()
	password := "correct horse battery staple"
	user, err := s.Signup(ctx, "User@example.com", password)
	if err != nil || user.Status != "unverified" || user.EmailVerified {
		t.Fatal("signup failed", err)
	}
	if _, err := s.Signup(ctx, "USER@EXAMPLE.COM", password); !errors.Is(err, ErrSignupUnavailable) {
		t.Fatal("duplicate signup accepted")
	}
	if _, err := s.Login(ctx, "user@example.com", password); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("unverified login accepted")
	}
	verify := delivery.last(t, "User@example.com", VerifyEmail)
	verified, err := s.VerifyEmail(ctx, verify.Token)
	if err != nil || !verified.EmailVerified || verified.Status != "active" {
		t.Fatal("verification failed", err)
	}
	if _, err := s.VerifyEmail(ctx, verify.Token); !errors.Is(err, ErrTokenInvalid) {
		t.Fatal("verification reuse accepted")
	}
	if _, err := s.VerifyEmail(ctx, "invalid"); !errors.Is(err, ErrTokenInvalid) {
		t.Fatal("invalid token accepted")
	}
	if _, err := s.Login(ctx, "USER@example.com", password); err != nil {
		t.Fatal("login failed", err)
	}
	for _, email := range []string{"user@example.com", "missing@example.com"} {
		if _, err := s.Login(ctx, email, "incorrect password value"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatal("login error was not generic")
		}
	}
	if err := s.RequestChallenge(ctx, "user@example.com", ResetPassword); err != nil {
		t.Fatal(err)
	}
	reset := delivery.last(t, "User@example.com", ResetPassword)
	if err := s.RequestChallenge(ctx, "missing@example.com", ResetPassword); err != nil {
		t.Fatal("missing account response failed")
	}
	if err := s.ResetPassword(ctx, reset.Token, "a different secure passphrase"); err != nil {
		t.Fatal("reset failed", err)
	}
	if err := s.ResetPassword(ctx, reset.Token, password); !errors.Is(err, ErrTokenInvalid) {
		t.Fatal("reset reuse accepted")
	}
	if _, err := s.Login(ctx, "user@example.com", password); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("old password accepted")
	}
	if _, err := s.Login(ctx, "user@example.com", "a different secure passphrase"); err != nil {
		t.Fatal("new password rejected")
	}
	// Expiration is checked by PostgreSQL at consumption, not just by the application clock.
	s.RequestChallenge(ctx, "user@example.com", ResetPassword)
	expired := delivery.last(t, "User@example.com", ResetPassword)
	digest, _ := TokenDigest(expired.Token)
	if _, err := pool.Exec(ctx, `UPDATE basestack_auth.challenges SET created_at=clock_timestamp()-interval '2 hours',expires_at=clock_timestamp()-interval '1 hour' WHERE digest=$1`, digest); err != nil {
		t.Fatal("expire fixture failed")
	}
	if err := s.ResetPassword(ctx, expired.Token, password); !errors.Is(err, ErrTokenInvalid) {
		t.Fatal("expired reset accepted")
	}
	if err := s.Suspend(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Login(ctx, "user@example.com", "a different secure passphrase"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("suspended user logged in")
	}
	if err := s.Disable(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Login(ctx, "user@example.com", "a different secure passphrase"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("disabled user logged in")
	}
	if err := s.UpdatePassword(ctx, user.ID, "admin supplied secure passphrase"); err != nil {
		t.Fatal("internal password update failed")
	}
	if _, err := s.FindByID(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	var events int
	if err := pool.QueryRow(ctx, `SELECT count(DISTINCT type) FROM basestack_auth.events`).Scan(&events); err != nil || events < 9 {
		t.Fatal("security events missing")
	}
	var stored string
	if err := pool.QueryRow(ctx, `SELECT password_hash FROM basestack_auth.users WHERE id=$1`, user.ID).Scan(&stored); err != nil || !strings.HasPrefix(stored, "$argon2id$") {
		t.Fatal("password not hashed")
	}
	if _, err := pool.Exec(ctx, `UPDATE basestack_auth.users SET status='owner' WHERE id=$1`, user.ID); err == nil {
		t.Fatal("status constraint missing")
	}
	if _, err := pool.Exec(ctx, `UPDATE basestack_auth.users SET email_normalized='other@example.com' WHERE id=$1`, user.ID); err == nil {
		t.Fatal("normalization constraint missing")
	}
	// Concurrent duplicate signup: exactly one committed user and one creation event.
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := s.Signup(ctx, "race@example.com", password); results <- err }()
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrSignupUnavailable) {
			conflict++
		} else {
			t.Fatal("unexpected signup error", err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal("duplicate signup race")
	}
	raceVerify := delivery.last(t, "race@example.com", VerifyEmail)
	digest, _ = TokenDigest(raceVerify.Token)
	pool.Exec(ctx, `UPDATE basestack_auth.challenges SET created_at=clock_timestamp()-interval '2 hours',expires_at=clock_timestamp()-interval '1 hour' WHERE digest=$1`, digest)
	if _, err := s.VerifyEmail(ctx, raceVerify.Token); !errors.Is(err, ErrTokenInvalid) {
		t.Fatal("expired verification accepted")
	}
}
func TestPostgreSQLAuthHTTP(t *testing.T) {
	s, _, delivery := setupAuth(t)
	handler := Handler(s, NewLimiter())
	password := "correct horse battery staple"
	request := func(path, body, ip string) (int, string) {
		t.Helper()
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r.RemoteAddr = ip + ":1234"
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		response := w.Body.String()
		if strings.Contains(response, password) || strings.Contains(response, "$argon2") || strings.Contains(response, "digest") {
			t.Fatal("sensitive response")
		}
		return w.Code, response
	}
	data, _ := json.Marshal(map[string]string{"email": "http@example.com", "password": password})
	status, response := request("/api/auth/signup", string(data), "127.0.0.1")
	if status != 201 || !strings.Contains(response, `"user"`) {
		t.Fatal("HTTP signup failed", status, response)
	}
	for _, body := range []string{`{"email":"x@example.com","password":"short"}`, `{"email":"x@example.com","password":"correct horse battery staple","extra":true}`, `{"email":"x@example.com","email":"y@example.com"}`} {
		if code, _ := request("/api/auth/signup", body, "127.0.0.2"); code != 400 {
			t.Fatal("bad input accepted")
		}
	}
	if code, _ := request("/api/auth/signup", strings.Repeat("a", 8193), "127.0.0.3"); code != 413 {
		t.Fatal("body limit missing")
	}
	v := delivery.last(t, "http@example.com", VerifyEmail)
	tokenBody, _ := json.Marshal(map[string]string{"token": v.Token})
	if code, _ := request("/api/auth/verify-email", string(tokenBody), "127.0.0.1"); code != 200 {
		t.Fatal("HTTP verify failed")
	}
	if code, body := request("/api/auth/login", string(data), "127.0.0.1"); code != 200 || !strings.Contains(body, `"sessionIssued":false`) {
		t.Fatal("login contract incorrect")
	}
	existsCode, exists := request("/api/auth/password/forgot", `{"email":"http@example.com"}`, "127.0.0.4")
	missingCode, missing := request("/api/auth/password/forgot", `{"email":"missing@example.com"}`, "127.0.0.4")
	if existsCode != 202 || existsCode != missingCode || exists != missing {
		t.Fatal("reset request enumerates accounts")
	}
	for i := 0; i < 10; i++ {
		request("/api/auth/login", `{"email":"missing@example.com","password":"wrong password value"}`, "127.0.0.5")
	}
	if code, _ := request("/api/auth/login", string(data), "127.0.0.5"); code != 429 {
		t.Fatal("HTTP limiter missing")
	}
	r := httptest.NewRequest("POST", "/api/auth/signup", io.NopCloser(strings.NewReader("{}")))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatal("JSON content type not required")
	}
}

func TestPostgreSQLAuthConcurrencyAndPolicy(t *testing.T) {
	s, pool, delivery := setupAuth(t)
	ctx := context.Background()
	password := "a long concurrency test passphrase"
	user, err := s.Signup(ctx, "tokens@example.com", password)
	if err != nil {
		t.Fatal(err)
	}
	token := delivery.last(t, "tokens@example.com", VerifyEmail).Token
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := s.VerifyEmail(ctx, token); results <- err }()
	}
	wg.Wait()
	close(results)
	success, invalid := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrTokenInvalid) {
			invalid++
		} else {
			t.Fatal("verification race failed", err)
		}
	}
	if success != 1 || invalid != 1 {
		t.Fatal("token was not atomically single-use")
	}
	if err := s.RequestChallenge(ctx, "tokens@example.com", ResetPassword); err != nil {
		t.Fatal(err)
	}
	reset := delivery.last(t, "tokens@example.com", ResetPassword).Token
	if _, err := s.VerifyEmail(ctx, reset); !errors.Is(err, ErrTokenInvalid) {
		t.Fatal("reset token used for verification")
	}
	results = make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- s.ResetPassword(ctx, reset, "replacement concurrent passphrase") }()
	}
	wg.Wait()
	close(results)
	success, invalid = 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrTokenInvalid) {
			invalid++
		} else {
			t.Fatal("reset race failed", err)
		}
	}
	if success != 1 || invalid != 1 {
		t.Fatal("reset token was not atomically single-use")
	}
	// Changing configured hash parameters upgrades a correct credential, not the credential itself.
	s.passwords, _ = NewPasswords(Parameters{20 * 1024, 2, 1})
	if _, err := s.Login(ctx, "tokens@example.com", "replacement concurrent passphrase"); err != nil {
		t.Fatal(err)
	}
	stored, err := s.repo.FindByID(ctx, user.ID)
	if err != nil || !strings.Contains(stored.PasswordHash, "m=20480") {
		t.Fatal("rehash was not persisted")
	}
	s.RequestChallenge(ctx, "tokens@example.com", ResetPassword)
	blocked := delivery.last(t, "tokens@example.com", ResetPassword).Token
	if err := s.Disable(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.ResetPassword(ctx, blocked, password); !errors.Is(err, ErrTokenInvalid) {
		t.Fatal("disabled account reset accepted")
	}
	// Verification can be explicitly disabled; an unverified email is still represented truthfully.
	noVerify, err := NewService(NewRepository(pool), Options{Environment: "development", Parameters: smallParameters(), VerificationTTL: time.Hour, ResetTTL: time.Minute}, delivery)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := noVerify.Signup(ctx, "no-verify@example.com", password)
	if err != nil || plain.EmailVerified || plain.Status != "active" {
		t.Fatal("verification policy not respected")
	}
	if _, err := noVerify.Login(ctx, plain.Email, password); err != nil {
		t.Fatal("explicit no-verification login rejected")
	}
}
