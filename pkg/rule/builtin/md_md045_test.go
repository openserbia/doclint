package builtin_test

import (
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func noAltTextFindings(t *testing.T, raw []byte) []rule.Finding {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out []rule.Finding
	(builtin.NoAltText{}).Check(doc, func(f rule.Finding) { out = append(out, f) })
	return out
}

func TestNoAltText_Meta(t *testing.T) {
	m := (builtin.NoAltText{}).Meta()
	if m.Name != "no-alt-text" {
		t.Errorf("Name = %q, want no-alt-text", m.Name)
	}
	if m.Severity != rule.Warning {
		t.Errorf("Severity = %v, want Warning", m.Severity)
	}
	if m.Safety != rule.NoFix {
		t.Errorf("Safety = %v, want NoFix", m.Safety)
	}
	if !m.AppliesTo(document.Markdown) {
		t.Error("rule should apply to markdown")
	}
}

func TestNoAltText_FlagsEmptyAlt(t *testing.T) {
	raw := []byte("![](https://example.com/logo.png)\n")
	got := noAltTextFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	f := got[0]
	if f.Severity != rule.Warning || f.Line != 1 || f.Col != 1 {
		t.Errorf("finding = %+v, want Warning at line 1 col 1", f)
	}
	if f.Safety != rule.NoFix || len(f.Fixes) != 0 {
		t.Errorf("expected no fix, got safety=%v fixes=%d", f.Safety, len(f.Fixes))
	}
	if !strings.Contains(f.Message, "alt text") {
		t.Errorf("message = %q, want it to mention alt text", f.Message)
	}
}

func TestNoAltText_FlagsWhitespaceOnlyAlt(t *testing.T) {
	for _, raw := range []string{"![ ](u)\n", "![\t](u)\n", "![   ](u)\n"} {
		if got := noAltTextFindings(t, []byte(raw)); len(got) != 1 {
			t.Errorf("raw %q: got %d findings, want 1", raw, len(got))
		}
	}
}

func TestNoAltText_AcceptsNonEmptyAlt(t *testing.T) {
	raw := []byte("![company logo](https://example.com/logo.png)\n")
	if got := noAltTextFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestNoAltText_IgnoresInsideCodeSpan(t *testing.T) {
	// The image syntax is illustrative content inside an inline code span and
	// renders no image, so it must not be flagged.
	raw := []byte("Use `![](url)` to embed an image.\n")
	if got := noAltTextFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0 (inside code span)", len(got))
	}
}

func TestNoAltText_FlagsAfterClosedCodeSpan(t *testing.T) {
	// `code` is a closed span; the image that follows is real markup.
	raw := []byte("`code` ![](u)\n")
	got := noAltTextFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if got[0].Col != 8 { // '!' is byte index 7 → 1-based col 8
		t.Errorf("Col = %d, want 8 (the '!')", got[0].Col)
	}
}

func TestNoAltText_FlagsAfterUnclosedBacktick(t *testing.T) {
	// A lone backtick with no matching close is literal text, not a code span,
	// so the image after it still renders and must be flagged.
	raw := []byte("`![](u)\n")
	got := noAltTextFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1 (unclosed backtick is literal)", len(got))
	}
	if got[0].Col != 2 { // '!' is byte index 1 → col 2
		t.Errorf("Col = %d, want 2", got[0].Col)
	}
}

func TestNoAltText_IgnoresInsideFence(t *testing.T) {
	raw := []byte("```\n![](u)\n```\n")
	if got := noAltTextFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0 (inside fenced code block)", len(got))
	}
}

func TestNoAltText_ColumnAtBang(t *testing.T) {
	raw := []byte("See ![](u) here\n")
	got := noAltTextFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if got[0].Col != 5 { // '!' is byte index 4 → col 5
		t.Errorf("Col = %d, want 5", got[0].Col)
	}
}

func TestNoAltText_MultipleOnOneLine(t *testing.T) {
	raw := []byte("![](a) and ![](b)\n")
	got := noAltTextFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
	if got[0].Col != 1 || got[1].Col != 12 {
		t.Errorf("cols = %d,%d, want 1,12", got[0].Col, got[1].Col)
	}
}

func TestNoAltText_ReferenceImageNotFlagged(t *testing.T) {
	// Reference-style image (![][ref]) is not the inline ![](url) form this
	// rule targets, so it is left alone (no false positive).
	raw := []byte("![][logo]\n\n[logo]: https://example.com/logo.png\n")
	if got := noAltTextFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestNoAltText_MixedEmptyAndPresentOnLine(t *testing.T) {
	raw := []byte("![ok](a) and ![](b)\n")
	got := noAltTextFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1 (only the empty one)", len(got))
	}
	if got[0].Col != 14 { // second '!' byte index 13 → col 14
		t.Errorf("Col = %d, want 14", got[0].Col)
	}
}
