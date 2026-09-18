package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var envKey = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// Environment does not mutate os.Environ. Process > named profile > .env > local.env.
func Environment(dir string) (map[string]string, error) {
	return EnvironmentWithStore(dir, nil)
}

// EnvironmentWithStore supports encrypted/external stores without changing consumers.
func EnvironmentWithStore(dir string, store SecretStore) (map[string]string, error) {
	values := map[string]string{}
	for _, path := range []string{LocalEnv, ".env"} {
		f, err := os.Open(filepath.Join(dir, path))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("cannot read %s", path)
		}
		scanner := bufio.NewScanner(f)
		line := 0
		seen := map[string]bool{}
		for scanner.Scan() {
			line++
			text := strings.TrimSpace(scanner.Text())
			if text == "" || strings.HasPrefix(text, "#") {
				continue
			}
			key, value, ok := strings.Cut(text, "=")
			if !ok || !envKey.MatchString(key) || seen[key] {
				f.Close()
				return nil, fmt.Errorf("invalid or duplicate environment entry in %s line %d", path, line)
			}
			value = strings.TrimSpace(value)
			if strings.HasPrefix(value, `"`) || strings.HasPrefix(value, "'") {
				quote := value[:1]
				if len(value) < 2 || !strings.HasSuffix(value, quote) {
					f.Close()
					return nil, fmt.Errorf("invalid quoting in %s line %d", path, line)
				}
				value = value[1 : len(value)-1]
			}
			values[key] = value
			seen[key] = true
		}
		err = scanner.Err()
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("cannot parse %s", path)
		}
	}
	name := "production"
	if v, ok := values["BASESTACK_ENV"]; ok {
		name = v
	}
	if v, ok := os.LookupEnv("BASESTACK_ENV"); ok {
		name = v
	}
	if !ValidEnvironment(name) {
		return nil, ErrProfile
	}
	if store == nil {
		store = FileSecrets{Dir: dir}
	}
	secrets, err := store.Read(name)
	if err != nil {
		return nil, fmt.Errorf("cannot load private environment profile")
	}
	for key, value := range secrets {
		if !validPrivateKey(key) || !validSecret(value) {
			return nil, ErrProfile
		}
		values[key] = value
	}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	return values, nil
}
