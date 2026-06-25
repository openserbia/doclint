package builtin_test

import (
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func noTrailingSpacesFindings(t *testing.T, raw []byte) []rule.Finding {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out []rule.Finding
	(builtin.NoTrailingSpaces{}).Check(doc, func(f rule.Finding) { out = append(out, f) })
	return out
}

func TestNoTrailingSpaces_Meta(t *testing.T) {
	m := (builtin.NoTrailingSpaces{}).Meta()
	if m.Name != "no-trailing-spaces" {
		t.Errorf("Name = %q, want no-trailing-spaces", m.Name)
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

func TestNoTrailingSpaces_FlagsSingleStraySpace(t *testing.T) {
	raw := []byte("hello \n") // exactly one trailing space
	got := noTrailingSpacesFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	f := got[0]
	if f.Severity != rule.Warning || f.Line != 1 {
		t.Errorf("finding = %+v, want Warning at line 1", f)
	}
	if f.Safety != rule.NoFix || len(f.Fixes) != 0 {
		t.Errorf("expected no fix, got safety=%v fixes=%d", f.Safety, len(f.Fixes))
	}
	if f.Col != 6 { // "hello" is 5 bytes; the stray space is at col 6
		t.Errorf("Col = %d, want 6 (first trailing space)", f.Col)
	}
	if !strings.Contains(f.Message, "trailing") {
		t.Errorf("message = %q, want it to mention trailing", f.Message)
	}
}

func TestNoTrailingSpaces_AllowsTwoSpaceHardBreak(t *testing.T) {
	// Exactly two trailing spaces is an intentional markdown hard line break and
	// must never be flagged — this is the whole reason the rule is NoFix.
	raw := []byte("hello  \nworld\n")
	if got := noTrailingSpacesFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0 (two-space hard break is intentional)", len(got))
	}
}

func TestNoTrailingSpaces_FlagsThreeSpaces(t *testing.T) {
	// Three+ trailing spaces collapse to a 2-space hard break — likely unintended.
	raw := []byte("hello   \n")
	got := noTrailingSpacesFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if got[0].Col != 6 {
		t.Errorf("Col = %d, want 6", got[0].Col)
	}
}

func TestNoTrailingSpaces_FlagsManySpaces(t *testing.T) {
	raw := []byte("hello     \n") // five trailing spaces
	if got := noTrailingSpacesFindings(t, raw); len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
}

func TestNoTrailingSpaces_DistinctMessagesByCount(t *testing.T) {
	single := noTrailingSpacesFindings(t, []byte("a \n"))
	many := noTrailingSpacesFindings(t, []byte("a    \n"))
	if len(single) != 1 || len(many) != 1 {
		t.Fatalf("got %d single / %d many, want 1/1", len(single), len(many))
	}
	if single[0].Message == many[0].Message {
		t.Errorf("messages should be count-specific, both = %q", single[0].Message)
	}
}

func TestNoTrailingSpaces_FlagsWhitespaceOnlyLine(t *testing.T) {
	// A whitespace-only line has no preceding content, so trailing spaces cannot
	// be a hard break — flag it regardless of the count.
	raw := []byte("text\n   \nmore\n") // middle line is three spaces only
	got := noTrailingSpacesFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if got[0].Line != 2 {
		t.Errorf("Line = %d, want 2", got[0].Line)
	}
}

func TestNoTrailingSpaces_FlagsWhitespaceOnlyTwoSpaces(t *testing.T) {
	// Even exactly two spaces on an otherwise-empty line is flagged: the hard-break
	// exception only applies when there is content the break would attach to.
	raw := []byte("text\n  \nmore\n")
	if got := noTrailingSpacesFindings(t, raw); len(got) != 1 {
		t.Fatalf("got %d findings, want 1 (whitespace-only overrides the 2-space rule)", len(got))
	}
}

func TestNoTrailingSpaces_AcceptsNoTrailingSpace(t *testing.T) {
	raw := []byte("clean line\nanother\n")
	if got := noTrailingSpacesFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestNoTrailingSpaces_IgnoresInsideFence(t *testing.T) {
	// Trailing spaces inside a fenced code block are significant content and must
	// not be touched.
	raw := []byte("```\ncode   \n```\n")
	if got := noTrailingSpacesFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0 (inside fenced code block)", len(got))
	}
}

func TestNoTrailingSpaces_TrailingTabNotFlagged(t *testing.T) {
	// MD009 targets trailing SPACES; a trailing tab is not a space run and is left
	// alone (TrimRight(\" \") removes only spaces).
	raw := []byte("hello\t\n")
	if got := noTrailingSpacesFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0 (trailing tab is not a trailing space)", len(got))
	}
}

func TestNoTrailingSpaces_MultipleLines(t *testing.T) {
	raw := []byte("a \nb  \nc   \nd\n") // line1: 1 space (flag), line2: 2 (ok), line3: 3 (flag), line4: none
	got := noTrailingSpacesFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
	if got[0].Line != 1 || got[1].Line != 3 {
		t.Errorf("lines = %d,%d, want 1,3", got[0].Line, got[1].Line)
	}
}
