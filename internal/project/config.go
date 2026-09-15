package project

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Janon-Emersion-T/Basestack/internal/registry"
)

const ConfigFile = "basestack.json"

type Section struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Variant string          `json:"variant"`
	Props   json.RawMessage `json:"props"`
}
type Page struct {
	ID       string    `json:"id"`
	Path     string    `json:"path"`
	Title    string    `json:"title"`
	Sections []Section `json:"sections"`
}
type Config struct {
	SchemaVersion int    `json:"schemaVersion"`
	Name          string `json:"name"`
	Theme         string `json:"theme"`
	Pages         []Page `json:"pages"`
}

func DefaultSection(kind, id, variant string) (Section, error) {
	c, ok := registry.Find(kind)
	if !ok {
		return Section{}, fmt.Errorf("unknown template %q; run basestack templates", kind)
	}
	if variant == "" {
		variant = c.DefaultVariant
	}
	if !c.HasVariant(variant) {
		return Section{}, fmt.Errorf("unknown variant %q for %s; run basestack templates %s", variant, kind, kind)
	}
	return Section{ID: id, Type: kind, Variant: variant, Props: append(json.RawMessage(nil), c.DefaultProps...)}, nil
}
func New(name string) Config {
	sections := []Section{}
	for _, kind := range []string{"navbar", "hero", "footer"} {
		s, _ := DefaultSection(kind, kind, "")
		sections = append(sections, s)
	}
	return Config{SchemaVersion: 2, Name: name, Theme: "default", Pages: []Page{{ID: "home", Path: "/", Title: "Home", Sections: sections}}}
}
func Parse(data []byte) (Config, error) {
	var c Config
	// Recognize v1 explicitly without interpreting or modifying it.
	var header map[string]json.RawMessage
	if err := registry.Decode(data, &header); err != nil {
		return c, err
	}
	var version int
	if err := json.Unmarshal(header["schemaVersion"], &version); err != nil {
		return c, fmt.Errorf("schemaVersion must be an integer")
	}
	if version != 2 {
		return c, fmt.Errorf("unsupported schemaVersion %d; expected 2 (v1 projects require manual migration; see docs/CONFIGURATION.md)", version)
	}
	if err := registry.Decode(data, &c); err != nil {
		return c, err
	}
	return c, Validate(c)
}
func Read() (Config, error) {
	data, err := os.ReadFile(ConfigFile)
	if err != nil {
		return Config{}, fmt.Errorf("open basestack.json (run this inside a generated project): %w", err)
	}
	return Parse(data)
}
func (c *Config) SelectPage(id string) (*Page, error) {
	if id == "" {
		for i := range c.Pages {
			if c.Pages[i].ID == "home" {
				return &c.Pages[i], nil
			}
		}
		if len(c.Pages) == 1 {
			return &c.Pages[0], nil
		}
		return nil, fmt.Errorf("no home page; choose a page with --page <id>")
	}
	for i := range c.Pages {
		if c.Pages[i].ID == id {
			return &c.Pages[i], nil
		}
	}
	return nil, fmt.Errorf("page %q not found; run basestack page list", id)
}
