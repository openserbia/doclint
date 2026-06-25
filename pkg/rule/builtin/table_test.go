package builtin_test

import (
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func tableFindings(t *testing.T, raw []byte) []rule.Finding {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out []rule.Finding
	(builtin.TableColumnCount{}).Check(doc, func(f rule.Finding) { out = append(out, f) })
	return out
}

func TestTableColumnCount_Meta(t *testing.T) {
	m := (builtin.TableColumnCount{}).Meta()
	if m.Name != "table-column-count" {
		t.Errorf("Name = %q, want table-column-count", m.Name)
	}
	if m.Severity != rule.Error {
		t.Errorf("Severity = %v, want Error", m.Severity)
	}
	if m.Safety != rule.NoFix {
		t.Errorf("Safety = %v, want NoFix", m.Safety)
	}
	if !m.AppliesTo(document.Markdown) {
		t.Error("rule should apply to markdown")
	}
}

func TestTableColumnCount_MalformedRow(t *testing.T) {
	raw := []byte("| a | b | c |\n|---|---|---|\n| x |\n")
	got := tableFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if got[0].Severity != rule.Error || got[0].Line != 3 {
		t.Errorf("finding = %+v, want Error on line 3", got[0])
	}
	if len(got[0].Fixes) != 0 {
		t.Errorf("rule must not emit fixes, got %d", len(got[0].Fixes))
	}
}

func TestTableColumnCount_WellFormed(t *testing.T) {
	raw := []byte("| a | b | c |\n|---|---|---|\n| 1 | 2 | 3 |\n| 4 | 5 | 6 |\n")
	if got := tableFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestTableColumnCount_IgnoresFenced(t *testing.T) {
	raw := []byte("```\n| a | b | c |\n|---|---|---|\n| x |\n```\n")
	if got := tableFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings inside fence, want 0", len(got))
	}
}
