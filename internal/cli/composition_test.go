package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/Janon-Emersion-T/Basestack/internal/project"
)

func read(t *testing.T) project.Config {
	t.Helper()
	c, err := project.Read()
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func rejectUnchanged(t *testing.T, args ...string) {
	t.Helper()
	before, err := os.ReadFile(project.ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(args, &bytes.Buffer{}); err == nil {
		t.Fatalf("accepted %v", args)
	}
	after, err := os.ReadFile(project.ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("failed command changed config: %v", args)
	}
}
func TestComposition(t *testing.T) {
	inTemp(t)
	run(t, "init", "business")
	if err := os.Chdir("business"); err != nil {
		t.Fatal(err)
	}
	c := read(t)
	if c.SchemaVersion != 2 || c.Theme != "default" || len(c.Pages) != 1 || c.Pages[0].ID != "home" || c.Pages[0].Path != "/" || c.Pages[0].Title != "Home" {
		t.Fatalf("unexpected defaults: %+v", c)
	}
	run(t, "page", "add", "about")
	run(t, "page", "add", "contact", "--path", "/get-in-touch", "--title", "Contact Us")
	c = read(t)
	if c.Pages[1].Path != "/about" || c.Pages[1].Title != "About" || len(c.Pages[1].Sections) != 0 || c.Pages[2].Title != "Contact Us" {
		t.Fatalf("page defaults/options: %+v", c.Pages)
	}
	for _, args := range [][]string{{"page", "add", "about"}, {"page", "add", "other", "--path", "/about"}, {"page", "add", "../bad"}, {"page", "add", "bad", "--path", "//bad"}, {"page", "remove", "missing"}, {"add", "hero", "--page", "missing"}, {"add", "hero", "--variant", "missing"}, {"add", "hero", "--variant", ""}, {"add", "hero", "--oops", "x"}, {"add", "hero", "--page", "home", "--page", "about"}, {"theme", "set", "unknown"}, {"remove", "hero", "--page", "about"}} {
		rejectUnchanged(t, args...)
	}
	run(t, "add", "hero", "--page", "about", "--variant", "split")
	run(t, "add", "contact", "--page", "contact", "--variant", "centered")
	run(t, "add", "features")
	c = read(t)
	if len(c.Pages[0].Sections) != 4 || c.Pages[0].Sections[2].Variant != "default" || c.Pages[1].Sections[0].Variant != "split" || c.Pages[2].Sections[0].Type != "contact" {
		t.Fatalf("page-aware add: %+v", c.Pages)
	}
	run(t, "remove", "hero", "--page", "about")
	if len(read(t).Pages[1].Sections) != 0 {
		t.Fatal("remove failed")
	}
	for _, theme := range []string{"modern", "minimal", "default"} {
		run(t, "theme", "set", theme)
		if read(t).Theme != theme {
			t.Fatal("theme not persisted")
		}
	}
	if !strings.Contains(run(t, "page", "list"), "/get-in-touch") {
		t.Fatal("missing page output")
	}
	run(t, "page", "remove", "home")
	rejectUnchanged(t, "add", "hero")
	run(t, "page", "remove", "about")
	run(t, "add", "hero")
	if read(t).Pages[0].Sections[1].Type != "hero" {
		t.Fatal("single-page fallback failed")
	}
	rejectUnchanged(t, "page", "remove", "contact")
}
func TestEveryVariant(t *testing.T) {
	inTemp(t)
	run(t, "init", "variants")
	if err := os.Chdir("variants"); err != nil {
		t.Fatal(err)
	}
	for kind, variant := range map[string]string{"navbar": "centered", "hero": "split", "features": "cards", "pricing": "simple", "contact": "centered", "footer": "columns"} {
		run(t, "add", kind, "--variant", variant)
		if !strings.Contains(run(t, "templates", kind), variant) {
			t.Fatal("variant not listed")
		}
	}
	run(t, "check")
	if !strings.Contains(run(t, "theme", "list"), "minimal") {
		t.Fatal("theme not listed")
	}
}
func TestCorruptCommandsDoNotWrite(t *testing.T) {
	inTemp(t)
	if err := os.WriteFile(project.ConfigFile, []byte(`{"schemaVersion":2,`), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "hero"}, {"remove", "hero"}, {"page", "add", "about"}, {"theme", "set", "modern"}, {"check"}, {"build"}, {"dev"}} {
		rejectUnchanged(t, args...)
	}
}
func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{{"init"}, {"init", "a", "b"}, {"add"}, {"remove"}, {"page"}, {"page", "list", "extra"}, {"page", "add"}, {"page", "remove"}, {"page", "nope"}, {"theme"}, {"theme", "set"}, {"templates", "hero", "extra"}, {"templates", "unknown"}, {"check", "extra"}, {"version", "extra"}} {
		if err := Run(args, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestFrontendCommands(t *testing.T) {
	inTemp(t)
	run(t, "init", "frontend")
	if err := os.Chdir("frontend"); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	t.Setenv("PATH", bin)
	if err := Run([]string{"dev"}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "npm is required") {
		t.Fatalf("unexpected missing npm error: %v", err)
	}
	if err := os.WriteFile(bin+"/npm", []byte("#!/bin/sh\nprintf '%s %s' \"$1\" \"$2\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"dev", "build"} {
		if got := run(t, command); !strings.HasSuffix(got, "run "+command) {
			t.Fatalf("wrong npm invocation: %q", got)
		}
	}
	if err := os.WriteFile(bin+"/npm", []byte("#!/bin/sh\nexit 7\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"build"}, &bytes.Buffer{}); err == nil {
		t.Fatal("npm failure hidden")
	}
}
