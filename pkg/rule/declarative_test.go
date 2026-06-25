package rule

import (
	"testing"

	"github.com/openserbia/doclint/pkg/document"
)

func docWith(fm map[string]any, path string) *document.Document {
	return &document.Document{Path: path, Format: document.Markdown, Frontmatter: fm}
}

func runRule(t *testing.T, spec DeclSpec, doc *document.Document) []Finding {
	t.Helper()
	r, err := NewDeclarativeRule(spec)
	if err != nil {
		t.Fatalf("NewDeclarativeRule: %v", err)
	}
	var out []Finding
	r.Check(doc, func(f Finding) { out = append(out, f) })
	return out
}

func TestDeclarative_RequiredSkipsDraft(t *testing.T) {
	spec := DeclSpec{ID: "desc-req", Type: "required", Field: "description", SkipDrafts: true, Severity: Error}
	if got := runRule(t, spec, docWith(map[string]any{"draft": true}, "p.md")); len(got) != 0 {
		t.Fatalf("draft should be skipped, got %d findings", len(got))
	}
	if got := runRule(t, spec, docWith(map[string]any{}, "p.md")); len(got) != 1 {
		t.Fatalf("missing description should flag, got %d", len(got))
	}
}

func TestDeclarative_Length(t *testing.T) {
	spec := DeclSpec{ID: "len", Type: "length", Field: "description", Min: 5, Max: 10, Severity: Warning}
	if got := runRule(t, spec, docWith(map[string]any{"description": "hi"}, "p.md")); len(got) != 1 {
		t.Fatalf("too-short should flag, got %d", len(got))
	}
	if got := runRule(t, spec, docWith(map[string]any{"description": "just right"}, "p.md")); len(got) != 0 {
		t.Fatalf("in-range should pass, got %d", len(got))
	}
}

func TestDeclarative_NotEqual(t *testing.T) {
	spec := DeclSpec{ID: "ne", Type: "not_equal", Fields: []string{"description", "lead"}, Severity: Warning}
	doc := docWith(map[string]any{"description": "same", "lead": "same"}, "p.md")
	if got := runRule(t, spec, doc); len(got) != 1 {
		t.Fatalf("equal fields should flag, got %d", len(got))
	}
}

func TestDeclarative_GlobScope(t *testing.T) {
	spec := DeclSpec{ID: "g", Type: "required", Field: "description", Glob: "content/guides/**/*.md", Severity: Error}
	if got := runRule(t, spec, docWith(map[string]any{}, "content/blog/x.md")); len(got) != 0 {
		t.Fatalf("out-of-glob file should be skipped, got %d", len(got))
	}
	if got := runRule(t, spec, docWith(map[string]any{}, "content/guides/a/x.md")); len(got) != 1 {
		t.Fatalf("in-glob file should be checked, got %d", len(got))
	}
}
