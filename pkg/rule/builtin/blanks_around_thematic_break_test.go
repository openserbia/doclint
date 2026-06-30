package builtin_test

import (
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/engine"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func blanksAroundThematicBreakFindings(t *testing.T, raw []byte) []rule.Finding {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out []rule.Finding
	(builtin.BlanksAroundThematicBreak{}).Check(doc, func(f rule.Finding) { out = append(out, f) })
	return out
}

func TestBlanksAroundThematicBreak_Meta(t *testing.T) {
	m := (builtin.BlanksAroundThematicBreak{}).Meta()
	if m.Name != "blanks-around-thematic-break" {
		t.Errorf("Name = %q, want blanks-around-thematic-break", m.Name)
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

func TestBlanksAroundThematicBreak_FlagsMissingBlankBefore(t *testing.T) {
	// Use "***" — unambiguously a thematic break (never a setext underline).
	raw := []byte("before\n***\n")
	got := blanksAroundThematicBreakFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	f := got[0]
	if f.Severity != rule.Warning || f.Line != 2 {
		t.Errorf("finding = %+v, want Warning at line 2", f)
	}
	if !strings.Contains(f.Message, "before") {
		t.Errorf("message = %q, want it to mention before", f.Message)
	}
	if f.Safety != rule.Safe || len(f.Fixes) != 1 {
		t.Fatalf("expected one safe fix, got safety=%v fixes=%d", f.Safety, len(f.Fixes))
	}
	fixed, err := engine.ApplyEdits(raw, f.Fixes)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if want := "before\n\n***\n"; string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
}

// TestBlanksAroundThematicBreak_FlagsMissingBlankBeforeDash verifies that a
// "---" preceded by a structural boundary (ATX heading) — not paragraph text —
// is recognised as a thematic break and flagged for the missing blank.
func TestBlanksAroundThematicBreak_FlagsMissingBlankBeforeDash(t *testing.T) {
	raw := []byte("# heading\n---\n")
	got := blanksAroundThematicBreakFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	f := got[0]
	if f.Line != 2 || !strings.Contains(f.Message, "before") {
		t.Errorf("finding = %+v, want Warning at line 2 mentioning before", f)
	}
	fixed, err := engine.ApplyEdits(raw, f.Fixes)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if want := "# heading\n\n---\n"; string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
}

func TestBlanksAroundThematicBreak_FlagsMissingBlankAfter(t *testing.T) {
	raw := []byte("---\nafter\n")
	got := blanksAroundThematicBreakFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	f := got[0]
	if f.Line != 1 {
		t.Errorf("Line = %d, want 1", f.Line)
	}
	if !strings.Contains(f.Message, "after") {
		t.Errorf("message = %q, want it to mention after", f.Message)
	}
	fixed, err := engine.ApplyEdits(raw, f.Fixes)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if want := "---\n\nafter\n"; string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
}

func TestBlanksAroundThematicBreak_FlagsBothSides(t *testing.T) {
	// "***" is unambiguously a thematic break; plain text before it is not
	// paragraph setext text for "***" (which can never be a setext underline).
	raw := []byte("before\n***\nafter\n")
	got := blanksAroundThematicBreakFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
	fixed, err := engine.ApplyEdits(raw, append(got[0].Fixes, got[1].Fixes...))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if want := "before\n\n***\n\nafter\n"; string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
}

func TestBlanksAroundThematicBreak_ExemptsDocumentStartAndEnd(t *testing.T) {
	// Thematic break is the only line: first and last, no blank needed on either side.
	if got := blanksAroundThematicBreakFindings(t, []byte("***\n")); len(got) != 0 {
		t.Fatalf("got %d findings, want 0 (first+last line exempt)", len(got))
	}
	// Break at start (first line) with content after it — "after" side is flagged, start is exempt.
	got := blanksAroundThematicBreakFindings(t, []byte("***\nafter\n"))
	if len(got) != 1 || !strings.Contains(got[0].Message, "after") {
		t.Errorf("got %v, want exactly 1 after-finding (start exempt)", got)
	}
	// Break at end (last line) with content before it via structural boundary —
	// use an ATX heading so "---" is not misread as a setext underline.
	got = blanksAroundThematicBreakFindings(t, []byte("# heading\n---\n"))
	if len(got) != 1 || !strings.Contains(got[0].Message, "before") {
		t.Errorf("got %v, want exactly 1 before-finding (end exempt)", got)
	}
}

func TestBlanksAroundThematicBreak_AcceptsWellSpaced(t *testing.T) {
	raw := []byte("para\n\n---\n\npara\n")
	if got := blanksAroundThematicBreakFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

// TestBlanksAroundThematicBreak_SkipsSetextHeading verifies that a "---" line
// directly following plain paragraph text is treated as a setext heading underline
// and not flagged by this rule (BlanksAroundHeadings handles it instead).
func TestBlanksAroundThematicBreak_SkipsSetextHeading(t *testing.T) {
	// "text\n---\n" — the "---" is the setext underline for the heading "text".
	raw := []byte("text\n---\nafter\n")
	if got := blanksAroundThematicBreakFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0 (setext heading, not thematic break)", len(got))
	}
}

// TestBlanksAroundThematicBreak_HugoShortcodeBeforeBreak verifies that a "---"
// following a Hugo shortcode is treated as a thematic break (not a setext
// underline), because shortcodes are block-level elements in Goldmark/Hugo.
func TestBlanksAroundThematicBreak_HugoShortcodeBeforeBreak(t *testing.T) {
	raw := []byte("{{< figure src=\"x.jpg\" >}}\n---\nafter\n")
	got := blanksAroundThematicBreakFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2 (before + after)", len(got))
	}
	if !strings.Contains(got[0].Message, "before") {
		t.Errorf("first finding message = %q, want it to mention before", got[0].Message)
	}
	if !strings.Contains(got[1].Message, "after") {
		t.Errorf("second finding message = %q, want it to mention after", got[1].Message)
	}
}

func TestBlanksAroundThematicBreak_AsteriskBreak(t *testing.T) {
	raw := []byte("before\n***\nafter\n")
	got := blanksAroundThematicBreakFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings for ***, want 2", len(got))
	}
}

func TestBlanksAroundThematicBreak_UnderscoreBreak(t *testing.T) {
	raw := []byte("before\n___\nafter\n")
	got := blanksAroundThematicBreakFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings for ___, want 2", len(got))
	}
}

func TestBlanksAroundThematicBreak_SpacedVariant(t *testing.T) {
	// "- - -" is a thematic break (thematicBreakRe) but does NOT match underlineRe,
	// so the setext-heading exemption is never applied.
	raw := []byte("before\n- - -\nafter\n")
	got := blanksAroundThematicBreakFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings for '- - -', want 2", len(got))
	}
}

func TestBlanksAroundThematicBreak_SkipsFencedContent(t *testing.T) {
	// A "---" inside a fenced code block is not a thematic break.
	raw := []byte("```\nbefore\n---\nafter\n```\n")
	if got := blanksAroundThematicBreakFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings inside fence, want 0", len(got))
	}
}

func TestBlanksAroundThematicBreak_SkipsFrontmatter(t *testing.T) {
	// The "---" delimiters of YAML frontmatter must not be flagged.
	raw := []byte("---\ntitle: foo\n---\n\nbody\n")
	if got := blanksAroundThematicBreakFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings for frontmatter, want 0", len(got))
	}
}

func TestBlanksAroundThematicBreak_FixIsIdempotent(t *testing.T) {
	raw := []byte("before\n***\nafter\n")
	got := blanksAroundThematicBreakFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2 to apply", len(got))
	}
	fixed, err := engine.ApplyEdits(raw, append(got[0].Fixes, got[1].Fixes...))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if again := blanksAroundThematicBreakFindings(t, fixed); len(again) != 0 {
		t.Fatalf("fixed text still flagged: %d findings", len(again))
	}
}
