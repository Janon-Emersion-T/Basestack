package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidation(t *testing.T) {
	cases := map[string]func(map[string]any){
		"case-insensitive root": func(c map[string]any) { c["Theme"] = "default" },
		"case-insensitive page": func(c map[string]any) { page(c)["Title"] = "Home" },
		"unknown root":          func(c map[string]any) { c["oops"] = true },
		"old version":           func(c map[string]any) { c["schemaVersion"] = 1 },
		"future version":        func(c map[string]any) { c["schemaVersion"] = 3 },
		"null version":          func(c map[string]any) { c["schemaVersion"] = nil },
		"bad name":              func(c map[string]any) { c["name"] = "../bad" },
		"theme":                 func(c map[string]any) { c["theme"] = "bad" },
		"missing theme":         func(c map[string]any) { delete(c, "theme") },
		"empty pages":           func(c map[string]any) { c["pages"] = []any{} },
		"null pages":            func(c map[string]any) { c["pages"] = nil },
		"duplicate page":        func(c map[string]any) { c["pages"] = append(c["pages"].([]any), page(c)) },
		"duplicate path": func(c map[string]any) {
			c["pages"] = append(c["pages"].([]any), map[string]any{"id": "other", "path": "/", "title": "Other", "sections": []any{}})
		},
		"unsafe page":       func(c map[string]any) { page(c)["id"] = "../bad" },
		"unknown page":      func(c map[string]any) { page(c)["extra"] = true },
		"blank title":       func(c map[string]any) { page(c)["title"] = " " },
		"null sections":     func(c map[string]any) { page(c)["sections"] = nil },
		"null section":      func(c map[string]any) { page(c)["sections"] = []any{nil} },
		"duplicate section": func(c map[string]any) { page(c)["sections"] = []any{section(c), section(c)} },
		"unsafe section":    func(c map[string]any) { section(c)["id"] = "bad/id" },
		"reserved section":  func(c map[string]any) { section(c)["id"] = "content" },
		"unknown section":   func(c map[string]any) { section(c)["extra"] = true },
		"unknown type":      func(c map[string]any) { section(c)["type"] = "bad" },
		"invalid variant":   func(c map[string]any) { section(c)["variant"] = "split" },
		"missing variant":   func(c map[string]any) { delete(section(c), "variant") },
		"null props":        func(c map[string]any) { section(c)["props"] = nil },
		"array props":       func(c map[string]any) { section(c)["props"] = []any{} },
		"unknown prop":      func(c map[string]any) { props(c)["typo"] = true },
		"wrong prop type":   func(c map[string]any) { props(c)["title"] = 4 },
		"null subtitle":     func(c map[string]any) { props(c)["subtitle"] = nil },
		"missing prop":      func(c map[string]any) { delete(props(c), "subtitle") },
		"malformed link": func(c map[string]any) {
			props(c)["links"] = []any{map[string]any{"label": "X", "href": "javascript:alert(1)"}}
		},
		"unknown nested prop": func(c map[string]any) {
			props(c)["links"] = []any{map[string]any{"label": "X", "href": "/", "extra": 1}}
		},
		"missing nested prop": func(c map[string]any) { props(c)["links"] = []any{map[string]any{"label": "X"}} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			b, _ := json.Marshal(New("app"))
			var c map[string]any
			json.Unmarshal(b, &c)
			mutate(c)
			b, _ = json.Marshal(c)
			if _, err := Parse(b); err == nil {
				t.Fatalf("accepted %s", b)
			}
		})
	}
	for _, path := range []string{"about", "//about", "/about/", "/../about", "/a//b", "/a?b", "/a#b", "/a%2fb", "/A", "/a b", "/a\\b"} {
		c := New("app")
		c.Pages[0].Path = path
		if err := Validate(c); err == nil {
			t.Fatalf("accepted path %q", path)
		}
	}
	for _, path := range []string{"/", "/about", "/services/web-design", "/v2/item_1"} {
		c := New("app")
		c.Pages[0].Path = path
		if err := Validate(c); err != nil {
			t.Fatal(err)
		}
	}
}
func page(c map[string]any) map[string]any    { return c["pages"].([]any)[0].(map[string]any) }
func section(c map[string]any) map[string]any { return page(c)["sections"].([]any)[0].(map[string]any) }
func props(c map[string]any) map[string]any   { return section(c)["props"].(map[string]any) }
func TestStrictJSON(t *testing.T) {
	b, _ := json.Marshal(New("app"))
	valid := string(b)
	for _, input := range []string{"", "null", "[]", "{", valid + " {}", strings.Replace(valid, `"schemaVersion":2`, `"schemaVersion":2,"schemaVersion":2`, 1), strings.Replace(valid, `"title":"Your brand"`, `"title":"Your brand","title":"Oops"`, 1)} {
		if _, err := Parse([]byte(input)); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
	if _, err := Parse(b); err != nil {
		t.Fatal(err)
	}
}
func TestAtomicWriteProtection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFile)
	if err := WriteAt(path, New("app")); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	bad := New("app")
	bad.Pages = nil
	if err := WriteAt(path, bad); err == nil {
		t.Fatal("invalid write accepted")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("invalid write damaged file")
	}
	target := filepath.Join(dir, "link")
	if err := os.Symlink(path, target); err != nil {
		t.Fatal(err)
	}
	if err := WriteAt(target, New("other")); err == nil {
		t.Fatal("symlink replaced")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Fatal("temporary files leaked")
	}
}
