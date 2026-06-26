package builtin_test

import (
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func blanksAroundHeadingsFindings(t *testing.T, raw []byte) []rule.Finding {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out []rule.Finding
	(builtin.BlanksAroundHeadings{}).Check(doc, func(f rule.Finding) { out = append(out, f) })
	return out
}

func TestBlanksAroundHeadings_Meta(t *testing.T) {
	m := (builtin.BlanksAroundHeadings{}).Meta()
	if m.Name != "blanks-around-headings" {
		t.Errorf("Name = %q, want blanks-around-headings", m.Name)
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

func TestBlanksAroundHeadings_ATXMissingBlankAbove(t *testing.T) {
	raw := []byte("para\n# Heading\n")
	got := blanksAroundHeadingsFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	f := got[0]
	if f.Severity != rule.Warning || f.Line != 2 {
		t.Errorf("finding = %+v, want Warning at line 2", f)
	}
	if !strings.Contains(f.Message, "above") {
		t.Errorf("message = %q, want it to mention above", f.Message)
	}
	if f.Safety != rule.Safe || len(f.Fixes) != 1 {
		t.Fatalf("expected one safe fix, got safety=%v fixes=%d", f.Safety, len(f.Fixes))
	}
	if want := "para\n\n# Heading\n"; string(applyAll(t, raw, got)) != want {
		t.Errorf("fixed = %q, want %q", string(applyAll(t, raw, got)), want)
	}
}

func TestBlanksAroundHeadings_ATXMissingBlankBelow(t *testing.T) {
	raw := []byte("# Heading\npara\n")
	got := blanksAroundHeadingsFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	f := got[0]
	if f.Line != 1 {
		t.Errorf("Line = %d, want 1 (heading line)", f.Line)
	}
	if !strings.Contains(f.Message, "below") {
		t.Errorf("message = %q, want it to mention below", f.Message)
	}
	if want := "# Heading\n\npara\n"; string(applyAll(t, raw, got)) != want {
		t.Errorf("fixed = %q, want %q", string(applyAll(t, raw, got)), want)
	}
}

func TestBlanksAroundHeadings_ATXBothSides(t *testing.T) {
	raw := []byte("before\n## Heading\nafter\n")
	got := blanksAroundHeadingsFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
	if want := "before\n\n## Heading\n\nafter\n"; string(applyAll(t, raw, got)) != want {
		t.Errorf("fixed = %q, want %q", string(applyAll(t, raw, got)), want)
	}
}

func TestBlanksAroundHeadings_AcceptsWellSpaced(t *testing.T) {
	raw := []byte("para\n\n# Heading\n\npara\n")
	if got := blanksAroundHeadingsFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestBlanksAroundHeadings_ExemptsDocumentStartAndEnd(t *testing.T) {
	// The heading is both the first and the last line: both edges are document
	// boundaries, so nothing is flagged.
	if got := blanksAroundHeadingsFindings(t, []byte("# Heading\n")); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestBlanksAroundHeadings_ExemptsFrontmatterBoundary(t *testing.T) {
	// A heading directly after the frontmatter close is not flagged (the YAML
	// "---" is frontmatter, never a setext underline or thematic break), and the
	// blank below it keeps the document clean.
	raw := []byte("---\ntitle: x\n---\n# Heading\n\nbody\n")
	if got := blanksAroundHeadingsFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestBlanksAroundHeadings_IgnoresHashInsideFence(t *testing.T) {
	// "# H" lives inside a fenced code block (InFence), so it is not a heading and
	// nothing is flagged.
	raw := []byte("text\n\n```\n# H\n```\n\ntext\n")
	if got := blanksAroundHeadingsFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestBlanksAroundHeadings_SetextMissingBlankBelow(t *testing.T) {
	raw := []byte("Title\n=====\ntext\n")
	got := blanksAroundHeadingsFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if !strings.Contains(got[0].Message, "below") {
		t.Errorf("message = %q, want it to mention below", got[0].Message)
	}
	if want := "Title\n=====\n\ntext\n"; string(applyAll(t, raw, got)) != want {
		t.Errorf("fixed = %q, want %q", string(applyAll(t, raw, got)), want)
	}
}

func TestBlanksAroundHeadings_SetextDashMissingBlankBelow(t *testing.T) {
	raw := []byte("Title\n-----\ntext\n")
	got := blanksAroundHeadingsFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if want := "Title\n-----\n\ntext\n"; string(applyAll(t, raw, got)) != want {
		t.Errorf("fixed = %q, want %q", string(applyAll(t, raw, got)), want)
	}
}

func TestBlanksAroundHeadings_SetextWellSpaced(t *testing.T) {
	raw := []byte("intro\n\nTitle\n=====\n\nafter\n")
	if got := blanksAroundHeadingsFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestBlanksAroundHeadings_ThematicBreakIsNotSetext(t *testing.T) {
	// "---" preceded and followed by a blank is a thematic break, not a setext
	// underline (its text line would be blank), so nothing is flagged.
	raw := []byte("para\n\n---\n\nmore\n")
	if got := blanksAroundHeadingsFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestBlanksAroundHeadings_MultiLineSetextNotSplit(t *testing.T) {
	// The setext text spans two lines; the rule must NOT insert a blank between
	// them (that would split the heading paragraph and change its text). Only the
	// missing blank BELOW the underline is flagged.
	raw := []byte("line one\nline two\n=====\ntext\n")
	got := blanksAroundHeadingsFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if want := "line one\nline two\n=====\n\ntext\n"; string(applyAll(t, raw, got)) != want {
		t.Errorf("fixed = %q, want %q", string(applyAll(t, raw, got)), want)
	}
}

func TestBlanksAroundHeadings_SetextAfterATXFlagsAbove(t *testing.T) {
	// An ATX heading directly above a setext heading is a structural boundary, so
	// the setext heading's missing blank above IS flagged (alongside the ATX's
	// missing blank below).
	raw := []byte("# ATX\nTitle\n=====\n")
	got := blanksAroundHeadingsFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
}

func TestBlanksAroundHeadings_LeavesListNestedHeading(t *testing.T) {
	// "# nested" is structurally nested in the list item; inserting a blank would
	// de-nest it, so the rule withholds the fix entirely.
	raw := []byte("- item\n  # nested\nmore\n")
	for _, f := range blanksAroundHeadingsFindings(t, raw) {
		if strings.Contains(f.Message, "above") {
			t.Errorf("nested heading should not be flagged above: %+v", f)
		}
	}
}

func TestBlanksAroundHeadings_FixIsIdempotent(t *testing.T) {
	raw := []byte("before\n# Heading\nafter\n")
	got := blanksAroundHeadingsFindings(t, raw)
	fixed := applyAll(t, raw, got)
	if again := blanksAroundHeadingsFindings(t, fixed); len(again) != 0 {
		t.Fatalf("fixed text still flagged: %d findings", len(again))
	}
}
