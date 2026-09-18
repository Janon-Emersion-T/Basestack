// Package config loads public service settings and private environment values.
package config

import (
	"encoding/json"
	"fmt"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/auth"
	"github.com/Janon-Emersion-T/Basestack/internal/strictjson"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const File = "basestack/services.json"
const LocalEnv = ".basestack/local.env"

type Database struct {
	Enabled *bool `json:"enabled"`
	Port    int   `json:"port"`
}
type API struct {
	Enabled     *bool    `json:"enabled"`
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	CORSOrigins []string `json:"corsOrigins"`
}
type Auth struct {
	Enabled                  *bool `json:"enabled"`
	RequireEmailVerification *bool `json:"requireEmailVerification"`
}
type Services struct {
	Auth          *Auth    `json:"auth,omitempty"`
	SchemaVersion int      `json:"schemaVersion"`
	Database      Database `json:"database"`
	API           API      `json:"api"`
}
type Runtime struct {
	Environment     string
	AuthDelivery    string
	AuthParameters  auth.Parameters
	VerificationTTL time.Duration
	ResetTTL        time.Duration
	Services        Services
	DatabaseURL     string
	DatabasePort    int
	Host            string
	Port            int
	Origins         []string
}

func Enabled(value *bool) bool { return value != nil && *value }
func Default() Services {
	yes, apiYes, authYes, verifyYes := true, true, true, true
	return Services{SchemaVersion: 2, Auth: &Auth{Enabled: &authYes, RequireEmailVerification: &verifyYes}, Database: Database{Enabled: &yes, Port: 54322}, API: API{Enabled: &apiYes, Host: "127.0.0.1", Port: 54321, CORSOrigins: []string{"http://localhost:5173", "http://127.0.0.1:5173"}}}
}
func Parse(data []byte) (Services, error) {
	var c Services
	if err := strictjson.Decode(data, &c); err != nil {
		return c, fmt.Errorf("invalid services configuration: %w", err)
	}
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(data, &raw)
	if c.SchemaVersion == 1 {
		if _, exists := raw["auth"]; exists {
			return c, fmt.Errorf("auth is not allowed in services schema v1")
		}
	}
	return c, c.Validate()
}
func Read(dir string) (Services, error) {
	data, err := os.ReadFile(filepath.Join(dir, File))
	if err != nil {
		return Services{}, fmt.Errorf("open %s: services require a 0.3 generated project; see docs/SERVICES.md", File)
	}
	return Parse(data)
}
func Port(value int) bool         { return value >= 1 && value <= 65535 }
func validHost(value string) bool { return value == "localhost" || net.ParseIP(value) != nil }
func Origins(values []string) error {
	if values == nil {
		return fmt.Errorf("api.corsOrigins must be an array (use [] to disable cross-origin access)")
	}
	seen := map[string]bool{}
	for _, origin := range values {
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") || strings.ContainsAny(origin, "*\\ \t\r\n") || seen[origin] {
			return fmt.Errorf("api.corsOrigins must contain unique explicit http(s) origins without paths or credentials")
		}
		seen[origin] = true
	}
	return nil
}
func (c Services) Validate() error {
	if c.SchemaVersion != 1 && c.SchemaVersion != 2 {
		return fmt.Errorf("unsupported services schemaVersion; expected 1 or 2")
	}
	if c.SchemaVersion == 1 && c.Auth != nil {
		return fmt.Errorf("auth is not allowed in services schema v1")
	}
	if c.SchemaVersion == 2 && (c.Auth == nil || c.Auth.Enabled == nil || c.Auth.RequireEmailVerification == nil) {
		return fmt.Errorf("services schema v2 requires auth.enabled and auth.requireEmailVerification booleans")
	}
	if c.Auth != nil && Enabled(c.Auth.Enabled) && !Enabled(c.Database.Enabled) {
		return fmt.Errorf("Auth requires database.enabled")
	}
	if c.Database.Enabled == nil || c.API.Enabled == nil {
		return fmt.Errorf("database.enabled and api.enabled require booleans")
	}
	if !Port(c.Database.Port) || !Port(c.API.Port) {
		return fmt.Errorf("database.port and api.port must be integers from 1 to 65535")
	}
	if !validHost(c.API.Host) {
		return fmt.Errorf("api.host must be an IP address or localhost")
	}
	return Origins(c.API.CORSOrigins)
}
func Load(dir string) (Runtime, error) {
	services, err := Read(dir)
	if err != nil {
		return Runtime{}, err
	}
	values, err := Environment(dir)
	if err != nil {
		return Runtime{}, err
	}
	return Resolve(services, func(key string) (string, bool) { v, ok := values[key]; return v, ok })
}

// Resolve accepts an injected environment lookup so tests never depend on developer credentials.
func Resolve(c Services, lookup func(string) (string, bool)) (Runtime, error) {
	if err := c.Validate(); err != nil {
		return Runtime{}, err
	}
	result := Runtime{Services: c, DatabasePort: c.Database.Port, Host: c.API.Host, Port: c.API.Port, Origins: append([]string{}, c.API.CORSOrigins...)}
	if v, ok := lookup("BASESTACK_API_HOST"); ok {
		if !validHost(v) {
			return result, fmt.Errorf("BASESTACK_API_HOST must be an IP address or localhost")
		}
		result.Host = v
	}
	for _, entry := range []struct {
		key    string
		target *int
	}{{"BASESTACK_API_PORT", &result.Port}, {"BASESTACK_DATABASE_PORT", &result.DatabasePort}} {
		if v, ok := lookup(entry.key); ok {
			p, err := strconv.Atoi(v)
			if err != nil || !Port(p) {
				return result, fmt.Errorf("%s must be an integer from 1 to 65535", entry.key)
			}
			*entry.target = p
		}
	}
	if v, ok := lookup("BASESTACK_CORS_ORIGINS"); ok {
		result.Origins = []string{}
		if v != "" {
			for _, origin := range strings.Split(v, ",") {
				result.Origins = append(result.Origins, strings.TrimSpace(origin))
			}
		}
		if err := Origins(result.Origins); err != nil {
			return result, err
		}
	}
	if v, ok := lookup("BASESTACK_DATABASE_URL"); ok {
		result.DatabaseURL = v
	} else if password, ok := lookup("BASESTACK_DB_PASSWORD"); ok && password != "" {
		u := url.URL{Scheme: "postgres", User: url.UserPassword("basestack", password), Host: net.JoinHostPort("127.0.0.1", strconv.Itoa(result.DatabasePort)), Path: "/basestack", RawQuery: "sslmode=disable"}
		result.DatabaseURL = u.String()
	}
	result.Environment = "production"
	result.AuthDelivery = "none"
	result.AuthParameters = auth.DefaultParameters()
	result.VerificationTTL = 24 * time.Hour
	result.ResetTTL = 30 * time.Minute
	if v, ok := lookup("BASESTACK_ENV"); ok {
		result.Environment = v
	}
	if result.Environment != "production" && result.Environment != "development" {
		return result, fmt.Errorf("BASESTACK_ENV must be production or development")
	}
	if v, ok := lookup("BASESTACK_AUTH_DELIVERY"); ok {
		result.AuthDelivery = v
	}
	if result.AuthDelivery != "none" && result.AuthDelivery != "local" && result.AuthDelivery != "external" {
		return result, fmt.Errorf("BASESTACK_AUTH_DELIVERY must be none, local or external")
	}
	if result.AuthDelivery == "local" && result.Environment != "development" {
		return result, fmt.Errorf("local Auth delivery is forbidden outside explicit development mode")
	}
	for _, entry := range []struct {
		key      string
		set      func(int)
		min, max int
	}{
		{"BASESTACK_AUTH_ARGON2_MEMORY_KIB", func(n int) { result.AuthParameters.Memory = uint32(n) }, 19 * 1024, 256 * 1024},
		{"BASESTACK_AUTH_ARGON2_ITERATIONS", func(n int) { result.AuthParameters.Iterations = uint32(n) }, 2, 6},
		{"BASESTACK_AUTH_ARGON2_PARALLELISM", func(n int) { result.AuthParameters.Parallelism = uint8(n) }, 1, 4},
		{"BASESTACK_AUTH_VERIFY_TTL_SECONDS", func(n int) { result.VerificationTTL = time.Duration(n) * time.Second }, 1, 86400},
		{"BASESTACK_AUTH_RESET_TTL_SECONDS", func(n int) { result.ResetTTL = time.Duration(n) * time.Second }, 1, 3600},
	} {
		if v, ok := lookup(entry.key); ok {
			n, err := strconv.Atoi(v)
			if err != nil || n < entry.min || n > entry.max {
				return result, fmt.Errorf("%s is outside supported bounds", entry.key)
			}
			entry.set(n)
		}
	}
	// DSN content is private. pgx validates it when opening the database.
	return result, nil
}
