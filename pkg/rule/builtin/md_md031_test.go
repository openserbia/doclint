package builtin_test

import (
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/engine"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func blanksAroundFencesFindings(t *testing.T, raw []byte) []rule.Finding {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out []rule.Finding
	(builtin.BlanksAroundFences{}).Check(doc, func(f rule.Finding) { out = append(out, f) })
	return out
}

func TestBlanksAroundFences_Meta(t *testing.T) {
	m := (builtin.BlanksAroundFences{}).Meta()
	if m.Name != "blanks-around-fences" {
		t.Errorf("Name = %q, want blanks-around-fences", m.Name)
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

func TestBlanksAroundFences_FlagsMissingBlankBefore(t *testing.T) {
	raw := []byte("text\n```\ncode\n```\n")
	got := blanksAroundFencesFindings(t, raw)
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
	if want := "text\n\n```\ncode\n```\n"; string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
}

func TestBlanksAroundFences_FlagsMissingBlankAfter(t *testing.T) {
	raw := []byte("```\ncode\n```\ntext\n")
	got := blanksAroundFencesFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	f := got[0]
	if f.Line != 3 {
		t.Errorf("Line = %d, want 3 (closing delimiter)", f.Line)
	}
	if !strings.Contains(f.Message, "after") {
		t.Errorf("message = %q, want it to mention after", f.Message)
	}
	fixed, err := engine.ApplyEdits(raw, f.Fixes)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if want := "```\ncode\n```\n\ntext\n"; string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
}

func TestBlanksAroundFences_FlagsBothSides(t *testing.T) {
	raw := []byte("before\n```\ncode\n```\nafter\n")
	got := blanksAroundFencesFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
	fixed, err := engine.ApplyEdits(raw, append(got[0].Fixes, got[1].Fixes...))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if want := "before\n\n```\ncode\n```\n\nafter\n"; string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
}

func TestBlanksAroundFences_ExemptsDocumentStartAndEnd(t *testing.T) {
	// Fence opens on the first line and closes on the last; both edges are
	// document boundaries, so neither delimiter is flagged.
	if got := blanksAroundFencesFindings(t, []byte("```\ncode\n```\n")); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestBlanksAroundFences_AcceptsWellSpaced(t *testing.T) {
	raw := []byte("para\n\n```\ncode\n```\n\npara\n")
	if got := blanksAroundFencesFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestBlanksAroundFences_TildeFence(t *testing.T) {
	raw := []byte("text\n~~~\ncode\n~~~\n")
	got := blanksAroundFencesFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
}

func TestBlanksAroundFences_IgnoresDelimiterTextInsideFence(t *testing.T) {
	// The interior line is fenced content (InFence), not a delimiter, so the only
	// flagged delimiter is the opening one preceded by the paragraph.
	raw := []byte("text\n```\nplain code line\n```\n")
	got := blanksAroundFencesFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
}

func TestBlanksAroundFences_BackToBackFences(t *testing.T) {
	// A closing delimiter immediately followed by the next opening delimiter:
	// the closing one wants a blank after, the opening one a blank before.
	raw := []byte("```\na\n```\n```\nb\n```\n")
	got := blanksAroundFencesFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
}

func TestBlanksAroundFences_FixIsIdempotent(t *testing.T) {
	raw := []byte("before\n```\ncode\n```\nafter\n")
	got := blanksAroundFencesFindings(t, raw)
	fixed, err := engine.ApplyEdits(raw, append(got[0].Fixes, got[1].Fixes...))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if again := blanksAroundFencesFindings(t, fixed); len(again) != 0 {
		t.Fatalf("fixed text still flagged: %d findings", len(again))
	}
}
