package builtin_test

import (
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func listMarkerIndentFindings(t *testing.T, raw []byte) []rule.Finding {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out []rule.Finding
	(builtin.ListMarkerIndent{}).Check(doc, func(f rule.Finding) { out = append(out, f) })
	return out
}

func TestListMarkerIndent_Meta(t *testing.T) {
	m := (builtin.ListMarkerIndent{}).Meta()
	if m.Name != "list-marker-indent" {
		t.Errorf("Name = %q, want list-marker-indent", m.Name)
	}
	if m.Severity != rule.Warning {
		t.Errorf("Severity = %v, want Warning", m.Severity)
	}
	if m.Safety != rule.Unsafe {
		t.Errorf("Safety = %v, want Unsafe", m.Safety)
	}
}

// The leading paragraph already sits at the content column (3); only the bullets
// (2) are under-indented. The fix must lift the bullets to 3 WITHOUT pushing the
// already-correct paragraph to 4 — the v0.5.0 uniform-shift bug.
func TestListMarkerIndent_PreservesLeadingParagraph(t *testing.T) {
	raw := []byte("2. {{< details \"x\" >}}\n   para at 3\n  - bullet at 2\n  {{< /details >}}\n3. next\n")
	got := listMarkerIndentFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	want := "2. {{< details \"x\" >}}\n   para at 3\n   - bullet at 2\n   {{< /details >}}\n3. next\n"
	if fixed := string(applyAll(t, raw, got)); fixed != want {
		t.Errorf("fixed = %q\n want %q", fixed, want)
	}
}

// A nested sub-list two columns deeper must keep its relative depth: shifting the
// first-level bullets 2->3 shifts the sub-bullets 4->5, not flattens them.
func TestListMarkerIndent_PreservesNesting(t *testing.T) {
	raw := []byte("1. {{< details \"x\" >}}\n  - bullet at 2\n    - sub at 4\n  {{< /details >}}\n1. next\n")
	got := listMarkerIndentFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	want := "1. {{< details \"x\" >}}\n   - bullet at 2\n     - sub at 4\n   {{< /details >}}\n1. next\n"
	if fixed := string(applyAll(t, raw, got)); fixed != want {
		t.Errorf("fixed = %q\n want %q", fixed, want)
	}
}

// The bullets are already correct (3); only the closing shortcode de-indented out
// of the list (2). The fix lifts just that trailing line, leaving the bullets.
func TestListMarkerIndent_FixesTrailingCloser(t *testing.T) {
	raw := []byte("8. {{< details \"x\" >}}\n   - a\n   - b\n  {{< /details >}}\n9. next\n")
	got := listMarkerIndentFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	want := "8. {{< details \"x\" >}}\n   - a\n   - b\n   {{< /details >}}\n9. next\n"
	if fixed := string(applyAll(t, raw, got)); fixed != want {
		t.Errorf("fixed = %q\n want %q", fixed, want)
	}
}

func TestListMarkerIndent_IgnoresWellIndented(t *testing.T) {
	raw := []byte("1. {{< details \"x\" >}}\n   - a\n   {{< /details >}}\n1. next\n")
	if got := listMarkerIndentFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestListMarkerIndent_FixIsIdempotent(t *testing.T) {
	raw := []byte("2. {{< details \"x\" >}}\n   para\n  - a\n  {{< /details >}}\n3. next\n")
	got := listMarkerIndentFindings(t, raw)
	fixed := applyAll(t, raw, got)
	if again := listMarkerIndentFindings(t, fixed); len(again) != 0 {
		t.Fatalf("fixed text still flagged: %d findings", len(again))
	}
}
