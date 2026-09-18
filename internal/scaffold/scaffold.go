// Package scaffold copies independently usable frontend source into a new directory.
package scaffold

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Janon-Emersion-T/Basestack/internal/project"
	"github.com/Janon-Emersion-T/Basestack/internal/registry"
	runtimesource "github.com/Janon-Emersion-T/Basestack/internal/runtime"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/auth"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/config"
	"github.com/Janon-Emersion-T/Basestack/internal/strictjson"
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
	instance := make([]byte, 6)
	if _, err := rand.Read(instance); err != nil {
		return err
	}
	err := fs.WalkDir(files, "template", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(path, "template/")
		if rel == "env_example" {
			rel = ".env.example"
		}
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
		return os.WriteFile(dest, []byte(strings.ReplaceAll(strings.ReplaceAll(string(data), "__PROJECT_NAME__", name), "__COMPOSE_ID__", hex.EncodeToString(instance))), 0644)
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
	if err = os.WriteFile(filepath.Join(name, "basestack", "migrations", "000002_auth_core.sql"), []byte(auth.SchemaSQL), 0644); err != nil {
		return err
	}
	services, err := json.MarshalIndent(config.Default(), "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(name, config.File), append(services, '\n'), 0644); err != nil {
		return err
	}
	if err = copyRuntime(name, runtimesource.Source, ""); err != nil {
		return err
	}
	if err = copyRuntime(name, strictjson.Source, "internal/strictjson/"); err != nil {
		return err
	}
	complete = true
	return nil
}

// Runtime files are the same sources tested in this repository, with only the module path changed.
func copyRuntime(name string, source fs.FS, prefix string) error {
	return fs.WalkDir(source, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		dest := prefix + path
		if prefix == "" {
			dest = "internal/runtime/" + path
			switch path {
			case "module.txt":
				dest = "go.mod"
			case "sums.txt":
				dest = "go.sum"
			case "command/main.go":
				dest = "cmd/server/main.go"
			}
		}
		data, err := fs.ReadFile(source, path)
		if err != nil {
			return err
		}
		data = []byte(strings.ReplaceAll(string(data), "github.com/Janon-Emersion-T/Basestack", "example.com/"+name))
		target := filepath.Join(name, filepath.FromSlash(dest))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}
