package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Janon-Emersion-T/Basestack/internal/project"
	"github.com/Janon-Emersion-T/Basestack/internal/registry"
)

func TestCreateAndProtectPaths(t *testing.T) {
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Fatal(err)
		}
	})
	if err := Create("app"); err != nil {
		t.Fatal(err)
	}
	catalog, err := os.ReadFile("app/src/registry.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(catalog, registry.BuiltinJSON) {
		t.Fatal("generated catalog differs from CLI")
	}
	data, err := os.ReadFile(filepath.Join("app", project.ConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := project.Parse(data); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("existing-file", []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("app", "existing-link"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"app", "existing-file", "existing-link", "../outside", "UPPER", ""} {
		if err := Create(name); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
	data, err = os.ReadFile("existing-file")
	if err != nil || string(data) != "keep" {
		t.Fatal("existing file damaged")
	}
	if info, err := os.Lstat("existing-link"); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink damaged")
	}
}

func TestGeneratedRuntimeAndNoSecrets(t *testing.T) {
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
	if err := Create("services-app"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"go.mod", "go.sum", "cmd/server/main.go", "internal/runtime/app/app.go", "internal/runtime/config/config.go", "internal/runtime/database/database.go", "internal/runtime/server/server_test.go", "internal/runtime/migrations/migrations.go", "internal/strictjson/json.go", "compose.yaml", "basestack/services.json", "basestack/migrations/000001_initial.sql", ".env.example"} {
		if _, err := os.Stat(filepath.Join("services-app", path)); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{".env", ".basestack/local.env"} {
		if _, err := os.Stat(filepath.Join("services-app", path)); !os.IsNotExist(err) {
			t.Fatalf("generated secret file %s", path)
		}
	}
	ignore, _ := os.ReadFile("services-app/.gitignore")
	if !bytes.Contains(ignore, []byte(".env")) || !bytes.Contains(ignore, []byte(".basestack/")) {
		t.Fatal("secret files not ignored")
	}
	module, _ := os.ReadFile("services-app/go.mod")
	if bytes.Contains(module, []byte("github.com/Janon-Emersion-T/Basestack")) {
		t.Fatal("generated project depends on BaseStack module")
	}
}
