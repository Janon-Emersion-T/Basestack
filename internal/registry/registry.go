// Package registry defines the portable component property and theme contract.
package registry

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/Janon-Emersion-T/Basestack/internal/strictjson"
)

//go:embed builtin.json
var BuiltinJSON []byte

type Rule struct {
	Type     string          `json:"type"`
	Required bool            `json:"required,omitempty"`
	NonEmpty bool            `json:"nonEmpty,omitempty"`
	Format   string          `json:"format,omitempty"`
	Fields   map[string]Rule `json:"fields,omitempty"`
	Items    *Rule           `json:"items,omitempty"`
}
type Component struct {
	Type           string          `json:"type"`
	Variants       []string        `json:"variants"`
	DefaultVariant string          `json:"defaultVariant"`
	Props          Rule            `json:"props"`
	DefaultProps   json.RawMessage `json:"defaultProps"`
}
type Theme struct {
	Name   string            `json:"name"`
	Tokens map[string]string `json:"tokens"`
}
type Catalog struct {
	Components []Component `json:"components"`
	Themes     []Theme     `json:"themes"`
}

var Builtin = func() Catalog {
	var c Catalog
	if err := Decode(BuiltinJSON, &c); err != nil {
		panic(err)
	}
	return c
}()

func Find(kind string) (Component, bool) {
	for _, c := range Builtin.Components {
		if c.Type == kind {
			return c, true
		}
	}
	return Component{}, false
}
func KnownTheme(name string) bool {
	for _, t := range Builtin.Themes {
		if t.Name == name {
			return true
		}
	}
	return false
}
func (c Component) HasVariant(variant string) bool {
	for _, v := range c.Variants {
		if v == variant {
			return true
		}
	}
	return false
}

// Decode preserves the existing registry API while sharing strict parsing with services.
func Decode(data []byte, dst any) error { return strictjson.Decode(data, dst) }

var emailPattern = regexp.MustCompile(`^[A-Za-z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+$`)

func SafeLink(s string) bool {
	if s == "" || strings.Contains(s, "\\") || strings.ContainsFunc(s, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) || r == '\ufeff' }) {
		return false
	}
	if strings.HasPrefix(s, "#") {
		return len(s) > 1
	}
	if strings.HasPrefix(s, "/") {
		return !strings.HasPrefix(s, "//")
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return ((u.Scheme == "https" || u.Scheme == "http") && u.Host != "") || (strings.HasPrefix(s, "mailto:") && emailPattern.MatchString(strings.TrimPrefix(s, "mailto:")))
}
func (r Rule) Validate(value any, path string) error {
	bad := func() error { return fmt.Errorf("%s must be a valid %s", path, r.Type) }
	switch r.Type {
	case "string":
		s, ok := value.(string)
		if !ok {
			return bad()
		}
		if r.NonEmpty && strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s must not be empty", path)
		}
		if r.Format == "link" && !SafeLink(s) {
			return fmt.Errorf("%s must be a safe relative, http(s), anchor or mailto link", path)
		}
		if r.Format == "email" && !emailPattern.MatchString(s) {
			return fmt.Errorf("%s must be an email address", path)
		}
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			return bad()
		}
		for k := range obj {
			if _, ok := r.Fields[k]; !ok {
				return fmt.Errorf("unknown field %s.%s", path, k)
			}
		}
		for k, rule := range r.Fields {
			v, exists := obj[k]
			if !exists {
				if rule.Required {
					return fmt.Errorf("%s.%s is required", path, k)
				}
				continue
			}
			if err := rule.Validate(v, path+"."+k); err != nil {
				return err
			}
		}
	case "array":
		a, ok := value.([]any)
		if !ok {
			return bad()
		}
		for i, v := range a {
			if err := r.Items.Validate(v, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported registry rule %q", r.Type)
	}
	return nil
}
