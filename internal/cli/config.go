package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var variants = map[string]string{
	"navbar": "minimal",
	"hero": "split",
	"features": "list",
	"pricing": "compact",
	"contact": "centered",
	"footer": "minimal",
}

type Section struct {
	ID string `json:"id"`
	Type string `json:"type"`
	Title string `json:"title"`
	Text string `json:"text"`
	Variant string `json:"variant,omitempty"`
}

func (s *Section) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID *string `json:"id"`
		Type *string `json:"type"`
		Title *string `json:"title"`
		Text *string `json:"text"`
		Variant json.RawMessage `json:"variant"`
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&raw); err != nil { return err }
	if raw.ID == nil || raw.Type == nil || raw.Title == nil || raw.Text == nil { return errors.New("each section requires string id, type, title and text fields") }
	variant := ""
	if raw.Variant != nil {
		if err := json.Unmarshal(raw.Variant, &variant); err != nil || variant == "" { return errors.New("variant must be a non-empty string when provided") }
	}
	*s = Section{ID:*raw.ID, Type:*raw.Type, Title:*raw.Title, Text:*raw.Text, Variant:variant}
	return nil
}

type Page struct {
	Slug string `json:"slug"`
	Title string `json:"title"`
	Sections []Section `json:"sections"`
}

type Config struct {
	SchemaVersion int `json:"schemaVersion"`
	Name string `json:"name"`
	Sections []Section `json:"sections"`
	Pages []Page `json:"pages,omitempty"`
}

// Keep schema 1 compatible with projects created by the first milestone.
// A missing pages field is valid; explicitly supplying null is not.
func (c *Config) UnmarshalJSON(data []byte) error {
	type plain Config
	var raw plain
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&raw); err != nil { return err }
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil { return err }
	if pages, ok := fields["pages"]; ok && bytes.Equal(bytes.TrimSpace(pages), []byte("null")) { return errors.New("pages must be an array when provided") }
	*c = Config(raw)
	return nil
}

func validVariant(kind, variant string) bool {
	return known(kind) && (variant == "" || variant == "default" || variant == variants[kind])
}

func validateSections(sections []Section) error {
	if sections == nil { return errors.New("sections must be an array") }
	seen := map[string]bool{}
	for _, s := range sections {
		if !known(s.Type) { return fmt.Errorf("unknown section type %q", s.Type) }
		if !namePattern.MatchString(s.ID) || seen[s.ID] { return fmt.Errorf("invalid or duplicate section id %q", s.ID) }
		seen[s.ID] = true
		if strings.TrimSpace(s.Title) == "" { return fmt.Errorf("section %q needs a title", s.ID) }
		if !validVariant(s.Type, s.Variant) { return fmt.Errorf("section %q: unsupported %s variant %q", s.ID, s.Type, s.Variant) }
	}
	return nil
}

func validate(c Config) error {
	if c.SchemaVersion != 1 { return errors.New("unsupported schemaVersion; expected 1") }
	if !namePattern.MatchString(c.Name) { return errors.New("name must start with a lowercase letter and contain only lowercase letters, digits or hyphens (max 50 characters)") }
	if err := validateSections(c.Sections); err != nil { return fmt.Errorf("home: %w", err) }
	seen := map[string]bool{"home": true}
	for _, page := range c.Pages {
		if !namePattern.MatchString(page.Slug) || seen[page.Slug] { return fmt.Errorf("invalid, reserved or duplicate page slug %q", page.Slug) }
		seen[page.Slug] = true
		if strings.TrimSpace(page.Title) == "" { return fmt.Errorf("page %q needs a title", page.Slug) }
		if err := validateSections(page.Sections); err != nil { return fmt.Errorf("page %q: %w", page.Slug, err) }
	}
	return nil
}

func pageSections(c *Config, slug string) (*[]Section, error) {
	if slug == "home" { return &c.Sections, nil }
	for i := range c.Pages { if c.Pages[i].Slug == slug { return &c.Pages[i].Sections, nil } }
	return nil, fmt.Errorf("page %q not found; run basestack page list", slug)
}
