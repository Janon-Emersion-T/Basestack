// Package auth implements identities, credentials, challenges and opaque sessions.
package auth

import (
	"errors"
	"time"
)

var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSignupUnavailable  = errors.New("signup unavailable")
	ErrTokenInvalid       = errors.New("token invalid or expired")
	ErrUnavailable        = errors.New("auth unavailable")
	ErrBusy               = errors.New("auth busy")
)

// Internal fields cannot accidentally be serialized even if a caller ignores Public().
type User struct {
	ID                string     `json:"-"`
	Email             string     `json:"-"`
	EmailNormalized   string     `json:"-"`
	PasswordHash      string     `json:"-"`
	EmailVerifiedAt   *time.Time `json:"-"`
	Status            string     `json:"-"`
	CreatedAt         time.Time  `json:"-"`
	UpdatedAt         time.Time  `json:"-"`
	LastLoginAt       *time.Time `json:"-"`
	PasswordChangedAt time.Time  `json:"-"`
}
type PublicUser struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"emailVerified"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (u User) Public() PublicUser {
	return PublicUser{u.ID, u.Email, u.EmailVerifiedAt != nil, u.Status, u.CreatedAt}
}
func (u User) eligible() bool { return u.Status == "active" || u.Status == "unverified" }
