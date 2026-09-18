// Package rbac checks explicit permissions for authenticated identities.
package rbac

import (
	"context"
	_ "embed"
	"errors"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var SchemaSQL string

var ErrDenied = errors.New("permission denied")
var ErrUnavailable = errors.New("authorization unavailable")
var name = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{0,63}$`)
var userID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func ValidName(value string) bool { return name.MatchString(value) }

type Authorizer interface {
	Check(context.Context, string, string) error
}
type Manager interface {
	Authorizer
	CreateRole(context.Context, string) error
	Grant(context.Context, string, string) error
	Revoke(context.Context, string, string) error
	Assign(context.Context, string, string) error
	Unassign(context.Context, string, string) error
}

// Postgres reads current assignments on each check; revocation requires no cache expiry.
// Management methods are trusted server/CLI operations, never public self-service routes.
type Postgres struct{ Pool *pgxpool.Pool }

func (p Postgres) Ready(ctx context.Context) error {
	var ready bool
	if err := p.Pool.QueryRow(ctx, `SELECT to_regclass('basestack_rbac.roles') IS NOT NULL AND to_regclass('basestack_rbac.permissions') IS NOT NULL AND to_regclass('basestack_rbac.user_roles') IS NOT NULL`).Scan(&ready); err != nil || !ready {
		return ErrUnavailable
	}
	return nil
}

func (p Postgres) Check(ctx context.Context, id, permission string) error {
	if !userID.MatchString(id) || !ValidName(permission) {
		return ErrDenied
	}
	var allowed bool
	err := p.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM basestack_rbac.user_roles ur JOIN basestack_rbac.permissions rp ON rp.role=ur.role JOIN basestack_auth.users u ON u.id=ur.user_id WHERE ur.user_id=$1 AND rp.permission=$2 AND u.status='active')`, id, permission).Scan(&allowed)
	if err != nil {
		return ErrUnavailable
	}
	if !allowed {
		return ErrDenied
	}
	return nil
}
func (p Postgres) execute(ctx context.Context, query string, args ...any) error {
	if _, err := p.Pool.Exec(ctx, query, args...); err != nil {
		return ErrUnavailable
	}
	return nil
}
func (p Postgres) CreateRole(ctx context.Context, role string) error {
	if !ValidName(role) {
		return ErrDenied
	}
	return p.execute(ctx, `INSERT INTO basestack_rbac.roles(name) VALUES($1) ON CONFLICT DO NOTHING`, role)
}
func (p Postgres) Grant(ctx context.Context, role, permission string) error {
	if !ValidName(role) || !ValidName(permission) {
		return ErrDenied
	}
	return p.execute(ctx, `INSERT INTO basestack_rbac.permissions(role,permission) VALUES($1,$2) ON CONFLICT DO NOTHING`, role, permission)
}
func (p Postgres) Revoke(ctx context.Context, role, permission string) error {
	if !ValidName(role) || !ValidName(permission) {
		return ErrDenied
	}
	return p.execute(ctx, `DELETE FROM basestack_rbac.permissions WHERE role=$1 AND permission=$2`, role, permission)
}
func (p Postgres) Assign(ctx context.Context, id, role string) error {
	if !ValidName(role) || !userID.MatchString(id) {
		return ErrDenied
	}
	return p.execute(ctx, `INSERT INTO basestack_rbac.user_roles(user_id,role) VALUES($1,$2) ON CONFLICT DO NOTHING`, id, role)
}
func (p Postgres) Unassign(ctx context.Context, id, role string) error {
	if !ValidName(role) || !userID.MatchString(id) {
		return ErrDenied
	}
	return p.execute(ctx, `DELETE FROM basestack_rbac.user_roles WHERE user_id=$1 AND role=$2`, id, role)
}
