package registry

import (
	"testing"
)

func TestAllDefaults(t *testing.T) {
	for _, c := range Builtin.Components {
		var props any
		if err := Decode(c.DefaultProps, &props); err != nil {
			t.Fatal(err)
		}
		if err := c.Props.Validate(props, c.Type); err != nil {
			t.Fatal(err)
		}
		if !c.HasVariant(c.DefaultVariant) {
			t.Fatal("invalid default variant")
		}
	}
}
func TestLinks(t *testing.T) {
	for _, s := range []string{"/about", "/services/web", "#content", "https://example.com/a", "http://localhost:3000", "mailto:hello@example.com"} {
		if !SafeLink(s) {
			t.Fatalf("rejected %q", s)
		}
	}
	for _, s := range []string{"", "javascript:alert(1)", "data:text/html,hi", "//evil.test", "/\\evil.test", "https://", "mailto:bad", "/hello\nworld"} {
		if SafeLink(s) {
			t.Fatalf("accepted %q", s)
		}
	}
}
