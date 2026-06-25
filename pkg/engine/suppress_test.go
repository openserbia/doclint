package engine

import (
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

func TestSuppress_NextLineAndUnused(t *testing.T) {
	raw := []byte("<!-- doclint-disable-next-line details-blank-line -->\nline2\n<!-- doclint-disable-next-line other-rule -->\nline4\n")
	doc, _ := document.ParseMarkdown("t.md", raw)
	s := NewSuppressor(doc)

	// A finding on line 2 for details-blank-line is suppressed.
	f := rule.Finding{Rule: "details-blank-line", Line: 2}
	if !s.Suppressed(f) {
		t.Error("expected finding on line 2 to be suppressed")
	}
	// Nothing matched the line-4 directive -> it is unused.
	unused := s.Unused()
	if len(unused) != 1 {
		t.Fatalf("unused = %d, want 1", len(unused))
	}
	if unused[0].Rule != "unused-suppression" || !strings.Contains(unused[0].Message, "other-rule") {
		t.Errorf("unused finding = %+v, want rule=unused-suppression mentioning other-rule", unused[0])
	}
}

func TestSuppress_IgnoresDirectiveInFence(t *testing.T) {
	raw := []byte("```\n<!-- doclint-disable-next-line foo -->\n```\nline4\n")
	doc, _ := document.ParseMarkdown("t.md", raw)
	s := NewSuppressor(doc)
	if len(s.Unused()) != 0 {
		t.Errorf("directive inside a fenced block must be ignored, got %d unused", len(s.Unused()))
	}
}
