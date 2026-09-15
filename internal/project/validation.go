package project

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Janon-Emersion-T/Basestack/internal/registry"
)

var NamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,49}$`)
var pathPattern = regexp.MustCompile(`^/(?:[a-z0-9][a-z0-9_-]*(?:/[a-z0-9][a-z0-9_-]*)*)?$`)

func ValidPath(path string) bool { return pathPattern.MatchString(path) }
func Validate(c Config) error {
	if c.SchemaVersion != 2 {
		return fmt.Errorf("unsupported schemaVersion; expected 2")
	}
	if !NamePattern.MatchString(c.Name) {
		return fmt.Errorf("name must start with a lowercase letter and contain only lowercase letters, digits or hyphens (max 50 characters)")
	}
	if !registry.KnownTheme(c.Theme) {
		return fmt.Errorf("unknown theme %q; run basestack theme list", c.Theme)
	}
	if len(c.Pages) == 0 {
		return fmt.Errorf("pages must be a non-empty array")
	}
	ids, paths := map[string]bool{}, map[string]bool{}
	for _, p := range c.Pages {
		if !NamePattern.MatchString(p.ID) || ids[p.ID] {
			return fmt.Errorf("invalid or duplicate page id %q", p.ID)
		}
		ids[p.ID] = true
		if !ValidPath(p.Path) || paths[p.Path] {
			return fmt.Errorf("invalid or duplicate page path %q", p.Path)
		}
		paths[p.Path] = true
		if strings.TrimSpace(p.Title) == "" {
			return fmt.Errorf("page %q needs a title", p.ID)
		}
		if p.Sections == nil {
			return fmt.Errorf("page %q sections must be an array", p.ID)
		}
		seen := map[string]bool{"content": true}
		for _, s := range p.Sections {
			if !NamePattern.MatchString(s.ID) || seen[s.ID] {
				return fmt.Errorf("page %q: invalid, reserved or duplicate section id %q", p.ID, s.ID)
			}
			seen[s.ID] = true
			def, ok := registry.Find(s.Type)
			if !ok {
				return fmt.Errorf("unknown section type %q", s.Type)
			}
			if !def.HasVariant(s.Variant) {
				return fmt.Errorf("unknown variant %q for %s", s.Variant, s.Type)
			}
			var props any
			if err := registry.Decode(s.Props, &props); err != nil {
				return fmt.Errorf("section %q props: %w", s.ID, err)
			}
			if err := def.Props.Validate(props, "section "+s.ID+" props"); err != nil {
				return err
			}
		}
	}
	return nil
}
