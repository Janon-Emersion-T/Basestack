package cli

import (
	"bytes"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrationCLIAndLegacyCompatibility(t *testing.T) {
	inTemp(t)
	run(t, "init", "app")
	if err := os.Chdir("app"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile("basestack.json")
	output := run(t, "migration", "new", "create_example")
	if !strings.Contains(output, "000003_create_example.sql") {
		t.Fatal(output)
	}
	if _, err := os.Stat(filepath.Join("basestack", "migrations", "000003_create_example.sql")); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"migration", "new", "create_example"}, {"migration", "new", "../unsafe"}, {"migration"}, {"migration", "new"}, {"services", "delete"}, {"services", "start", "extra"}, {"db", "reset"}, {"db"}, {"api", "extra"}} {
		if err := Run(args, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	after, _ := os.ReadFile("basestack.json")
	if !bytes.Equal(before, after) {
		t.Fatal("services modified composition schema")
	}
	// Existing v2 projects have no service sidecar. Their original commands still work.
	if err := os.Remove(config.File); err != nil {
		t.Fatal(err)
	}
	run(t, "check")
	run(t, "page", "add", "legacy")
	run(t, "add", "hero", "--page", "legacy")
	run(t, "theme", "set", "modern")
	if err := Run([]string{"services", "status"}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "0.3") {
		t.Fatal("legacy service error missing")
	}
	if err := os.WriteFile(config.File, []byte(`{"schemaVersion":2}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"check"}, &bytes.Buffer{}); err == nil {
		t.Fatal("broken service config ignored")
	}
}

func TestAuthStatus(t *testing.T) {
	inTemp(t)
	run(t, "init", "auth-app")
	if err := os.Chdir("auth-app"); err != nil {
		t.Fatal(err)
	}
	output := run(t, "auth", "status")
	if !strings.Contains(output, "Auth enabled: true") || !strings.Contains(output, "no sessions") {
		t.Fatal(output)
	}
	t.Setenv("BASESTACK_DATABASE_URL", "postgres://PRIVATE_PASSWORD@localhost/db")
	if strings.Contains(run(t, "auth", "status"), "PRIVATE_PASSWORD") {
		t.Fatal("auth status leaked DSN")
	}
	if err := Run([]string{"auth", "enable"}, &bytes.Buffer{}); err == nil {
		t.Fatal("unimplemented Auth command accepted")
	}
}
