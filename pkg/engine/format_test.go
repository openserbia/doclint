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

func TestFormat_AlignsRaggedTable(t *testing.T) {
	in := "| name | id |\n|:--|--:|\n| alice | 1 |\n| bob | 100 |\n"
	want := "" +
		"| name  | id  |\n" +
		"| :---- | --: |\n" +
		"| alice | 1   |\n" +
		"| bob   | 100 |\n"
	if got := format(t, in); got != want {
		t.Errorf("aligned table:\n got=%q\nwant=%q", got, want)
	}
}

func TestFormat_TableIdempotent(t *testing.T) {
	in := "before\n\n| a | bbbb | c |\n|---|:-:|--:|\n| 1 | 2 | 3 |\n\nafter\n"
	once := format(t, in)
	doc, _ := document.ParseMarkdown("t.md", []byte(once))
	twice := string(Format(doc))
	if once != twice {
		t.Errorf("table format not idempotent:\n once=%q\ntwice=%q", once, twice)
	}
}

func TestFormat_FixesMissingSpaceATX(t *testing.T) {
	got := format(t, "#Heading\n\ntext\n")
	want := "# Heading\n\ntext\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormat_MissingSpaceATXIdempotent(t *testing.T) {
	in := "###Glued\n\nbody with a # in prose\n"
	once := format(t, in)
	doc, _ := document.ParseMarkdown("t.md", []byte(once))
	twice := string(Format(doc))
	if once != twice {
		t.Errorf("atx format not idempotent:\n once=%q\ntwice=%q", once, twice)
	}
	if want := "### Glued\n\nbody with a # in prose\n"; once != want {
		t.Errorf("got %q, want %q", once, want)
	}
}

func TestFormat_LeavesFencedHashLine(t *testing.T) {
	in := "```\n#Heading\n```\n"
	if got := format(t, in); got != in {
		t.Errorf("fenced hash line changed: got %q want %q", got, in)
	}
}

func TestFormat_LeavesMalformedTable(t *testing.T) {
	// A row with the wrong cell count makes the table malformed; leave it as-is.
	in := "| a | b | c |\n|---|---|---|\n| x |\n"
	if got := format(t, in); got != in {
		t.Errorf("malformed table changed:\n got=%q\nwant=%q", got, in)
	}
}

func TestFormat_DedentsIndentedHeading(t *testing.T) {
	got := format(t, "  ## Indented\n\ntext\n")
	want := "## Indented\n\ntext\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormat_IndentedHeadingIdempotent(t *testing.T) {
	in := "    # Lost\n\nbody\n"
	once := format(t, in)
	doc, _ := document.ParseMarkdown("t.md", []byte(once))
	twice := string(Format(doc))
	if once != twice {
		t.Errorf("indented-heading format not idempotent:\n once=%q\ntwice=%q", once, twice)
	}
	if want := "# Lost\n\nbody\n"; once != want {
		t.Errorf("got %q, want %q", once, want)
	}
}

func TestFormat_LeavesListNestedHeading(t *testing.T) {
	// The heading is structurally nested in the list item; dedenting it would
	// de-nest it, so fmt must leave it untouched.
	in := "- item\n  # nested\n"
	if got := format(t, in); got != in {
		t.Errorf("list-nested heading changed: got %q want %q", got, in)
	}
}
