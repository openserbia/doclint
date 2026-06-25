package builtin_test

import (
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func fencedCodeLanguageFindings(t *testing.T, raw []byte) []rule.Finding {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out []rule.Finding
	(builtin.FencedCodeLanguage{}).Check(doc, func(f rule.Finding) { out = append(out, f) })
	return out
}

func TestFencedCodeLanguage_Meta(t *testing.T) {
	m := (builtin.FencedCodeLanguage{}).Meta()
	if m.Name != "fenced-code-language" {
		t.Errorf("Name = %q, want fenced-code-language", m.Name)
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

func TestFencedCodeLanguage_FlagsEmptyInfoString(t *testing.T) {
	raw := []byte("```\ncode\n```\n")
	got := fencedCodeLanguageFindings(t, raw)
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
	if !strings.Contains(f.Message, "language") {
		t.Errorf("message = %q, want it to mention language", f.Message)
	}
}

func TestFencedCodeLanguage_AcceptsLanguage(t *testing.T) {
	raw := []byte("```python\ncode\n```\n")
	if got := fencedCodeLanguageFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestFencedCodeLanguage_IgnoresClosingDelimiter(t *testing.T) {
	// The closing ``` carries no info string but must not be flagged; only the
	// language-less opener counts.
	raw := []byte("text\n\n```\ncode\n```\n\nmore\n")
	got := fencedCodeLanguageFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1 (opener only)", len(got))
	}
	if got[0].Line != 3 {
		t.Errorf("Line = %d, want 3 (opening delimiter)", got[0].Line)
	}
}

func TestFencedCodeLanguage_TildeFenceEmpty(t *testing.T) {
	raw := []byte("~~~\ncode\n~~~\n")
	if got := fencedCodeLanguageFindings(t, raw); len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
}

func TestFencedCodeLanguage_TildeFenceWithLanguage(t *testing.T) {
	raw := []byte("~~~ruby\ncode\n~~~\n")
	if got := fencedCodeLanguageFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestFencedCodeLanguage_IndentedFence(t *testing.T) {
	// Leading whitespace is stripped before reading the info string.
	raw := []byte("  ```\n  code\n  ```\n")
	if got := fencedCodeLanguageFindings(t, raw); len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
}

func TestFencedCodeLanguage_TrailingWhitespaceOnlyInfo(t *testing.T) {
	// An info string of only spaces TrimSpaces to empty and is flagged.
	raw := []byte("```   \ncode\n```\n")
	if got := fencedCodeLanguageFindings(t, raw); len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
}

func TestFencedCodeLanguage_LongerFenceRunEmpty(t *testing.T) {
	raw := []byte("````\ncode\n````\n")
	if got := fencedCodeLanguageFindings(t, raw); len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
}

func TestFencedCodeLanguage_LongerFenceRunWithLanguage(t *testing.T) {
	raw := []byte("````go\ncode\n````\n")
	if got := fencedCodeLanguageFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestFencedCodeLanguage_MultipleFences(t *testing.T) {
	raw := []byte("```\nno-lang\n```\n\n```sh\nlang\n```\n\n```\nno-lang-again\n```\n")
	got := fencedCodeLanguageFindings(t, raw)
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2", len(got))
	}
	if got[0].Line != 1 || got[1].Line != 9 {
		t.Errorf("lines = %d,%d, want 1,9", got[0].Line, got[1].Line)
	}
}
