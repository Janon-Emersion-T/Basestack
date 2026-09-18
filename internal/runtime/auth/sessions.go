package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Session carries the bearer credential only in an explicit login response.
// Never log a Session or persist its raw Token in browser storage.
type Session struct {
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expiresAt"`
	User      PublicUser `json:"user"`
}

// Sessions is the identity-provider boundary consumed by protected HTTP routes.
type Sessions interface {
	SignIn(context.Context, string, string) (Session, error)
	Current(context.Context, string) (PublicUser, error)
	Logout(context.Context, string) error
}

type PostgresSessions struct{ Service *Service }

func (s PostgresSessions) SignIn(ctx context.Context, email, password string) (Session, error) {
	u, err := s.Service.verifyCredentials(ctx, email, password)
	if err != nil {
		return Session{}, err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Session{}, ErrUnavailable
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	digest, _ := TokenDigest(token)
	var expires time.Time
	err = s.Service.repo.transaction(ctx, func(tx pgx.Tx) error {
		current, err := scanUser(tx.QueryRow(ctx, `SELECT `+userColumns+` FROM basestack_auth.users WHERE id=$1 FOR UPDATE`, u.ID))
		if err != nil {
			return err
		}
		if current.Status != "active" || current.PasswordHash != u.PasswordHash || !current.PasswordChangedAt.Equal(u.PasswordChangedAt) {
			return ErrInvalidCredentials
		}
		// Bound active sessions per account; remove expired/oldest sessions before insertion.
		if _, err := tx.Exec(ctx, `DELETE FROM basestack_auth.sessions WHERE user_id=$1 AND (expires_at<=clock_timestamp() OR digest IN (SELECT digest FROM basestack_auth.sessions WHERE user_id=$1 ORDER BY created_at DESC OFFSET 9))`, u.ID); err != nil {
			return ErrUnavailable
		}
		if err := tx.QueryRow(ctx, `INSERT INTO basestack_auth.sessions(digest,user_id,password_changed_at,expires_at) VALUES($1,$2,$3,clock_timestamp()+interval '12 hours') RETURNING expires_at`, digest, u.ID, u.PasswordChangedAt).Scan(&expires); err != nil {
			return ErrUnavailable
		}
		return nil
	})
	if err != nil {
		return Session{}, err
	}
	return Session{token, expires, u.Public()}, nil
}
func (s PostgresSessions) Current(ctx context.Context, token string) (PublicUser, error) {
	digest, err := TokenDigest(token)
	if err != nil {
		return PublicUser{}, ErrInvalidCredentials
	}
	var id string
	err = s.Service.repo.pool.QueryRow(ctx, `SELECT s.user_id::text FROM basestack_auth.sessions s JOIN basestack_auth.users u ON u.id=s.user_id WHERE s.digest=$1 AND s.expires_at>clock_timestamp() AND u.status='active' AND s.password_changed_at=u.password_changed_at AND (NOT $2 OR u.email_verified_at IS NOT NULL)`, digest, s.Service.options.RequireVerification).Scan(&id)
	if err == pgx.ErrNoRows {
		return PublicUser{}, ErrInvalidCredentials
	}
	if err != nil {
		return PublicUser{}, ErrUnavailable
	}
	return s.Service.FindByID(ctx, id)
}
func (s PostgresSessions) Logout(ctx context.Context, token string) error {
	digest, err := TokenDigest(token)
	if err != nil {
		return ErrInvalidCredentials
	}
	if _, err := s.Service.repo.pool.Exec(ctx, `DELETE FROM basestack_auth.sessions WHERE digest=$1`, digest); err != nil {
		return ErrUnavailable
	}
	return nil
}

// Bearer never accepts identity from cookies, URL parameters or arbitrary user headers.
func Bearer(r *http.Request) string {
	values := r.Header.Values("Authorization")
	if len(values) != 1 {
		return ""
	}
	parts := strings.Fields(values[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func SessionHandler(core http.Handler, sessions Sessions, limiter Limiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/login" && r.URL.Path != "/api/auth/logout" && r.URL.Path != "/api/auth/me" {
			core.ServeHTTP(w, r)
			return
		}
		method := "POST"
		if r.URL.Path == "/api/auth/me" {
			method = "GET"
		}
		if r.Method != method {
			w.Header().Set("Allow", method+", OPTIONS")
			problem(w, 405, "method_not_allowed", "Method not allowed.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		if r.URL.Path == "/api/auth/login" {
			// Share the existing bounded IP limiter with credential/challenge routes.
			if ok, _ := limiter.Allow(clientIP(r)+":session-login", 10); !ok {
				w.Header().Set("Retry-After", "60")
				problem(w, 429, "rate_limited", "Try again later.")
				return
			}
			var input struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			if !body(w, r, &input) {
				return
			}
			session, err := sessions.SignIn(ctx, input.Email, input.Password)
			if err != nil {
				errorResponse(w, err)
				return
			}
			respond(w, 200, map[string]any{"data": map[string]any{"credentialsVerified": true, "sessionIssued": true, "session": session, "user": session.User}})
			return
		}
		token := Bearer(r)
		if r.URL.Path == "/api/auth/logout" {
			if err := sessions.Logout(ctx, token); err != nil {
				errorResponse(w, err)
				return
			}
			respond(w, 200, map[string]any{"data": map[string]bool{"loggedOut": true}})
			return
		}
		u, err := sessions.Current(ctx, token)
		if err != nil {
			errorResponse(w, err)
			return
		}
		respond(w, 200, map[string]any{"data": map[string]any{"user": u}})
	})
}
