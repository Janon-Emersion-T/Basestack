package project

import (
	"fmt"
	"os"
	"strings"
)

// EnsurePrivateIgnore also protects named profiles used in older frontend-only projects.
func EnsurePrivateIgnore() error {
	path := ".gitignore"
	var data []byte
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf(".gitignore must be a regular file before creating private state")
		}
		data, err = os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("cannot read .gitignore")
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("cannot inspect .gitignore")
	}
	lines := strings.Split(string(data), "\n")
	// Append at the end so a prior negation cannot accidentally expose the directory.
	if len(lines) >= 2 && lines[len(lines)-2] == "/.basestack/" {
		return nil
	}
	f, err := os.CreateTemp(".", ".gitignore-*")
	if err != nil {
		return fmt.Errorf("cannot update .gitignore")
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err = f.Write(append(data, []byte("\n# Private BaseStack application state\n/.basestack/\n")...)); err != nil {
		return fmt.Errorf("cannot update .gitignore")
	}
	if f.Chmod(0644) != nil || f.Sync() != nil || f.Close() != nil {
		return fmt.Errorf("cannot persist .gitignore")
	}
	if os.Rename(f.Name(), path) != nil {
		return fmt.Errorf("cannot replace .gitignore")
	}
	return nil
}
