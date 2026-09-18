package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStrictServices(t *testing.T) {
	valid, _ := json.Marshal(Default())
	if _, err := Parse(valid); err != nil {
		t.Fatal(err)
	}
	cases := []string{"{}", "null", "[]", string(valid) + " {}", strings.Replace(string(valid), `"schemaVersion":2`, `"schemaVersion":3`, 1), strings.Replace(string(valid), `"schemaVersion":2`, `"schemaVersion":2,"schemaVersion":2`, 1), strings.Replace(string(valid), `"enabled":true`, `"enabled":null`, 1), strings.Replace(string(valid), `"enabled":true`, `"Enabled":true`, 1), strings.Replace(string(valid), `"enabled":true`, `"enabled":true,"password":"DO_NOT_PRINT"`, 1), strings.Replace(string(valid), `"port":54322`, `"port":0`, 1), strings.Replace(string(valid), `"host":"127.0.0.1"`, `"host":"bad/host"`, 1)}
	for _, data := range cases {
		if _, err := Parse([]byte(data)); err == nil {
			t.Fatalf("accepted %s", data)
		}
	}
	for _, origins := range [][]string{nil, {"*"}, {"http://localhost:5173/"}, {"http://user:password@localhost:5173"}, {"null"}, {"https://example.com", "https://example.com"}} {
		c := Default()
		c.API.CORSOrigins = origins
		if c.Validate() == nil {
			t.Fatalf("accepted origins %q", origins)
		}
	}
	c := Default()
	c.API.CORSOrigins = []string{}
	if c.Validate() != nil {
		t.Fatal("empty CORS list rejected")
	}
}
func TestEnvironmentOverrides(t *testing.T) {
	env := map[string]string{"BASESTACK_API_HOST": "::1", "BASESTACK_API_PORT": "6000", "BASESTACK_DATABASE_PORT": "6001", "BASESTACK_DB_PASSWORD": "private@value", "BASESTACK_CORS_ORIGINS": "https://example.com,http://localhost:3000"}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }
	c, err := Resolve(Default(), lookup)
	if err != nil {
		t.Fatal(err)
	}
	if c.Host != "::1" || c.Port != 6000 || c.DatabasePort != 6001 || len(c.Origins) != 2 || !strings.Contains(c.DatabaseURL, "private%40value@127.0.0.1:6001") {
		t.Fatal("overrides not applied")
	}
	env["BASESTACK_DATABASE_URL"] = "postgres://external/database"
	c, err = Resolve(Default(), lookup)
	if err != nil || c.DatabaseURL != env["BASESTACK_DATABASE_URL"] {
		t.Fatal("explicit URL not used")
	}
	for _, key := range []string{"BASESTACK_API_HOST", "BASESTACK_API_PORT", "BASESTACK_DATABASE_PORT", "BASESTACK_CORS_ORIGINS"} {
		previous := env[key]
		env[key] = "INVALID_PRIVATE_VALUE"
		if _, err := Resolve(Default(), lookup); err == nil || strings.Contains(err.Error(), env[key]) {
			t.Fatalf("invalid %s accepted or leaked", key)
		}
		env[key] = previous
	}
	env["BASESTACK_CORS_ORIGINS"] = ""
	c, err = Resolve(Default(), lookup)
	if err != nil || len(c.Origins) != 0 {
		t.Fatal("CORS disable failed")
	}
}
func TestEnvFilesPrecedenceAndErrors(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".basestack"), 0700)
	os.WriteFile(filepath.Join(dir, LocalEnv), []byte("BASESTACK_TEST_VALUE=local\nBASESTACK_TEST_QUOTED='literal$VALUE'\n"), 0600)
	os.WriteFile(filepath.Join(dir, ".env"), []byte("BASESTACK_TEST_VALUE=override\n"), 0600)
	values, err := Environment(dir)
	if err != nil || values["BASESTACK_TEST_VALUE"] != "override" || values["BASESTACK_TEST_QUOTED"] != "literal$VALUE" {
		t.Fatal("file precedence/literal parsing failed")
	}
	t.Setenv("BASESTACK_TEST_VALUE", "process")
	values, err = Environment(dir)
	if err != nil || values["BASESTACK_TEST_VALUE"] != "process" {
		t.Fatal("process precedence failed")
	}
	for _, data := range []string{"KEY=one\nKEY=secret", "export KEY=secret", "KEY='secret", "secret"} {
		os.WriteFile(filepath.Join(dir, ".env"), []byte(data), 0600)
		if _, err := Environment(dir); err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatal("invalid dotenv accepted or value leaked")
		}
	}
}

func TestAuthConfigVersionsAndProduction(t *testing.T) {
	c := Default()
	if c.SchemaVersion != 2 || c.Auth == nil || !Enabled(c.Auth.Enabled) || !Enabled(c.Auth.RequireEmailVerification) {
		t.Fatal("unsafe Auth defaults")
	}
	values := map[string]string{}
	lookup := func(k string) (string, bool) { v, ok := values[k]; return v, ok }
	resolved, err := Resolve(c, lookup)
	if err != nil || resolved.Environment != "production" || resolved.AuthDelivery != "none" {
		t.Fatal("default environment is not production")
	}
	values["BASESTACK_AUTH_DELIVERY"] = "local"
	if _, err := Resolve(c, lookup); err == nil {
		t.Fatal("production local delivery accepted")
	}
	values["BASESTACK_ENV"] = "development"
	if _, err := Resolve(c, lookup); err != nil {
		t.Fatal(err)
	}
	values["BASESTACK_AUTH_ARGON2_MEMORY_KIB"] = "1"
	if _, err := Resolve(c, lookup); err == nil {
		t.Fatal("weak Argon parameters accepted")
	}
	delete(values, "BASESTACK_AUTH_ARGON2_MEMORY_KIB")
	c.SchemaVersion = 1
	c.Auth = nil
	b, _ := json.Marshal(c)
	if _, err := Parse(b); err != nil {
		t.Fatal("old service config rejected")
	}
	b = []byte(strings.Replace(string(b), `"schemaVersion":1`, `"schemaVersion":1,"auth":null`, 1))
	if _, err := Parse(b); err == nil {
		t.Fatal("new auth field silently accepted in old schema")
	}
}
