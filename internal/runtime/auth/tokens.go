package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

const VerifyEmail = "verify_email"
const ResetPassword = "reset_password"

type Challenge struct {
	Digest    []byte
	Purpose   string
	ExpiresAt time.Time
}

func UUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", ErrUnavailable
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
func NewChallenge(purpose string, ttl time.Duration) (string, Challenge, error) {
	if (purpose != VerifyEmail && purpose != ResetPassword) || ttl <= 0 {
		return "", Challenge{}, ErrInvalidRequest
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", Challenge{}, ErrUnavailable
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256(raw)
	return token, Challenge{Digest: digest[:], Purpose: purpose, ExpiresAt: time.Now().UTC().Add(ttl)}, nil
}
func TokenDigest(token string) ([]byte, error) {
	if len(token) != 43 {
		return nil, ErrTokenInvalid
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil || len(raw) != 32 {
		return nil, ErrTokenInvalid
	}
	digest := sha256.Sum256(raw)
	return digest[:], nil
}
