package builtin_test

import (
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/engine"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func findings(t *testing.T, raw []byte) []rule.Finding {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out []rule.Finding
	(builtin.DetailsBlankLine{}).Check(doc, func(f rule.Finding) { out = append(out, f) })
	return out
}

func TestDetails_MissingBlankLine(t *testing.T) {
	raw := []byte("<details><summary>x</summary>\n- item\n")
	got := findings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if got[0].Safety != rule.Safe || len(got[0].Fixes) != 1 {
		t.Errorf("expected one safe fix, got safety=%v fixes=%d", got[0].Safety, len(got[0].Fixes))
	}
}

func TestDetails_OkWithBlankLine(t *testing.T) {
	raw := []byte("<details><summary>x</summary>\n\n- item\n")
	if got := findings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestDetails_IgnoresFencedCode(t *testing.T) {
	// Relies on document.ParseMarkdown setting Line.InFence for lines inside the fence.
	raw := []byte("```html\n<details><summary>x</summary>\ncode\n```\n")
	if got := findings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings inside fence, want 0", len(got))
	}
}

func TestDetails_FixProducesBlankLine(t *testing.T) {
	raw := []byte("<details><summary>x</summary>\n- item\n")
	got := findings(t, raw)
	fixed, err := engine.ApplyEdits(raw, got[0].Fixes)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	want := "<details><summary>x</summary>\n\n- item\n"
	if string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
}
