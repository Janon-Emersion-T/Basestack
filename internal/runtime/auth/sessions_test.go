package auth

import (
	"context"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/rbac"
	"net/http/httptest"
	"testing"
)

func TestPostgreSQLSessionsAndPermissions(t *testing.T) {
	s, pool, delivery := setupAuth(t)
	ctx := context.Background()
	password := "correct horse battery staple"
	u, err := s.Signup(ctx, "session@example.com", password)
	if err != nil {
		t.Fatal(err)
	}
	sessions := PostgresSessions{Service: s}
	if _, err := sessions.SignIn(ctx, u.Email, password); err == nil {
		t.Fatal("unverified session issued")
	}
	if _, err := s.VerifyEmail(ctx, delivery.last(t, u.Email, VerifyEmail).Token); err != nil {
		t.Fatal(err)
	}
	session, err := sessions.SignIn(ctx, u.Email, password)
	if err != nil {
		t.Fatal(err)
	}
	if current, err := sessions.Current(ctx, session.Token); err != nil || current.ID != u.ID {
		t.Fatal("current user failed", err)
	}
	var stored []byte
	if err := pool.QueryRow(ctx, "SELECT digest FROM basestack_auth.sessions WHERE user_id=$1", u.ID).Scan(&stored); err != nil || len(stored) != 32 || string(stored) == session.Token {
		t.Fatal("unsafe token storage")
	}
	roles := rbac.Postgres{Pool: pool}
	if roles.Check(ctx, u.ID, "files.read") == nil {
		t.Fatal("unassigned permission allowed")
	}
	if err := roles.CreateRole(ctx, "editor"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 11; i++ {
		if _, err := sessions.SignIn(ctx, u.Email, password); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM basestack_auth.sessions WHERE user_id=$1", u.ID).Scan(&count); err != nil || count != 10 {
		t.Fatal("session count is not bounded")
	}
	if _, err := sessions.Current(ctx, session.Token); err == nil {
		t.Fatal("oldest session not evicted")
	}
	session, err = sessions.SignIn(ctx, u.Email, password)
	if err != nil {
		t.Fatal(err)
	}
	if err := roles.Grant(ctx, "editor", "files.read"); err != nil {
		t.Fatal(err)
	}
	if err := roles.Assign(ctx, u.ID, "editor"); err != nil {
		t.Fatal(err)
	}
	if err := roles.Check(ctx, u.ID, "files.read"); err != nil {
		t.Fatal("permission missing", err)
	}
	if roles.Check(ctx, u.ID, "files.write") == nil {
		t.Fatal("implicit permission allowed")
	}
	if err := roles.Revoke(ctx, "editor", "files.read"); err != nil {
		t.Fatal(err)
	}
	if roles.Check(ctx, u.ID, "files.read") == nil {
		t.Fatal("permission revocation stale")
	}
	roles.Grant(ctx, "editor", "files.read")
	roles.Unassign(ctx, u.ID, "editor")
	if roles.Check(ctx, u.ID, "files.read") == nil {
		t.Fatal("role revocation stale")
	}
	if err := sessions.Logout(ctx, session.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Current(ctx, session.Token); err == nil {
		t.Fatal("logout failed")
	}
	if err := sessions.Logout(ctx, session.Token); err != nil {
		t.Fatal("logout is not idempotent")
	}
	session, err = sessions.SignIn(ctx, u.Email, password)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE basestack_auth.sessions SET expires_at=clock_timestamp()-interval '1 second' WHERE user_id=$1", u.ID); err != nil {
		t.Fatal("expiry fixture failed")
	}
	if _, err := sessions.Current(ctx, session.Token); err == nil {
		t.Fatal("expired session accepted")
	}
	session, err = sessions.SignIn(ctx, u.Email, password)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdatePassword(ctx, u.ID, "replacement horse battery staple"); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Current(ctx, session.Token); err == nil {
		t.Fatal("password change did not invalidate sessions")
	}
	session, err = sessions.SignIn(ctx, u.Email, "replacement horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	roles.Assign(ctx, u.ID, "editor")
	if err := s.Suspend(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Current(ctx, session.Token); err == nil {
		t.Fatal("suspended session accepted")
	}
	if roles.Check(ctx, u.ID, "files.read") == nil {
		t.Fatal("suspended user authorized")
	}
}
func TestBearerSource(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/auth/me?token=secret", nil)
	r.Header.Set("Cookie", "token=secret")
	if Bearer(r) != "" {
		t.Fatal("ambient credential accepted")
	}
	r.Header.Set("Authorization", "Bearer value")
	if Bearer(r) != "value" {
		t.Fatal("bearer rejected")
	}
	r.Header.Add("Authorization", "Bearer other")
	if Bearer(r) != "" {
		t.Fatal("duplicate credentials accepted")
	}
}
