package engine

import (
	"testing"

	"github.com/openserbia/doclint/pkg/document"
)

func format(t *testing.T, raw string) string {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", []byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return string(Format(doc))
}

func TestFormat_CollapsesBlankLinesAndFinalNewline(t *testing.T) {
	got := format(t, "a\n\n\n\nb")
	want := "a\n\nb\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormat_PreservesFencedInterior(t *testing.T) {
	in := "```\n\n\n\n```\n"
	if got := format(t, in); got != in {
		t.Errorf("fenced interior changed: got %q want %q", got, in)
	}
}

func TestFormat_Idempotent(t *testing.T) {
	in := "x\n\n\n\ny<details><summary>s</summary>\n- i\n"
	once := format(t, in)
	doc, _ := document.ParseMarkdown("t.md", []byte(once))
	twice := string(Format(doc))
	if once != twice {
		t.Errorf("not idempotent:\n once=%q\ntwice=%q", once, twice)
	}
}
