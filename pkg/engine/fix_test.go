package engine

import (
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/rule"
)

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
