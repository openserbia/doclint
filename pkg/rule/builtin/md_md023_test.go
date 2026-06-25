package builtin_test

import (
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/engine"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func headingStartFindings(t *testing.T, raw []byte) []rule.Finding {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out []rule.Finding
	(builtin.HeadingStartLeft{}).Check(doc, func(f rule.Finding) { out = append(out, f) })
	return out
}

func TestHeadingStartLeft_Meta(t *testing.T) {
	m := (builtin.HeadingStartLeft{}).Meta()
	if m.Name != "heading-start-left" {
		t.Errorf("Name = %q, want heading-start-left", m.Name)
	}
	if m.Severity != rule.Warning {
		t.Errorf("Severity = %v, want Warning", m.Severity)
	}
	if m.Safety != rule.Safe {
		t.Errorf("Safety = %v, want Safe", m.Safety)
	}
	if !m.AppliesTo(document.Markdown) {
		t.Error("rule should apply to markdown")
	}
}

func TestHeadingStartLeft_FlagsIndentedHeading(t *testing.T) {
	got := headingStartFindings(t, []byte("  # Heading\n"))
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	f := got[0]
	if f.Severity != rule.Warning || f.Line != 1 || f.Col != 1 {
		t.Errorf("finding = %+v, want Warning at line 1 col 1", f)
	}
	if f.Safety != rule.Safe || len(f.Fixes) != 1 {
		t.Fatalf("expected one safe fix, got safety=%v fixes=%d", f.Safety, len(f.Fixes))
	}
}

func TestHeadingStartLeft_FixDedents(t *testing.T) {
	raw := []byte("   ## Heading\n")
	got := headingStartFindings(t, raw)
	fixed, err := engine.ApplyEdits(raw, got[0].Fixes)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if want := "## Heading\n"; string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
}

func TestHeadingStartLeft_FixIsIdempotent(t *testing.T) {
	raw := []byte("  ### Heading\n")
	got := headingStartFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	fixed, err := engine.ApplyEdits(raw, got[0].Fixes)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if want := "### Heading\n"; string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
	if again := headingStartFindings(t, fixed); len(again) != 0 {
		t.Fatalf("fixed text still flagged: %d findings", len(again))
	}
}

func TestHeadingStartLeft_EscalatesAtFourSpaces(t *testing.T) {
	got := headingStartFindings(t, []byte("    # Lost\n"))
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if !strings.Contains(got[0].Message, "code block") {
		t.Errorf("message = %q, want it to mention the heading becomes a code block", got[0].Message)
	}
	// Still top-level, so a dedent fix is attached.
	if got[0].Safety != rule.Safe || len(got[0].Fixes) != 1 {
		t.Fatalf("expected a safe fix at >=4 indent, got safety=%v fixes=%d", got[0].Safety, len(got[0].Fixes))
	}
}

func TestHeadingStartLeft_TabIndentDedents(t *testing.T) {
	raw := []byte("\t# Tabbed\n")
	got := headingStartFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	fixed, err := engine.ApplyEdits(raw, got[0].Fixes)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if want := "# Tabbed\n"; string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
}

func TestHeadingStartLeft_AcceptsLeftAligned(t *testing.T) {
	cases := []string{
		"# Heading\n",
		"## Sub\n",
		"###### Deep\n",
		"#\n",         // bare hash, left aligned
		"# Title #\n", // closed ATX
	}
	for _, c := range cases {
		if got := headingStartFindings(t, []byte(c)); len(got) != 0 {
			t.Errorf("%q: got %d findings, want 0", c, len(got))
		}
	}
}

func TestHeadingStartLeft_NonHeadingShapes(t *testing.T) {
	cases := []string{
		"  #Heading\n",        // no space after #: not an ATX heading (MD018 territory)
		"  ####### seven\n",   // seven hashes is not a heading
		"  text # mid line\n", // hash not at the indented start
		"  plain paragraph\n", // indented prose, no hash
	}
	for _, c := range cases {
		if got := headingStartFindings(t, []byte(c)); len(got) != 0 {
			t.Errorf("%q: got %d findings, want 0", c, len(got))
		}
	}
}

func TestHeadingStartLeft_IgnoresFence(t *testing.T) {
	raw := []byte("```\n  # Heading\n```\n")
	if got := headingStartFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings inside fence, want 0", len(got))
	}
}

func TestHeadingStartLeft_IgnoresFrontmatter(t *testing.T) {
	raw := []byte("---\ntags:\n  - one\n  - two\n---\n\n  # Body\n")
	got := headingStartFindings(t, raw)
	// The frontmatter list items must not be treated as an enclosing list for
	// the body heading: the body heading is top-level and gets a fix.
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if got[0].Line != 7 {
		t.Errorf("Line = %d, want 7", got[0].Line)
	}
	if got[0].Safety != rule.Safe || len(got[0].Fixes) != 1 {
		t.Fatalf("body heading should get a fix; safety=%v fixes=%d", got[0].Safety, len(got[0].Fixes))
	}
}

func TestHeadingStartLeft_GuardNestedUnorderedList(t *testing.T) {
	raw := []byte("- item\n  # nested heading\n")
	got := headingStartFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if got[0].Safety != rule.NoFix || len(got[0].Fixes) != 0 {
		t.Fatalf("nested heading must NOT get a fix; safety=%v fixes=%d", got[0].Safety, len(got[0].Fixes))
	}
}

func TestHeadingStartLeft_GuardNestedOrderedList(t *testing.T) {
	raw := []byte("1. item\n   # nested heading\n")
	got := headingStartFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if got[0].Safety != rule.NoFix || len(got[0].Fixes) != 0 {
		t.Fatalf("nested heading must NOT get a fix; safety=%v fixes=%d", got[0].Safety, len(got[0].Fixes))
	}
}

func TestHeadingStartLeft_GuardDeepNestedList(t *testing.T) {
	raw := []byte("- outer\n  - inner\n    # deep heading\n")
	got := headingStartFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if got[0].Safety != rule.NoFix || len(got[0].Fixes) != 0 {
		t.Fatalf("deeply nested heading must NOT get a fix; safety=%v fixes=%d", got[0].Safety, len(got[0].Fixes))
	}
}

func TestHeadingStartLeft_TopLevelAfterListGetsFix(t *testing.T) {
	// A root-level paragraph closes the list, so the indented heading below is
	// top-level and the dedent fix is safe.
	raw := []byte("- item\n\nback at root\n\n  # heading\n")
	got := headingStartFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if got[0].Safety != rule.Safe || len(got[0].Fixes) != 1 {
		t.Fatalf("top-level heading should get a fix; safety=%v fixes=%d", got[0].Safety, len(got[0].Fixes))
	}
}
