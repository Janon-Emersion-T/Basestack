// Package migrations manages ordered SQL migrations and explicit, checksummed rollback.
package migrations

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const Directory = "basestack/migrations"

type File struct {
	Version      int64
	Name         string
	SQL          string
	Checksum     string
	DownSQL      string
	DownChecksum string
}

var filename = regexp.MustCompile(`^([0-9]{6})_([a-z][a-z0-9_]{0,63})\.sql$`)
var migrationName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func Discover(dir string) ([]File, error) {
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, fmt.Errorf("open migrations directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("migrations must be a real directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	result := []File{}
	versions, names := map[int64]bool{}, map[string]bool{}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".down.sql") {
			up := strings.TrimSuffix(entry.Name(), ".down.sql") + ".sql"
			if !filename.MatchString(up) {
				return nil, fmt.Errorf("invalid down migration filename")
			}
			if info, err := os.Lstat(filepath.Join(dir, up)); err != nil || !info.Mode().IsRegular() {
				return nil, fmt.Errorf("orphan down migration")
			}
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		parts := filename.FindStringSubmatch(entry.Name())
		if parts == nil {
			return nil, fmt.Errorf("invalid migration filename %q; expected 000001_lowercase_name.sql", entry.Name())
		}
		version, _ := strconv.ParseInt(parts[1], 10, 64)
		if version == 0 || versions[version] || names[parts[2]] {
			return nil, fmt.Errorf("duplicate or invalid migration version/name: %s", entry.Name())
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("migration %s must be a regular file", entry.Name())
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if err := transactionFree(string(data)); err != nil {
			return nil, fmt.Errorf("migration %s: %w", entry.Name(), err)
		}
		file := File{Version: version, Name: entry.Name(), SQL: string(data), Checksum: fmt.Sprintf("%x", sha256.Sum256(data))}
		down := filepath.Join(dir, strings.TrimSuffix(entry.Name(), ".sql")+".down.sql")
		if info, err := os.Lstat(down); err == nil {
			if !info.Mode().IsRegular() {
				return nil, fmt.Errorf("down migration must be a regular file")
			}
			data, err := os.ReadFile(down)
			if err != nil {
				return nil, err
			}
			if len(strings.TrimSpace(string(data))) == 0 {
				return nil, fmt.Errorf("down migration must not be empty")
			}
			if err := transactionFree(string(data)); err != nil {
				return nil, err
			}
			file.DownSQL = string(data)
			file.DownChecksum = fmt.Sprintf("%x", sha256.Sum256(data))
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		result = append(result, file)
		versions[version] = true
		names[parts[2]] = true
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Version < result[j].Version })
	return result, nil
}
func Create(dir, name string) (string, error) {
	if !migrationName.MatchString(name) {
		return "", fmt.Errorf("migration name must start with a lowercase letter and contain lowercase letters, digits or underscores (max 64)")
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("migrations directory is missing or unsafe")
	}
	lock, err := os.OpenFile(filepath.Join(dir, ".create.lock"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", fmt.Errorf("migration creation is locked; retry or inspect %s/.create.lock", dir)
	}
	lock.Close()
	defer os.Remove(lock.Name())
	files, err := Discover(dir)
	if err != nil {
		return "", err
	}
	next := int64(1)
	for _, file := range files {
		if filename.FindStringSubmatch(file.Name)[2] == name {
			return "", fmt.Errorf("migration name %q already exists", name)
		}
		next = file.Version + 1
	}
	if next > 999999 {
		return "", fmt.Errorf("migration sequence is exhausted")
	}
	path := filepath.Join(dir, fmt.Sprintf("%06d_%s.sql", next, name))
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return "", err
	}
	complete := false
	defer func() {
		f.Close()
		if !complete {
			os.Remove(path)
		}
	}()
	_, err = fmt.Fprintf(f, "-- %s\n-- Forward-only SQL. BaseStack wraps the batch in a transaction.\n-- Do not add BEGIN/COMMIT or edit this file after it has been applied.\n\n", name)
	if err != nil {
		return "", err
	}
	if err = f.Sync(); err != nil {
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	complete = true
	return path, nil
}
