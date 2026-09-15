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
