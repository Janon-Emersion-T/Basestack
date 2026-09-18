// Package privatefs manages private local state under an owner-controlled project.
// It rejects symlinks and unsafe permissions; it is not a sandbox against an
// attacker who can concurrently replace the owner's directories.
package privatefs

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrUnsafe = errors.New("private state is unavailable or has unsafe permissions")

// Directory accepts fixed, application-owned path components, not user paths.
func Directory(root string, create bool, parts ...string) (string, error) {
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || filepath.Base(part) != part {
			return "", ErrUnsafe
		}
		root = filepath.Join(root, part)
		if create {
			if err := os.Mkdir(root, 0700); err != nil && !os.IsExist(err) {
				return "", ErrUnsafe
			}
		}
		info, err := os.Lstat(root)
		if os.IsNotExist(err) && !create {
			return "", os.ErrNotExist
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
			return "", ErrUnsafe
		}
	}
	return root, nil
}

func Regular(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return os.ErrNotExist
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return ErrUnsafe
	}
	return nil
}
