package auth

import (
	"context"
	"errors"
	"time"
)

type Options struct {
	RequireVerification bool
	Environment         string
	VerificationTTL     time.Duration
	ResetTTL            time.Duration
	Parameters          Parameters
}
type Service struct {
	repo      *Repository
	passwords *Passwords
	delivery  Delivery
	options   Options
	dummy     string
}

func NewService(repo *Repository, options Options, delivery Delivery) (*Service, error) {
	if repo == nil || delivery == nil || (options.Environment != "development" && options.Environment != "production") || (options.Environment != "development" && delivery.DevelopmentOnly()) || options.VerificationTTL <= 0 || options.VerificationTTL > 24*time.Hour || options.ResetTTL <= 0 || options.ResetTTL > time.Hour {
		return nil, ErrUnavailable
	}
	passwords, err := NewPasswords(options.Parameters)
	if err != nil {
		return nil, err
	}
	dummy, err := passwords.Hash(context.Background(), "BaseStack dummy credential comparison")
	if err != nil {
		return nil, err
	}
	return &Service{repo: repo, passwords: passwords, delivery: delivery, options: options, dummy: dummy}, nil
}
func (s *Service) deliver(ctx context.Context, u User, token string, c Challenge) {
	// Persist first; delivery is a separate side effect. Resend is explicit and generic on failure.
	if s.delivery.Deliver(ctx, Message{u.Email, c.Purpose, token, c.ExpiresAt}) != nil {
		_ = s.repo.Event(ctx, u.ID, "delivery_failed")
	}
}
func (s *Service) Signup(ctx context.Context, email, password string) (PublicUser, error) {
	display, normalized, err := NormalizeEmail(email)
	if err != nil {
		return PublicUser{}, err
	}
	hash, err := s.passwords.Hash(ctx, password)
	if err != nil {
		return PublicUser{}, err
	}
	id, err := UUID()
	if err != nil {
		return PublicUser{}, err
	}
	u := User{ID: id, Email: display, EmailNormalized: normalized, PasswordHash: hash, Status: "active"}
	var challenge *Challenge
	var token string
	if s.options.RequireVerification {
		u.Status = "unverified"
		var c Challenge
		token, c, err = NewChallenge(VerifyEmail, s.options.VerificationTTL)
		if err != nil {
			return PublicUser{}, err
		}
		challenge = &c
	}
	u, err = s.repo.Create(ctx, u, challenge)
	if err != nil {
		return PublicUser{}, err
	}
	if challenge != nil {
		s.deliver(ctx, u, token, *challenge)
	}
	return u.Public(), nil
}
func (s *Service) Login(ctx context.Context, email, password string) (PublicUser, error) {
	_, normalized, err := NormalizeEmail(email)
	if err != nil || len(password) > 1024 {
		return PublicUser{}, ErrInvalidCredentials
	}
	u, findErr := s.repo.FindByEmail(ctx, normalized)
	if findErr != nil && !errors.Is(findErr, errNotFound) {
		return PublicUser{}, findErr
	}
	hash := s.dummy
	if findErr == nil {
		hash = u.PasswordHash
	}
	ok, rehash, err := s.passwords.Verify(ctx, password, hash)
	if errors.Is(err, ErrBusy) || errors.Is(err, ErrUnavailable) {
		return PublicUser{}, err
	}
	if err != nil || !ok || findErr != nil || u.Status != "active" || (s.options.RequireVerification && u.EmailVerifiedAt == nil) {
		id := ""
		if findErr == nil {
			id = u.ID
		}
		if err := s.repo.Event(ctx, id, "login_failed"); err != nil {
			return PublicUser{}, err
		}
		return PublicUser{}, ErrInvalidCredentials
	}
	newHash := ""
	if rehash {
		newHash, err = s.passwords.Hash(ctx, password)
		if err != nil {
			return PublicUser{}, err
		}
	}
	id := u.ID
	u, err = s.repo.Login(ctx, u.ID, u.PasswordHash, newHash, s.options.RequireVerification)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			_ = s.repo.Event(ctx, id, "login_failed")
		}
		return PublicUser{}, err
	}
	return u.Public(), nil
}
func (s *Service) RequestChallenge(ctx context.Context, email, purpose string) error {
	_, normalized, err := NormalizeEmail(email)
	if err != nil {
		return ErrInvalidRequest
	}
	ttl := s.options.ResetTTL
	if purpose == VerifyEmail {
		ttl = s.options.VerificationTTL
	}
	token, c, err := NewChallenge(purpose, ttl)
	if err != nil {
		return err
	}
	u, err := s.repo.Issue(ctx, normalized, c)
	if err != nil {
		return err
	}
	if u != nil {
		s.deliver(ctx, *u, token, c)
	}
	return nil
}
func (s *Service) VerifyEmail(ctx context.Context, token string) (PublicUser, error) {
	digest, err := TokenDigest(token)
	if err != nil {
		return PublicUser{}, err
	}
	u, err := s.repo.Consume(ctx, digest, VerifyEmail, "")
	return u.Public(), err
}
func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if err := ValidatePassword(password); err != nil {
		return err
	}
	digest, err := TokenDigest(token)
	if err != nil {
		return err
	}
	if _, err := s.repo.ChallengeUser(ctx, digest, ResetPassword); err != nil {
		return err
	}
	hash, err := s.passwords.Hash(ctx, password)
	if err != nil {
		return err
	}
	_, err = s.repo.Consume(ctx, digest, ResetPassword, hash)
	return err
}
func (s *Service) FindByID(ctx context.Context, id string) (PublicUser, error) {
	u, err := s.repo.FindByID(ctx, id)
	return u.Public(), err
}
func (s *Service) FindByEmail(ctx context.Context, email string) (PublicUser, error) {
	_, normalized, err := NormalizeEmail(email)
	if err != nil {
		return PublicUser{}, err
	}
	u, err := s.repo.FindByEmail(ctx, normalized)
	return u.Public(), err
}
func (s *Service) Suspend(ctx context.Context, id string) error {
	return s.repo.SetStatus(ctx, id, "suspended")
}
func (s *Service) Disable(ctx context.Context, id string) error {
	return s.repo.SetStatus(ctx, id, "disabled")
}
func (s *Service) UpdatePassword(ctx context.Context, id, password string) error {
	hash, err := s.passwords.Hash(ctx, password)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, id, hash)
}
