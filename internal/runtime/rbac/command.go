package rbac

import (
	"context"
	"errors"
)

const Usage = "usage: basestack roles create <role> | grant <role> <permission> | revoke <role> <permission> | assign <user-id> <role> | unassign <user-id> <role> | check <user-id> <permission>"

func ValidCommand(args []string) bool {
	if len(args) == 2 && args[0] == "create" {
		return ValidName(args[1])
	}
	if len(args) != 3 {
		return false
	}
	switch args[0] {
	case "grant", "revoke":
		return ValidName(args[1]) && ValidName(args[2])
	case "assign", "unassign", "check":
		return userID.MatchString(args[1]) && ValidName(args[2])
	}
	return false
}
func Command(ctx context.Context, m Manager, args []string) error {
	if !ValidCommand(args) {
		return errors.New(Usage)
	}
	switch args[0] {
	case "create":
		return m.CreateRole(ctx, args[1])
	case "grant":
		return m.Grant(ctx, args[1], args[2])
	case "revoke":
		return m.Revoke(ctx, args[1], args[2])
	case "assign":
		return m.Assign(ctx, args[1], args[2])
	case "unassign":
		return m.Unassign(ctx, args[1], args[2])
	default:
		return m.Check(ctx, args[1], args[2])
	}
}
