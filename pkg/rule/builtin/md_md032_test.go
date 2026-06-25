package builtin_test

import (
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/engine"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func blanksAroundListsFindings(t *testing.T, raw []byte) []rule.Finding {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out []rule.Finding
	(builtin.BlanksAroundLists{}).Check(doc, func(f rule.Finding) { out = append(out, f) })
	return out
}

func applyAll(t *testing.T, raw []byte, fs []rule.Finding) []byte {
	t.Helper()
	var edits []rule.TextEdit
	for _, f := range fs {
		edits = append(edits, f.Fixes...)
	}
	out, err := engine.ApplyEdits(raw, edits)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	return out
}

func TestBlanksAroundLists_Meta(t *testing.T) {
	m := (builtin.BlanksAroundLists{}).Meta()
	if m.Name != "blanks-around-lists" {
		t.Errorf("Name = %q, want blanks-around-lists", m.Name)
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

func TestBlanksAroundLists_FlagsMissingBlankBefore(t *testing.T) {
	raw := []byte("para\n- item\n")
	got := blanksAroundListsFindings(t, raw)
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
	if want := "para\n\n- item\n"; string(applyAll(t, raw, got)) != want {
		t.Errorf("fixed = %q, want %q", string(applyAll(t, raw, got)), want)
	}
}

func TestBlanksAroundLists_FlagsMissingBlankAfter(t *testing.T) {
	raw := []byte("- item\npara\n")
	got := blanksAroundListsFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	f := got[0]
	if f.Line != 1 {
		t.Errorf("Line = %d, want 1 (last item line)", f.Line)
	}
	if !strings.Contains(f.Message, "after") {
		t.Errorf("message = %q, want it to mention after", f.Message)
	}
	if want := "- item\n\npara\n"; string(applyAll(t, raw, got)) != want {
		t.Errorf("fixed = %q, want %q", string(applyAll(t, raw, got)), want)
	}
}

func TestBlanksAroundLists_FlagsBothSides(t *testing.T) {
	raw := []byte("before\n- item\nafter\n")
	got := blanksAroundListsFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
	if want := "before\n\n- item\n\nafter\n"; string(applyAll(t, raw, got)) != want {
		t.Errorf("fixed = %q, want %q", string(applyAll(t, raw, got)), want)
	}
}

func TestBlanksAroundLists_AcceptsWellSpaced(t *testing.T) {
	raw := []byte("para\n\n- item\n\npara\n")
	if got := blanksAroundListsFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestBlanksAroundLists_ExemptsDocumentStartAndEnd(t *testing.T) {
	// The list opens on the first line and closes on the last; both edges are
	// document boundaries, so neither is flagged.
	if got := blanksAroundListsFindings(t, []byte("- a\n- b\n")); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestBlanksAroundLists_OrderedList(t *testing.T) {
	raw := []byte("text\n1. one\n2. two\nafter\n")
	got := blanksAroundListsFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
	if want := "text\n\n1. one\n2. two\n\nafter\n"; string(applyAll(t, raw, got)) != want {
		t.Errorf("fixed = %q, want %q", string(applyAll(t, raw, got)), want)
	}
}

func TestBlanksAroundLists_LooseListSpansInteriorBlank(t *testing.T) {
	// One interior blank line keeps the two items in a single region, so the
	// region's edges (against intro / outro) are each flagged exactly once.
	raw := []byte("intro\n- a\n\n- b\noutro\n")
	got := blanksAroundListsFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
	if want := "intro\n\n- a\n\n- b\n\noutro\n"; string(applyAll(t, raw, got)) != want {
		t.Errorf("fixed = %q, want %q", string(applyAll(t, raw, got)), want)
	}
}

func TestBlanksAroundLists_ContinuationLineIsPartOfRegion(t *testing.T) {
	// The indented "continued" line belongs to the item, so the after-edge sits
	// past it, not between the marker line and its continuation.
	raw := []byte("intro\n- a\n  continued\noutro\n")
	got := blanksAroundListsFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
	if want := "intro\n\n- a\n  continued\n\noutro\n"; string(applyAll(t, raw, got)) != want {
		t.Errorf("fixed = %q, want %q", string(applyAll(t, raw, got)), want)
	}
}

func TestBlanksAroundLists_NestedItemIsPartOfRegion(t *testing.T) {
	raw := []byte("intro\n- a\n  - nested\noutro\n")
	got := blanksAroundListsFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
	if want := "intro\n\n- a\n  - nested\n\noutro\n"; string(applyAll(t, raw, got)) != want {
		t.Errorf("fixed = %q, want %q", string(applyAll(t, raw, got)), want)
	}
}

func TestBlanksAroundLists_IgnoresListMarkerInsideFence(t *testing.T) {
	// "- x" lives inside a fenced code block (InFence), so it must not be read as
	// a list item and nothing is flagged.
	raw := []byte("text\n\n```\n- x\n```\n\ntext\n")
	if got := blanksAroundListsFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestBlanksAroundLists_SkipsFrontmatterList(t *testing.T) {
	// A YAML list in frontmatter is not a markdown list; touching it would corrupt
	// the frontmatter, so it is skipped entirely.
	raw := []byte("---\ntitle: x\ntags:\n  - a\n  - b\n---\n\nBody text\n")
	if got := blanksAroundListsFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestBlanksAroundLists_TwoBlankLinesEndRegion(t *testing.T) {
	// Two consecutive blank lines end the list, so each well-separated list is
	// already correctly spaced and nothing is flagged.
	raw := []byte("- a\n\n\n- b\n")
	if got := blanksAroundListsFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestBlanksAroundLists_FixIsIdempotent(t *testing.T) {
	raw := []byte("before\n- item\nafter\n")
	got := blanksAroundListsFindings(t, raw)
	fixed := applyAll(t, raw, got)
	if again := blanksAroundListsFindings(t, fixed); len(again) != 0 {
		t.Fatalf("fixed text still flagged: %d findings", len(again))
	}
}
