package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestPageLifecycleAndIsolation(t *testing.T) {
	inTemp(t); run(t, "init", "site")
	if err := os.Chdir("site"); err != nil { t.Fatal(err) }
	before, err := readConfig(); if err != nil { t.Fatal(err) }
	home, _ := json.Marshal(before.Sections)
	run(t, "page", "add", "about", "--title", "About our team")
	run(t, "add", "hero", "--page", "about", "--variant", "split")
	run(t, "add", "features", "--variant", "list", "--page", "about")
	run(t, "remove", "hero", "--page", "about")
	c, err := readConfig(); if err != nil { t.Fatal(err) }
	afterHome, _ := json.Marshal(c.Sections)
	if !bytes.Equal(home, afterHome) { t.Fatal("page mutation changed home") }
	if len(c.Pages) != 1 || c.Pages[0].Title != "About our team" { t.Fatal("page not created") }
	sections := c.Pages[0].Sections
	if sections[1].ID != "hero-2" || sections[1].Variant != "split" || sections[2].Variant != "list" || sections[3].Type != "footer" { t.Fatalf("unexpected page sections: %+v", sections) }
	if !strings.Contains(run(t, "page", "list"), "?page=about") { t.Fatal("page missing from list") }
	run(t, "page", "remove", "about")
	c, err = readConfig(); if err != nil { t.Fatal(err) }
	if len(c.Pages) != 0 { t.Fatal("page was not removed") }
	run(t, "page", "add", "about")
	run(t, "check")
}

func TestFailedCommandsPreserveManifest(t *testing.T) {
	inTemp(t); run(t, "init", "site")
	if err := os.Chdir("site"); err != nil { t.Fatal(err) }
	run(t, "page", "add", "about")
	before, err := os.ReadFile(configFile); if err != nil { t.Fatal(err) }
	for _, args := range [][]string{
		{"page"}, {"page", "nope"}, {"page", "add"}, {"page", "add", "home"},
		{"page", "add", "../bad"}, {"page", "add", "about"}, {"page", "remove", "home"},
		{"page", "remove", "missing"}, {"page", "list", "extra"}, {"page", "add", "team", "--title"},
		{"page", "add", "team", "--title", " "}, {"page", "add", "team", "--typo", "x"},
		{"add", "hero", "--variant", "list"}, {"add", "hero", "--page", "missing"},
		{"add", "hero", "--page"}, {"add", "hero", "--page", "about", "--page", "home"},
		{"add", "hero", "--variant", ""}, {"add", "hero", "--typo", "x"},
		{"remove", "hero", "--variant", "split"}, {"remove", "missing", "--page", "about"},
	} {
		if err := Run(args, &bytes.Buffer{}); err == nil { t.Errorf("accepted invalid command: %v", args) }
		after, err := os.ReadFile(configFile); if err != nil { t.Fatal(err) }
		if !bytes.Equal(before, after) { t.Fatalf("failed command modified manifest: %v", args) }
	}
}

func TestLegacyConfigAndAllVariants(t *testing.T) {
	inTemp(t); run(t, "init", "legacy")
	if err := os.Chdir("legacy"); err != nil { t.Fatal(err) }
	// Existing manifests omit pages and variant; they remain valid with no migration.
	run(t, "check")
	for kind, variant := range variants { run(t, "add", kind, "--variant", variant) }
	run(t, "check")
	c, err := readConfig(); if err != nil { t.Fatal(err) }
	if c.Sections[0].Type != "navbar" { t.Fatal("navbar was not inserted at the beginning") }
	if !strings.Contains(run(t, "templates"), "hero: default, split") { t.Fatal("missing variant catalogue") }
}

func TestSharedConfigFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/config-cases.json"); if err != nil { t.Fatal(err) }
	var cases []struct { Name string `json:"name"`; Valid bool `json:"valid"`; Config json.RawMessage `json:"config"` }
	if err := json.Unmarshal(data, &cases); err != nil { t.Fatal(err) }
	for _, fixture := range cases {
		t.Run(fixture.Name, func(t *testing.T) {
			var c Config
			err := json.Unmarshal(fixture.Config, &c)
			if err == nil { err = validate(c) }
			if (err == nil) != fixture.Valid { t.Fatalf("valid=%v; error=%v", fixture.Valid, err) }
		})
	}
}
