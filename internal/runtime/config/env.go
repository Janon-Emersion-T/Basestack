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

// Environment does not mutate os.Environ. Process variables override .env, which overrides local.env.
func Environment(dir string) (map[string]string, error) {
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
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	return values, nil
}
