package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/Janon-Emersion-T/Basestack/internal/runtime/privatefs"
	"github.com/Janon-Emersion-T/Basestack/internal/strictjson"
)

var ErrProfile = errors.New("invalid environment profile; use development, test or production and valid private keys/values")

func ValidEnvironment(name string) bool {
	return name == "development" || name == "test" || name == "production"
}

// SecretStore is the replacement boundary for a future encrypted secret store.
// Callers must never log returned values or provider errors containing values.
type SecretStore interface {
	Read(name string) (map[string]string, error)
	Set(name, key, value string) error
	Unset(name, key string) error
}

type FileSecrets struct{ Dir string }
type profile struct {
	SchemaVersion int               `json:"schemaVersion"`
	Values        map[string]string `json:"values"`
}

func validPrivateKey(key string) bool {
	return len(key) <= 128 && envKey.MatchString(key) && key != "BASESTACK_ENV" && !strings.HasPrefix(key, "VITE_")
}
func validSecret(value string) bool {
	return len(value) <= 65536 && utf8.ValidString(value) && !strings.ContainsRune(value, 0)
}

func (s FileSecrets) Read(name string) (map[string]string, error) {
	if !ValidEnvironment(name) {
		return nil, ErrProfile
	}
	dir, err := privatefs.Directory(s.Dir, false, ".basestack", "environments")
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, name+".json")
	if err := privatefs.Regular(path); os.IsNotExist(err) {
		return map[string]string{}, nil
	} else if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, privatefs.ErrUnsafe
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil || len(b) > 1<<20 {
		return nil, ErrProfile
	}
	var p profile
	if strictjson.Decode(b, &p) != nil || p.SchemaVersion != 1 || p.Values == nil || len(p.Values) > 128 {
		return nil, ErrProfile
	}
	var shape struct {
		Values map[string]any `json:"values"`
	}
	if json.Unmarshal(b, &shape) != nil {
		return nil, ErrProfile
	}
	for _, value := range shape.Values {
		if _, ok := value.(string); !ok {
			return nil, ErrProfile
		}
	}
	for key, value := range p.Values {
		if !validPrivateKey(key) || !validSecret(value) {
			return nil, ErrProfile
		}
	}
	return p.Values, nil
}

func (s FileSecrets) Set(name, key, value string) error {
	if !validSecret(value) {
		return ErrProfile
	}
	return s.update(name, key, &value)
}
func (s FileSecrets) Unset(name, key string) error { return s.update(name, key, nil) }
func (s FileSecrets) update(name, key string, value *string) error {
	if !ValidEnvironment(name) || !validPrivateKey(key) {
		return ErrProfile
	}
	dir, err := privatefs.Directory(s.Dir, true, ".basestack", "environments")
	if err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(dir, name+".lock"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("environment update is locked; retry or inspect its private lock file")
	}
	lock.Close()
	defer os.Remove(lock.Name())
	values, err := s.Read(name)
	if err != nil {
		return err
	}
	if value == nil {
		delete(values, key)
	} else {
		values[key] = *value
	}
	if len(values) > 128 {
		return ErrProfile
	}
	b, err := json.MarshalIndent(profile{1, values}, "", "  ")
	if err != nil || len(b)+1 > 1<<20 {
		return ErrProfile
	}
	f, err := os.CreateTemp(dir, ".profile-*")
	if err != nil {
		return privatefs.ErrUnsafe
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err = f.Write(append(b, '\n')); err != nil {
		return privatefs.ErrUnsafe
	}
	if err = f.Sync(); err != nil {
		return privatefs.ErrUnsafe
	}
	if err = f.Close(); err != nil {
		return privatefs.ErrUnsafe
	}
	if err = os.Rename(f.Name(), filepath.Join(dir, name+".json")); err != nil {
		return privatefs.ErrUnsafe
	}
	return nil
}
