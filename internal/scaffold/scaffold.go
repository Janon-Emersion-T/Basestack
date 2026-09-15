// Package scaffold copies independently usable frontend source into a new directory.
package scaffold

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Janon-Emersion-T/Basestack/internal/project"
	"github.com/Janon-Emersion-T/Basestack/internal/registry"
)

//go:embed template/* template/src/*
var files embed.FS

func Create(name string) error {
	if !project.NamePattern.MatchString(name) {
		return fmt.Errorf("use a project name such as my-app (lowercase letters, digits and hyphens, max 50 characters)")
	}
	if err := os.Mkdir(name, 0755); err != nil {
		return fmt.Errorf("create project directory (existing paths are never overwritten): %w", err)
	}
	complete := false
	defer func() {
		if !complete {
			os.RemoveAll(name)
		}
	}()
	err := fs.WalkDir(files, "template", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(path, "template/")
		if rel == "gitignore" {
			rel = ".gitignore"
		}
		dest := filepath.Join(name, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		data, err := files.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, []byte(strings.ReplaceAll(string(data), "__PROJECT_NAME__", name)), 0644)
	})
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(name, "src", "registry.json"), registry.BuiltinJSON, 0644); err != nil {
		return err
	}
	if err = project.WriteAt(filepath.Join(name, project.ConfigFile), project.New(name)); err != nil {
		return err
	}
	complete = true
	return nil
}
