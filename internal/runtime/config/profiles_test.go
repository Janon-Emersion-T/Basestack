package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type injectedSecrets struct {
	values map[string]string
	err    error
}

func (s injectedSecrets) Read(string) (map[string]string, error) { return s.values, s.err }
func (s injectedSecrets) Set(string, string, string) error       { return nil }
func (s injectedSecrets) Unset(string, string) error             { return nil }
func TestInjectedSecretsBoundary(t *testing.T) {
	t.Setenv("BASESTACK_ENV", "test")
	values, err := EnvironmentWithStore(t.TempDir(), injectedSecrets{values: map[string]string{"PROVIDER_TEST_VALUE": "private"}})
	if err != nil || values["PROVIDER_TEST_VALUE"] != "private" {
		t.Fatal("injected store not used")
	}
	if _, err := EnvironmentWithStore(t.TempDir(), injectedSecrets{err: errors.New("PRIVATE_PROVIDER_ERROR")}); err == nil || strings.Contains(err.Error(), "PRIVATE_PROVIDER_ERROR") {
		t.Fatal("provider error leaked")
	}
	if _, err := EnvironmentWithStore(t.TempDir(), injectedSecrets{values: map[string]string{"BASESTACK_ENV": "development"}}); err == nil {
		t.Fatal("provider changed environment selection")
	}
}

func TestProfilesRoundTripPrecedenceAndPrivacy(t *testing.T) {
	dir := t.TempDir()
	store := FileSecrets{dir}
	for _, name := range []string{"development", "test", "production"} {
		if err := store.Set(name, "PRIVATE_TEST_VALUE", "literal\n$secret='界'"); err != nil {
			t.Fatal(err)
		}
		values, err := store.Read(name)
		if err != nil || values["PRIVATE_TEST_VALUE"] != "literal\n$secret='界'" {
			t.Fatal("profile roundtrip failed")
		}
		info, _ := os.Stat(filepath.Join(dir, ".basestack", "environments", name+".json"))
		if info.Mode().Perm() != 0600 {
			t.Fatal("unsafe profile permissions")
		}
	}
	os.WriteFile(filepath.Join(dir, ".env"), []byte("BASESTACK_ENV=development\nPRIVATE_TEST_VALUE=dotenv\n"), 0600)
	t.Setenv("BASESTACK_ENV", "test")
	values, err := Environment(dir)
	if err != nil || values["PRIVATE_TEST_VALUE"] != "literal\n$secret='界'" {
		t.Fatal("profile precedence failed", err)
	}
	t.Setenv("PRIVATE_TEST_VALUE", "")
	values, err = Environment(dir)
	if err != nil || values["PRIVATE_TEST_VALUE"] != "" {
		t.Fatal("explicit empty process override ignored")
	}
	if err := store.Unset("test", "PRIVATE_TEST_VALUE"); err != nil {
		t.Fatal(err)
	}
	values, err = store.Read("test")
	if err != nil || len(values) != 0 {
		t.Fatal("unset failed")
	}
	for _, tc := range [][3]string{{"../escape", "KEY", "secret"}, {"test", "VITE_SECRET", "secret"}, {"test", "BASESTACK_ENV", "secret"}, {"test", "bad", "secret"}, {"test", "KEY", "\x00secret"}} {
		if err := store.Set(tc[0], tc[1], tc[2]); err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatal("invalid profile accepted or leaked")
		}
	}
}
func TestProfilesUnsafeStateAndLock(t *testing.T) {
	dir := t.TempDir()
	store := FileSecrets{dir}
	if err := store.Set("test", "KEY", "PRIVATE_VALUE"); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, ".basestack", "environments")
	path := filepath.Join(root, "test.json")
	os.WriteFile(filepath.Join(root, "test.lock"), nil, 0600)
	if store.Set("test", "OTHER", "value") == nil {
		t.Fatal("lock ignored")
	}
	os.Remove(filepath.Join(root, "test.lock"))
	for _, data := range []string{`{"schemaVersion":1,"values":{"KEY":"PRIVATE_VALUE","KEY":"duplicate"}}`, `{"schemaVersion":2,"values":{"KEY":"PRIVATE_VALUE"}}`, `{"schemaVersion":1,"values":null}`, `{"schemaVersion":1,"values":{"KEY":null}}`} {
		os.WriteFile(path, []byte(data), 0600)
		if _, err := store.Read("test"); err == nil || strings.Contains(err.Error(), "PRIVATE_VALUE") {
			t.Fatal("malformed profile accepted or leaked")
		}
	}
	os.Remove(path)
	os.Symlink(filepath.Join(t.TempDir(), "outside"), path)
	if store.Set("test", "KEY", "value") == nil {
		t.Fatal("file symlink accepted")
	}
	os.Remove(path)
	os.Chmod(root, 0755)
	if _, err := store.Read("test"); err == nil {
		t.Fatal("unsafe directory permissions accepted")
	}
	other := t.TempDir()
	os.Symlink(t.TempDir(), filepath.Join(other, ".basestack"))
	if (FileSecrets{other}).Set("test", "KEY", "value") == nil {
		t.Fatal("directory symlink accepted")
	}
}
func TestApplicationSchemaVersions(t *testing.T) {
	c := ApplicationDefault()
	b, _ := json.Marshal(c)
	if _, err := Parse(b); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Services){func(c *Services) { c.SchemaVersion = 2 }, func(c *Services) { c.Storage = nil }, func(c *Services) { c.Storage.Provider = "s3" }, func(c *Services) { c.Database.Provider = "unknown" }, func(c *Services) { c.Storage.MaxObjectBytes = 0 }, func(c *Services) { *c.Auth.Enabled = false }} {
		c := ApplicationDefault()
		mutate(&c)
		b, _ := json.Marshal(c)
		if _, err := Parse(b); err == nil {
			t.Fatal("invalid services accepted")
		}
	}
	legacy, _ := json.Marshal(Default())
	if _, err := Parse(legacy); err != nil {
		t.Fatal("legacy rejected")
	}
	for _, key := range []string{"storage", "functions", "authorization"} {
		bad := strings.Replace(string(legacy), `"schemaVersion":2`, `"schemaVersion":2,"`+key+`":null`, 1)
		if _, err := Parse([]byte(bad)); err == nil {
			t.Fatal("v3 field accepted in v2")
		}
	}
	r, err := Resolve(Default(), func(key string) (string, bool) { return "test", key == "BASESTACK_ENV" })
	if err != nil || r.Environment != "test" {
		t.Fatal("test environment rejected")
	}
}
