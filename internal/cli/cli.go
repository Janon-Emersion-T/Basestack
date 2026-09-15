package cli

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed scaffold/* scaffold/src/*
var scaffold embed.FS

const Version = "0.1.0"
const configFile = "basestack.json"

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,49}$`)
var kinds = []string{"navbar", "hero", "features", "pricing", "contact", "footer"}

type Section struct {
	ID string `json:"id"`
	Type string `json:"type"`
	Title string `json:"title"`
	Text string `json:"text"`
}

func (s *Section) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID *string `json:"id"`
		Type *string `json:"type"`
		Title *string `json:"title"`
		Text *string `json:"text"`
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&raw); err != nil { return err }
	if raw.ID == nil || raw.Type == nil || raw.Title == nil || raw.Text == nil { return errors.New("each section requires string id, type, title and text fields") }
	*s = Section{ID:*raw.ID, Type:*raw.Type, Title:*raw.Title, Text:*raw.Text}
	return nil
}

type Config struct {
	SchemaVersion int `json:"schemaVersion"`
	Name string `json:"name"`
	Sections []Section `json:"sections"`
}

func known(kind string) bool {
	for _, k := range kinds { if k == kind { return true } }
	return false
}

func defaults(kind, id string) Section {
	copy := map[string][2]string{
		"navbar": {"Your brand", "Built with BaseStack"},
		"hero": {"Your next idea starts here.", "Create something useful. Assemble your page from ready-made sections and make it your own."},
		"features": {"Everything starts with a solid foundation.", "Reusable sections. Responsive layouts. Source code you control."},
		"pricing": {"A plan that grows with you.", "Describe your offer and pricing here."},
		"contact": {"Let’s build something together.", "Replace the email address below with your business email."},
		"footer": {"Your brand", "Crafted with care. Powered by BaseStack."},
	}
	v := copy[kind]
	return Section{ID:id, Type:kind, Title:v[0], Text:v[1]}
}

func validate(c Config) error {
	if c.SchemaVersion != 1 { return errors.New("unsupported schemaVersion; expected 1") }
	if !namePattern.MatchString(c.Name) { return errors.New("name must start with a lowercase letter and contain only lowercase letters, digits or hyphens (max 50 characters)") }
	if c.Sections == nil { return errors.New("sections must be an array") }
	seen := map[string]bool{}
	for _, s := range c.Sections {
		if !known(s.Type) { return fmt.Errorf("unknown section type %q", s.Type) }
		if !namePattern.MatchString(s.ID) || seen[s.ID] { return fmt.Errorf("invalid or duplicate section id %q", s.ID) }
		seen[s.ID] = true
		if strings.TrimSpace(s.Title) == "" { return fmt.Errorf("section %q needs a title", s.ID) }
	}
	return nil
}

func readConfig() (Config, error) {
	var c Config
	f, err := os.Open(configFile)
	if err != nil { return c, fmt.Errorf("open basestack.json (run this inside a generated project): %w", err) }
	defer f.Close()
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()
	if err = d.Decode(&c); err != nil { return c, err }
	var extra any
	if err = d.Decode(&extra); err != io.EOF { return c, errors.New("configuration must contain exactly one JSON object") }
	return c, validate(c)
}

func writeConfig(c Config) error {
	if err := validate(c); err != nil { return err }
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil { return err }
	f, err := os.CreateTemp(".", ".basestack-*.tmp")
	if err != nil { return err }
	defer os.Remove(f.Name())
	if _, err = f.Write(append(b, '\n')); err != nil { f.Close(); return err }
	if err = f.Close(); err != nil { return err }
	return os.Rename(f.Name(), configFile)
}

func initProject(name string) error {
	if !namePattern.MatchString(name) { return errors.New("use a project name such as my-app (lowercase letters, digits and hyphens, max 50 characters)") }
	if err := os.Mkdir(name, 0755); err != nil { return fmt.Errorf("create project directory (existing paths are never overwritten): %w", err) }
	complete := false
	defer func() { if !complete { os.RemoveAll(name) } }()
	err := fs.WalkDir(scaffold, "scaffold", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil { return walkErr }
		if entry.IsDir() { return nil }
		rel := strings.TrimPrefix(path, "scaffold/")
		if rel == "gitignore" { rel = ".gitignore" }
		dest := filepath.Join(name, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil { return err }
		data, err := scaffold.ReadFile(path)
		if err != nil { return err }
		data = []byte(strings.ReplaceAll(string(data), "__PROJECT_NAME__", name))
		return os.WriteFile(dest, data, 0644)
	})
	if err != nil { return err }
	c := Config{SchemaVersion:1, Name:name, Sections:[]Section{defaults("navbar", "navbar"), defaults("hero", "hero"), defaults("footer", "footer")}}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil { return err }
	if err = os.WriteFile(filepath.Join(name, configFile), append(b, '\n'), 0644); err != nil { return err }
	complete = true
	return nil
}

func Run(args []string, out io.Writer) error {
	if len(args) == 0 { args = []string{"help"} }
	switch args[0] {
	case "help", "--help", "-h":
		fmt.Fprintln(out, "BaseStack — build from reusable sections, without AI.\n\nCommands:\n  init <name>       Create a React + TypeScript project\n  templates         List built-in sections\n  add <type>        Insert a section before the footer\n  remove <id>       Remove a section\n  check             Validate basestack.json\n  dev               Start the local Vite server\n  build             Validate and build for production\n  version           Print CLI version\n\nRun add/remove/check/dev/build inside a generated project.\nInstall frontend dependencies with npm install before dev/build.")
		return nil
	case "version", "--version":
		fmt.Fprintln(out, Version); return nil
	case "templates":
		for _, k := range kinds { fmt.Fprintln(out, k) }; return nil
	case "init":
		if len(args) != 2 { return errors.New("usage: basestack init <name>") }
		if err := initProject(args[1]); err != nil { return err }
		fmt.Fprintf(out, "Created %s.\n\nNext:\n  cd %s\n  npm install\n  basestack dev\n\nEdit basestack.json to customise content and order.\n", args[1], args[1]); return nil
	case "add", "remove":
		if len(args) != 2 { return fmt.Errorf("usage: basestack %s <%s>", args[0], map[string]string{"add":"type", "remove":"id"}[args[0]]) }
		c, err := readConfig(); if err != nil { return err }
		if args[0] == "add" {
			kind := args[1]; if !known(kind) { return fmt.Errorf("unknown template %q; run basestack templates", kind) }
			used := map[string]bool{}; for _, s := range c.Sections { used[s.ID] = true }
			id := kind; for n := 2; used[id]; n++ { id = fmt.Sprintf("%s-%d", kind, n) }
			s := defaults(kind, id)
			pos := len(c.Sections)
			if kind != "footer" { for i, existing := range c.Sections { if existing.Type == "footer" { pos = i; break } } }
			c.Sections = append(c.Sections, Section{})
			copy(c.Sections[pos+1:], c.Sections[pos:])
			c.Sections[pos] = s
			if err := writeConfig(c); err != nil { return err }; fmt.Fprintln(out, "Added", id)
		} else {
			found := false
			for i, s := range c.Sections { if s.ID == args[1] { c.Sections = append(c.Sections[:i], c.Sections[i+1:]...); found = true; break } }
			if !found { return fmt.Errorf("section %q not found", args[1]) }
			if err := writeConfig(c); err != nil { return err }; fmt.Fprintln(out, "Removed", args[1])
		}
		return nil
	case "check", "dev", "build":
		if len(args) != 1 { return fmt.Errorf("usage: basestack %s", args[0]) }
		if _, err := readConfig(); err != nil { return err }
		if args[0] == "check" { fmt.Fprintln(out, "Configuration is valid."); return nil }
		if _, err := exec.LookPath("npm"); err != nil { return errors.New("npm is required; install Node.js 22 or newer") }
		cmd := exec.Command("npm", "run", args[0]); cmd.Stdin = os.Stdin; cmd.Stdout = out; cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return fmt.Errorf("unknown command %q; run basestack help", args[0])
	}
}
