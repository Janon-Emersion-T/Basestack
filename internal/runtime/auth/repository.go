package auth

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

var errNotFound = errors.New("user not found")

const userColumns = `id::text,email,email_normalized,password_hash,email_verified_at,status,created_at,updated_at,last_login_at,password_changed_at`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.EmailNormalized, &u.PasswordHash, &u.EmailVerifiedAt, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt, &u.PasswordChangedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, errNotFound
	}
	if err != nil {
		return u, ErrUnavailable
	}
	return u, nil
}
func (r *Repository) FindByEmail(ctx context.Context, email string) (User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM basestack_auth.users WHERE email_normalized=$1`, email))
}
func (r *Repository) FindByID(ctx context.Context, id string) (User, error) {
	if !uuidPattern.MatchString(id) {
		return User{}, ErrInvalidRequest
	}
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM basestack_auth.users WHERE id=$1`, id))
}
func (r *Repository) transaction(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ErrUnavailable
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(c)
	}()
	if err := fn(tx); err != nil {
		return err
	}
	if tx.Commit(ctx) != nil {
		return ErrUnavailable
	}
	return nil
}
func audit(ctx context.Context, tx pgx.Tx, id, kind string) error {
	if _, err := tx.Exec(ctx, `INSERT INTO basestack_auth.events(user_id,type) VALUES(NULLIF($1,'')::uuid,$2)`, id, kind); err != nil {
		return ErrUnavailable
	}
	return nil
}
func (r *Repository) Event(ctx context.Context, id, kind string) error {
	return r.transaction(ctx, func(tx pgx.Tx) error { return audit(ctx, tx, id, kind) })
}
func insertChallenge(ctx context.Context, tx pgx.Tx, id string, c Challenge) error {
	if _, err := tx.Exec(ctx, `UPDATE basestack_auth.challenges SET consumed_at=clock_timestamp() WHERE user_id=$1 AND purpose=$2 AND consumed_at IS NULL`, id, c.Purpose); err != nil {
		return ErrUnavailable
	}
	if _, err := tx.Exec(ctx, `INSERT INTO basestack_auth.challenges(digest,user_id,purpose,expires_at) VALUES($1,$2,$3,$4)`, c.Digest, id, c.Purpose, c.ExpiresAt); err != nil {
		return ErrUnavailable
	}
	return nil
}
func (r *Repository) Create(ctx context.Context, u User, challenge *Challenge) (User, error) {
	var result User
	err := r.transaction(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `INSERT INTO basestack_auth.users(id,email,email_normalized,password_hash,status) VALUES($1,$2,$3,$4,$5) RETURNING `+userColumns, u.ID, u.Email, u.EmailNormalized, u.PasswordHash, u.Status)
		// Keep only the uniqueness category; never return driver text or stored values.
		err := row.Scan(&result.ID, &result.Email, &result.EmailNormalized, &result.PasswordHash, &result.EmailVerifiedAt, &result.Status, &result.CreatedAt, &result.UpdatedAt, &result.LastLoginAt, &result.PasswordChangedAt)
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) && pgerr.Code == "23505" {
			return ErrSignupUnavailable
		}
		if err != nil {
			return ErrUnavailable
		}
		if challenge != nil {
			if err := insertChallenge(ctx, tx, u.ID, *challenge); err != nil {
				return err
			}
		}
		return audit(ctx, tx, u.ID, "user_created")
	})
	return result, err
}
func (r *Repository) Issue(ctx context.Context, email string, c Challenge) (*User, error) {
	var recipient *User
	err := r.transaction(ctx, func(tx pgx.Tx) error {
		u, err := scanUser(tx.QueryRow(ctx, `SELECT `+userColumns+` FROM basestack_auth.users WHERE email_normalized=$1 FOR UPDATE`, email))
		if err != nil && !errors.Is(err, errNotFound) {
			return err
		}
		eligible := err == nil && u.eligible() && (c.Purpose != VerifyEmail || u.EmailVerifiedAt == nil)
		id := ""
		if eligible {
			if err := insertChallenge(ctx, tx, u.ID, c); err != nil {
				return err
			}
			recipient = &u
			id = u.ID
		}
		kind := "password_reset_requested"
		if c.Purpose == VerifyEmail {
			kind = "verification_requested"
		}
		return audit(ctx, tx, id, kind)
	})
	return recipient, err
}
func (r *Repository) ChallengeUser(ctx context.Context, digest []byte, purpose string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `SELECT user_id::text FROM basestack_auth.challenges WHERE digest=$1 AND purpose=$2 AND consumed_at IS NULL AND expires_at>clock_timestamp()`, digest, purpose).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrTokenInvalid
	}
	if err != nil {
		return "", ErrUnavailable
	}
	return id, nil
}
func (r *Repository) Consume(ctx context.Context, digest []byte, purpose, newHash string) (User, error) {
	id, err := r.ChallengeUser(ctx, digest, purpose)
	if err != nil {
		return User{}, err
	}
	var result User
	err = r.transaction(ctx, func(tx pgx.Tx) error {
		u, err := scanUser(tx.QueryRow(ctx, `SELECT `+userColumns+` FROM basestack_auth.users WHERE id=$1 FOR UPDATE`, id))
		if err != nil {
			return err
		}
		if !u.eligible() {
			return ErrTokenInvalid
		}
		var used string
		err = tx.QueryRow(ctx, `UPDATE basestack_auth.challenges SET consumed_at=clock_timestamp() WHERE digest=$1 AND user_id=$2 AND purpose=$3 AND consumed_at IS NULL AND expires_at>clock_timestamp() RETURNING user_id::text`, digest, id, purpose).Scan(&used)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTokenInvalid
		}
		if err != nil {
			return ErrUnavailable
		}
		if purpose == VerifyEmail {
			result, err = scanUser(tx.QueryRow(ctx, `UPDATE basestack_auth.users SET email_verified_at=clock_timestamp(),status='active',updated_at=clock_timestamp() WHERE id=$1 RETURNING `+userColumns, id))
		} else {
			result, err = scanUser(tx.QueryRow(ctx, `UPDATE basestack_auth.users SET password_hash=$2,password_changed_at=clock_timestamp(),updated_at=clock_timestamp() WHERE id=$1 RETURNING `+userColumns, id, newHash))
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE basestack_auth.challenges SET consumed_at=clock_timestamp() WHERE user_id=$1 AND purpose=$2 AND consumed_at IS NULL`, id, purpose); err != nil {
			return ErrUnavailable
		}
		kind := "email_verified"
		if purpose == ResetPassword {
			kind = "password_reset_completed"
		}
		return audit(ctx, tx, id, kind)
	})
	return result, err
}
func (r *Repository) Login(ctx context.Context, id, previousHash, newHash string, requireVerification bool) (User, error) {
	var result User
	err := r.transaction(ctx, func(tx pgx.Tx) error {
		u, err := scanUser(tx.QueryRow(ctx, `SELECT `+userColumns+` FROM basestack_auth.users WHERE id=$1 FOR UPDATE`, id))
		if err != nil {
			return err
		}
		if u.PasswordHash != previousHash || u.Status != "active" || (requireVerification && u.EmailVerifiedAt == nil) {
			return ErrInvalidCredentials
		}
		if newHash == "" {
			newHash = u.PasswordHash
		}
		result, err = scanUser(tx.QueryRow(ctx, `UPDATE basestack_auth.users SET password_hash=$2,last_login_at=clock_timestamp(),updated_at=clock_timestamp() WHERE id=$1 RETURNING `+userColumns, id, newHash))
		if err != nil {
			return err
		}
		return audit(ctx, tx, id, "login_succeeded")
	})
	return result, err
}

// Lifecycle operations are internal only. A verified admin authorization boundary must precede future HTTP exposure.
func (r *Repository) SetStatus(ctx context.Context, id, status string) error {
	if !uuidPattern.MatchString(id) || (status != "suspended" && status != "disabled") {
		return ErrInvalidRequest
	}
	return r.transaction(ctx, func(tx pgx.Tx) error {
		u, err := scanUser(tx.QueryRow(ctx, `SELECT `+userColumns+` FROM basestack_auth.users WHERE id=$1 FOR UPDATE`, id))
		if err != nil {
			return err
		}
		if u.Status == "disabled" && status != "disabled" {
			return ErrInvalidRequest
		}
		if u.Status == status {
			return nil
		}
		if _, err := tx.Exec(ctx, `UPDATE basestack_auth.users SET status=$2,updated_at=clock_timestamp() WHERE id=$1`, id, status); err != nil {
			return ErrUnavailable
		}
		if _, err := tx.Exec(ctx, `UPDATE basestack_auth.challenges SET consumed_at=clock_timestamp() WHERE user_id=$1 AND consumed_at IS NULL`, id); err != nil {
			return ErrUnavailable
		}
		return audit(ctx, tx, id, "user_"+status)
	})
}
func (r *Repository) UpdatePassword(ctx context.Context, id, hash string) error {
	if !uuidPattern.MatchString(id) {
		return ErrInvalidRequest
	}
	return r.transaction(ctx, func(tx pgx.Tx) error {
		if _, err := scanUser(tx.QueryRow(ctx, `SELECT `+userColumns+` FROM basestack_auth.users WHERE id=$1 FOR UPDATE`, id)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE basestack_auth.users SET password_hash=$2,password_changed_at=clock_timestamp(),updated_at=clock_timestamp() WHERE id=$1`, id, hash); err != nil {
			return ErrUnavailable
		}
		if _, err := tx.Exec(ctx, `UPDATE basestack_auth.challenges SET consumed_at=clock_timestamp() WHERE user_id=$1 AND purpose='reset_password' AND consumed_at IS NULL`, id); err != nil {
			return ErrUnavailable
		}
		return audit(ctx, tx, id, "password_changed")
	})
}

func (r *Repository) Ready(ctx context.Context) error {
	var ready bool
	if err := r.pool.QueryRow(ctx, `SELECT to_regclass('basestack_auth.users') IS NOT NULL AND to_regclass('basestack_auth.challenges') IS NOT NULL AND to_regclass('basestack_auth.events') IS NOT NULL`).Scan(&ready); err != nil || !ready {
		return ErrUnavailable
	}
	return nil
}
