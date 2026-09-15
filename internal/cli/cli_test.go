package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func inTemp(t *testing.T) {
	t.Helper()
	old, err := os.Getwd(); if err != nil { t.Fatal(err) }
	if err := os.Chdir(t.TempDir()); err != nil { t.Fatal(err) }
	t.Cleanup(func() { if err := os.Chdir(old); err != nil { t.Fatal(err) } })
}

func run(t *testing.T, args ...string) string {
	t.Helper(); var out bytes.Buffer
	if err := Run(args, &out); err != nil { t.Fatal(err) }
	return out.String()
}

func TestProjectLifecycle(t *testing.T) {
	inTemp(t)
	run(t, "init", "my-app")
	for _, file := range []string{"package.json", "index.html", "src/main.tsx", "src/sections.tsx", "src/styles.css", "tsconfig.json", ".gitignore", "basestack.json"} {
		if _, err := os.Stat(filepath.Join("my-app", file)); err != nil { t.Fatal(err) }
	}
	if err := os.Chdir("my-app"); err != nil { t.Fatal(err) }
	run(t, "check")
	for _, kind := range kinds { run(t, "add", kind) }
	run(t, "add", "hero")
	c, err := readConfig(); if err != nil { t.Fatal(err) }
	if len(c.Sections) != 10 { t.Fatalf("got %d sections", len(c.Sections)) }
	footer := -1
	for i, s := range c.Sections { if s.Type == "footer" { footer = i; break } }
	if footer != 8 { t.Fatalf("new content should precede original footer: %+v", c.Sections) }
	run(t, "remove", "hero-2")
	run(t, "check")
	if err := Run([]string{"remove", "missing"}, &bytes.Buffer{}); err == nil { t.Fatal("missing id accepted") }
	before, _ := os.ReadFile(configFile)
	if err := Run([]string{"add", "../../bad"}, &bytes.Buffer{}); err == nil { t.Fatal("unknown template accepted") }
	after, _ := os.ReadFile(configFile)
	if !bytes.Equal(before, after) { t.Fatal("failed add modified configuration") }
}

func TestInitNeverOverwrites(t *testing.T) {
	inTemp(t)
	run(t, "init", "existing")
	marker := filepath.Join("existing", "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0644); err != nil { t.Fatal(err) }
	if err := initProject("existing"); err == nil { t.Fatal("existing project overwritten") }
	if data, err := os.ReadFile(marker); err != nil || string(data) != "keep" { t.Fatal("user data lost") }
	for _, name := range []string{"../outside", "/tmp/outside", "UPPER", "", ".", "a/b", "a b", strings.Repeat("a", 51)} {
		if err := initProject(name); err == nil { t.Fatalf("unsafe name accepted: %q", name) }
	}
}

func TestInvalidConfig(t *testing.T) {
	inTemp(t)
	for _, input := range []string{
		`{}`, `{"schemaVersion":2,"name":"app","sections":[]}`,
		`{"schemaVersion":1,"name":"app","sections":null}`,
		`{"schemaVersion":1,"name":"app","sections":[],"typo":true}`,
		`{"schemaVersion":1,"name":"app","sections":[]} {}`,
		`{"schemaVersion":1,"name":"app","sections":[{"id":"x","type":"bad","title":"X","text":""}]}`,
		`{"schemaVersion":1,"name":"app","sections":[{"id":"x","type":"hero","title":" ","text":""}]}`,
		`{"schemaVersion":1,"name":"app","sections":[{"id":"x","type":"hero","title":"A","text":""},{"id":"x","type":"footer","title":"B","text":""}]}`,
	} {
		if err := os.WriteFile(configFile, []byte(input), 0644); err != nil { t.Fatal(err) }
		if _, err := readConfig(); err == nil { t.Fatalf("invalid configuration accepted: %s", input) }
	}
}

func TestRemoveAllAndReAdd(t *testing.T) {
	inTemp(t); run(t, "init", "empty-app")
	if err := os.Chdir("empty-app"); err != nil { t.Fatal(err) }
	for _, kind := range []string{"navbar", "hero", "footer"} { run(t, "remove", kind) }
	run(t, "check"); run(t, "add", "hero"); run(t, "check")
}

func TestScaffoldPackageAndHelp(t *testing.T) {
	inTemp(t); run(t, "init", "example")
	data, err := os.ReadFile("example/package.json"); if err != nil { t.Fatal(err) }
	var pkg map[string]any
	if err := json.Unmarshal(data, &pkg); err != nil { t.Fatal(err) }
	if pkg["name"] != "example" { t.Fatal("project name not substituted") }
	if !strings.Contains(run(t), "init <name>") { t.Fatal("missing help") }
	if err := Run([]string{"unknown"}, &bytes.Buffer{}); err == nil { t.Fatal("unknown command accepted") }
}
