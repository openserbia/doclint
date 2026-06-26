package engine

import (
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func TestCoalesceBlankInserts_HeadingThenList(t *testing.T) {
	// A heading butted against a list: blanks-around-headings wants a blank after
	// the heading and blanks-around-lists wants one before the list — both bracket
	// the same newline. Applying both must yield ONE blank line, not two.
	raw := []byte("## Heading\n- item one\n- item two\nplain trailing para\n")
	doc, err := document.ParseMarkdown("t.md", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var edits []rule.TextEdit
	collect := func(f rule.Finding) {
		if f.Safety == rule.Safe {
			edits = append(edits, f.Fixes...)
		}
	}
	(builtin.BlanksAroundHeadings{}).Check(doc, collect)
	(builtin.BlanksAroundLists{}).Check(doc, collect)

	out, err := ApplyEdits(raw, coalesceBlankInserts(raw, edits))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if strings.Contains(string(out), "\n\n\n") {
		t.Errorf("fix stacked two blank lines:\n%q", out)
	}
	if !strings.Contains(string(out), "## Heading\n\n- item one") {
		t.Errorf("want a single blank between heading and list, got:\n%q", out)
	}
}

func TestApplyEdits_OrdersAndSplices(t *testing.T) {
	src := []byte("hello world")
	edits := []rule.TextEdit{
		{Start: 6, End: 11, NewText: "there"},
		{Start: 0, End: 5, NewText: "HI"},
	}
	got, err := ApplyEdits(src, edits)
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	if string(got) != "HI there" {
		t.Errorf("got %q, want %q", got, "HI there")
	}
}

func TestApplyEdits_RejectsOverlap(t *testing.T) {
	src := []byte("abcdef")
	edits := []rule.TextEdit{
		{Start: 0, End: 3, NewText: "x"},
		{Start: 2, End: 5, NewText: "y"},
	}
	if _, err := ApplyEdits(src, edits); err == nil {
		t.Error("expected overlap error")
	}
}

func TestUnifiedDiff(t *testing.T) {
	d := UnifiedDiff("a.md", []byte("one\ntwo\n"), []byte("one\n2\n"))
	if !strings.Contains(d, "-two") || !strings.Contains(d, "+2") {
		t.Errorf("diff missing changes:\n%s", d)
	}
}
