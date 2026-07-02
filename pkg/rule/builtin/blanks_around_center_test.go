package builtin_test

import (
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/engine"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func blanksAroundCenterFindings(t *testing.T, raw []byte) []rule.Finding {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out []rule.Finding
	(builtin.BlanksAroundCenter{}).Check(doc, func(f rule.Finding) { out = append(out, f) })
	return out
}

func TestBlanksAroundCenter_Meta(t *testing.T) {
	m := (builtin.BlanksAroundCenter{}).Meta()
	if m.Name != "blanks-around-center" {
		t.Errorf("Name = %q, want blanks-around-center", m.Name)
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

func TestBlanksAroundCenter_FlagsMissingBlankAfterClose(t *testing.T) {
	raw := []byte("<center>\n{{< figure src=\"img.png\" >}}\n</center>\n- next item\n")
	got := blanksAroundCenterFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	f := got[0]
	if f.Severity != rule.Warning || f.Line != 3 {
		t.Errorf("finding = %+v, want Warning at line 3", f)
	}
	if !strings.Contains(f.Message, "after") {
		t.Errorf("message = %q, want it to mention after", f.Message)
	}
	if f.Safety != rule.Safe || len(f.Fixes) != 1 {
		t.Fatalf("expected one safe fix, got safety=%v fixes=%d", f.Safety, len(f.Fixes))
	}
	fixed, err := engine.ApplyEdits(raw, f.Fixes)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if want := "<center>\n{{< figure src=\"img.png\" >}}\n</center>\n\n- next item\n"; string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
}

func TestBlanksAroundCenter_FlagsMissingBlankBeforeOpen(t *testing.T) {
	raw := []byte("- list item\n<center>\n{{< figure src=\"img.png\" >}}\n</center>\n")
	got := blanksAroundCenterFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	f := got[0]
	if f.Line != 2 {
		t.Errorf("Line = %d, want 2 (opening tag)", f.Line)
	}
	if !strings.Contains(f.Message, "before") {
		t.Errorf("message = %q, want it to mention before", f.Message)
	}
	fixed, err := engine.ApplyEdits(raw, f.Fixes)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if want := "- list item\n\n<center>\n{{< figure src=\"img.png\" >}}\n</center>\n"; string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
}

func TestBlanksAroundCenter_FlagsBothSides(t *testing.T) {
	raw := []byte("- item above\n<center>\n{{< figure src=\"img.png\" >}}\n</center>\n- item below\n")
	got := blanksAroundCenterFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
	fixed, err := engine.ApplyEdits(raw, append(got[0].Fixes, got[1].Fixes...))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	want := "- item above\n\n<center>\n{{< figure src=\"img.png\" >}}\n</center>\n\n- item below\n"
	if string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
}

func TestBlanksAroundCenter_ExemptsDocumentStartAndEnd(t *testing.T) {
	raw := []byte("<center>\n{{< figure src=\"img.png\" >}}\n</center>\n")
	if got := blanksAroundCenterFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestBlanksAroundCenter_AcceptsWellSpaced(t *testing.T) {
	raw := []byte("para\n\n<center>\n{{< figure src=\"img.png\" >}}\n</center>\n\npara\n")
	if got := blanksAroundCenterFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestBlanksAroundCenter_IndentedTags(t *testing.T) {
	raw := []byte("- item\n   <center>\n   {{< figure src=\"img.png\" >}}\n   </center>\n   - next\n")
	got := blanksAroundCenterFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2 (before <center> and after </center>)", len(got))
	}
}

func TestBlanksAroundCenter_CaseInsensitive(t *testing.T) {
	raw := []byte("text\n<CENTER>\nimg\n</CENTER>\nmore\n")
	got := blanksAroundCenterFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
}

func TestBlanksAroundCenter_IgnoresInsideFence(t *testing.T) {
	raw := []byte("```html\n<center>\ncode\n</center>\n```\n")
	if got := blanksAroundCenterFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0 (inside fence)", len(got))
	}
}

func TestBlanksAroundCenter_FixIsIdempotent(t *testing.T) {
	raw := []byte("before\n<center>\nimg\n</center>\nafter\n")
	got := blanksAroundCenterFindings(t, raw)
	fixed, err := engine.ApplyEdits(raw, append(got[0].Fixes, got[1].Fixes...))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if again := blanksAroundCenterFindings(t, fixed); len(again) != 0 {
		t.Fatalf("fixed text still flagged: %d findings", len(again))
	}
}

func TestBlanksAroundCenter_RealWorldSrbGuide(t *testing.T) {
	// Reproduces the exact pattern from srb.guide content files: list item
	// directly after </center> with no separating blank line.
	raw := []byte("- Вход в основном здании\n<center>\n{{< figure link=\"media/img.png\" src=\"media/img.png\" width=250 >}}\n</center>\n- Вход не в основном здании\n")
	got := blanksAroundCenterFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2 (before <center> and after </center>)", len(got))
	}
	fixed, err := engine.ApplyEdits(raw, append(got[0].Fixes, got[1].Fixes...))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	want := "- Вход в основном здании\n\n<center>\n{{< figure link=\"media/img.png\" src=\"media/img.png\" width=250 >}}\n</center>\n\n- Вход не в основном здании\n"
	if string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
}
