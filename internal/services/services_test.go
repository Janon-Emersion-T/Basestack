package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setup(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "basestack"), 0755)
	b, _ := json.Marshal(config.Default())
	os.WriteFile(filepath.Join(dir, config.File), b, 0644)
	os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("services: {}"), 0644)
	return dir
}
func TestCommandsAndCredentials(t *testing.T) {
	dir := setup(t)
	calls := 0
	runner := func(ctx context.Context, wd string, args, env []string) (string, error) {
		calls++
		if wd != dir || args[0] != "compose" {
			t.Fatal("wrong command")
		}
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "down") || strings.Contains(joined, "--volumes") {
			t.Fatal("destructive command")
		}
		if ctx == nil {
			t.Fatal("missing context")
		}
		for _, e := range env {
			if strings.HasPrefix(e, "BASESTACK_DB_PASSWORD=") {
				return e, nil
			}
		}
		t.Fatal("password missing from environment")
		return "", nil
	}
	var out bytes.Buffer
	for _, action := range []string{"start", "status", "stop"} {
		if err := Execute(context.Background(), dir, action, &out, runner); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 3 || !strings.Contains(out.String(), "[redacted]") {
		t.Fatal("command/redaction failed")
	}
	path := filepath.Join(dir, config.LocalEnv)
	before, _ := os.ReadFile(path)
	if err := Credentials(dir); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("credentials overwritten")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("unsafe credential permissions")
	}
	dir2 := setup(t)
	Credentials(dir2)
	other, _ := os.ReadFile(filepath.Join(dir2, config.LocalEnv))
	if bytes.Equal(before, other) {
		t.Fatal("passwords reused")
	}
}
func TestFailureAndValidation(t *testing.T) {
	dir := setup(t)
	called := false
	runner := func(context.Context, string, []string, []string) (string, error) {
		called = true
		return "PRIVATE_SECRET", fmt.Errorf("PRIVATE_SECRET")
	}
	if err := Execute(context.Background(), dir, "delete", &bytes.Buffer{}, runner); err == nil || called {
		t.Fatal("invalid service command executed")
	}
	err := Execute(context.Background(), dir, "start", &bytes.Buffer{}, runner)
	if err == nil || strings.Contains(err.Error(), "PRIVATE_SECRET") {
		t.Fatal("failed command accepted or leaked")
	}
	c := config.Default()
	*c.Database.Enabled = false
	*c.Auth.Enabled = false
	b, _ := json.Marshal(c)
	os.WriteFile(filepath.Join(dir, config.File), b, 0644)
	called = false
	if err := Execute(context.Background(), dir, "start", &bytes.Buffer{}, runner); err == nil || called {
		t.Fatal("disabled service started")
	}
	unsafe := setup(t)
	os.Symlink(t.TempDir(), filepath.Join(unsafe, ".basestack"))
	if Credentials(unsafe) == nil {
		t.Fatal("symlink followed")
	}
}
