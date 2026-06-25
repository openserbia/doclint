package rule

import (
	"testing"

	"github.com/openserbia/doclint/pkg/document"
)

func TestSeverity_RoundTrip(t *testing.T) {
	for _, s := range []Severity{Info, Warning, Error} {
		parsed, err := ParseSeverity(s.String())
		if err != nil {
			t.Fatalf("ParseSeverity(%q): %v", s.String(), err)
		}
		if parsed != s {
			t.Errorf("round-trip %v -> %q -> %v", s, s.String(), parsed)
		}
	}
	if _, err := ParseSeverity("bogus"); err == nil {
		t.Error("expected error for bogus severity")
	}
}

type fakeRule struct{ name string }

func (f fakeRule) Meta() Meta {
	return Meta{Name: f.name, Formats: []document.Format{document.Markdown}}
}
func (f fakeRule) Check(_ *document.Document, _ func(Finding)) {}

func TestRegistry_Order(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeRule{"b"})
	r.Register(fakeRule{"a"})
	got := []string{r.All()[0].Meta().Name, r.All()[1].Meta().Name}
	if got[0] != "b" || got[1] != "a" {
		t.Errorf("order = %v, want [b a]", got)
	}
}
