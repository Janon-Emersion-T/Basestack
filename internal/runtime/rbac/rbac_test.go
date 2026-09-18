package rbac

import (
	"context"
	"testing"
)

func TestInvalidPermissionInputsFailClosed(t *testing.T) {
	p := Postgres{}
	ctx := context.Background()
	for _, value := range []string{"", "*", "Admin", "../permission", "permission;DROP TABLE", "permission with space"} {
		if ValidName(value) {
			t.Fatal("unsafe role/permission accepted")
		}
		if p.CreateRole(ctx, value) == nil || p.Grant(ctx, "member", value) == nil {
			t.Fatal("invalid operation accepted")
		}
	}
	if p.Check(ctx, "invalid-user", "files.read") != ErrDenied {
		t.Fatal("invalid identity not denied")
	}
	if ValidCommand([]string{"assign", "not-a-uuid", "admin"}) {
		t.Fatal("invalid CLI input accepted")
	}
}
