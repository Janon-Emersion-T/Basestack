package auth

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Janon-Emersion-T/Basestack/internal/strictjson"
	"io"
	"math"
	"mime"
	"net"
	"net/http"
	"strconv"
	"time"
)

func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
func problem(w http.ResponseWriter, status int, code, message string) {
	respond(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func errorResponse(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidRequest):
		problem(w, 400, "invalid_request", "Check the request fields and password length (15 characters to 1024 bytes).")
	case errors.Is(err, ErrInvalidCredentials):
		problem(w, 401, "invalid_credentials", "Credentials could not be verified.")
	case errors.Is(err, ErrSignupUnavailable):
		problem(w, 409, "signup_unavailable", "Signup could not be completed with these details.")
	case errors.Is(err, ErrTokenInvalid):
		problem(w, 400, "token_invalid", "The challenge is invalid, expired or already used.")
	case errors.Is(err, ErrBusy):
		w.Header().Set("Retry-After", "1")
		problem(w, 503, "auth_busy", "Try again shortly.")
	default:
		problem(w, 503, "auth_unavailable", "Authentication is temporarily unavailable.")
	}
}
func body(w http.ResponseWriter, r *http.Request, dst any) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		problem(w, 415, "unsupported_media_type", "Use application/json.")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		var size *http.MaxBytesError
		if errors.As(err, &size) {
			problem(w, 413, "request_too_large", "Request body exceeds 8192 bytes.")
		} else {
			errorResponse(w, ErrInvalidRequest)
		}
		return false
	}
	if strictjson.Decode(data, dst) != nil {
		errorResponse(w, ErrInvalidRequest)
		return false
	}
	return true
}
func Handler(service *Service, limiter Limiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := 5
		switch r.URL.Path {
		case "/api/auth/login", "/api/auth/verify-email":
			limit = 10
		case "/api/auth/signup", "/api/auth/password/forgot", "/api/auth/password/reset", "/api/auth/verify-email/request":
		default:
			problem(w, 404, "not_found", "Route not found.")
			return
		}
		if r.Method != "POST" {
			w.Header().Set("Allow", "POST, OPTIONS")
			problem(w, 405, "method_not_allowed", "Use POST.")
			return
		}
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = "unknown"
		}
		if ok, retry := limiter.Allow(ip+":"+r.URL.Path, limit); !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retry.Seconds()))))
			problem(w, 429, "rate_limited", "Try again later.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		switch r.URL.Path {
		case "/api/auth/signup", "/api/auth/login":
			var input struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			if !body(w, r, &input) {
				return
			}
			var user PublicUser
			if r.URL.Path == "/api/auth/signup" {
				user, err = service.Signup(ctx, input.Email, input.Password)
				if err == nil {
					respond(w, 201, map[string]any{"data": map[string]any{"user": user}})
					return
				}
			} else {
				user, err = service.Login(ctx, input.Email, input.Password)
				if err == nil {
					respond(w, 200, map[string]any{"data": map[string]any{"credentialsVerified": true, "sessionIssued": false, "user": user}})
					return
				}
			}
		case "/api/auth/verify-email":
			var input struct {
				Token string `json:"token"`
			}
			if !body(w, r, &input) {
				return
			}
			var user PublicUser
			user, err = service.VerifyEmail(ctx, input.Token)
			if err == nil {
				respond(w, 200, map[string]any{"data": map[string]any{"user": user}})
				return
			}
		case "/api/auth/password/forgot", "/api/auth/verify-email/request":
			var input struct {
				Email string `json:"email"`
			}
			if !body(w, r, &input) {
				return
			}
			// A minimum duration reduces simple local timing enumeration; it is not a constant-time network guarantee.
			started := time.Now()
			purpose := ResetPassword
			if r.URL.Path == "/api/auth/verify-email/request" {
				purpose = VerifyEmail
			}
			err = service.RequestChallenge(ctx, input.Email, purpose)
			if delay := 300*time.Millisecond - time.Since(started); delay > 0 {
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
			if !errors.Is(err, ErrInvalidRequest) {
				respond(w, 202, map[string]any{"data": map[string]string{"message": "If the account is eligible, instructions will be delivered."}})
				return
			}
		case "/api/auth/password/reset":
			var input struct {
				Token    string `json:"token"`
				Password string `json:"password"`
			}
			if !body(w, r, &input) {
				return
			}
			err = service.ResetPassword(ctx, input.Token, input.Password)
			if err == nil {
				respond(w, 200, map[string]any{"data": map[string]bool{"passwordChanged": true, "sessionIssued": false}})
				return
			}
		}
		errorResponse(w, err)
	})
}
