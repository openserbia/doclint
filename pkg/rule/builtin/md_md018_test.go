package builtin_test

import (
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/engine"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func atxFindings(t *testing.T, raw []byte) []rule.Finding {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out []rule.Finding
	(builtin.NoMissingSpaceATX{}).Check(doc, func(f rule.Finding) { out = append(out, f) })
	return out
}

func TestNoMissingSpaceATX_Meta(t *testing.T) {
	m := (builtin.NoMissingSpaceATX{}).Meta()
	if m.Name != "no-missing-space-atx" {
		t.Errorf("Name = %q, want no-missing-space-atx", m.Name)
	}
	if m.Severity != rule.Error {
		t.Errorf("Severity = %v, want Error", m.Severity)
	}
	if m.Safety != rule.Safe {
		t.Errorf("Safety = %v, want Safe", m.Safety)
	}
	if !m.AppliesTo(document.Markdown) {
		t.Error("rule should apply to markdown")
	}
}

func TestNoMissingSpaceATX_FlagsGluedHeading(t *testing.T) {
	got := atxFindings(t, []byte("#Heading\n"))
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	f := got[0]
	if f.Severity != rule.Error || f.Line != 1 || f.Col != 2 {
		t.Errorf("finding = %+v, want Error at line 1 col 2", f)
	}
	if f.Safety != rule.Safe || len(f.Fixes) != 1 {
		t.Fatalf("expected one safe fix, got safety=%v fixes=%d", f.Safety, len(f.Fixes))
	}
}

func TestNoMissingSpaceATX_FixInsertsSingleSpace(t *testing.T) {
	raw := []byte("#Heading\n")
	got := atxFindings(t, raw)
	fixed, err := engine.ApplyEdits(raw, got[0].Fixes)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if want := "# Heading\n"; string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
}

func TestNoMissingSpaceATX_FixIsIdempotent(t *testing.T) {
	// Applying the fix yields a line that no longer matches the rule.
	raw := []byte("###Heading\n")
	got := atxFindings(t, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	fixed, err := engine.ApplyEdits(raw, got[0].Fixes)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if want := "### Heading\n"; string(fixed) != want {
		t.Errorf("fixed = %q, want %q", string(fixed), want)
	}
	if again := atxFindings(t, fixed); len(again) != 0 {
		t.Fatalf("fixed text still flagged: %d findings", len(again))
	}
}

func TestNoMissingSpaceATX_IndentUpToThree(t *testing.T) {
	got := atxFindings(t, []byte("   #Heading\n"))
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if got[0].Col != 5 { // 3 indent + 1 hash + 1
		t.Errorf("Col = %d, want 5", got[0].Col)
	}
}

func TestNoMissingSpaceATX_IndentFourIsCodeBlock(t *testing.T) {
	// Four leading spaces is an indented code block, not a heading.
	if got := atxFindings(t, []byte("    #Heading\n")); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestNoMissingSpaceATX_AcceptsValidHeadings(t *testing.T) {
	cases := []string{
		"# Heading\n",
		"## Sub\n",
		"###### Deep\n",
		"#\ttab-after-hash\n", // a tab also separates the marker
	}
	for _, c := range cases {
		if got := atxFindings(t, []byte(c)); len(got) != 0 {
			t.Errorf("%q: got %d findings, want 0", c, len(got))
		}
	}
}

func TestNoMissingSpaceATX_NonHeadingShapes(t *testing.T) {
	cases := []string{
		"###\n",           // all hashes, no text
		"#\n",             // single hash, no text
		"#######x\n",      // seven hashes is over the ATX limit
		"text #Heading\n", // hash not at line start
		"#1 issue\n",      // digit follows: hashtag false-positive, suppressed
	}
	for _, c := range cases {
		if got := atxFindings(t, []byte(c)); len(got) != 0 {
			t.Errorf("%q: got %d findings, want 0", c, len(got))
		}
	}
}

func TestNoMissingSpaceATX_IgnoresFence(t *testing.T) {
	raw := []byte("```\n#Heading\n```\n")
	if got := atxFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings inside fence, want 0", len(got))
	}
}

func TestNoMissingSpaceATX_IgnoresFrontmatter(t *testing.T) {
	// A "#comment"-shaped line inside YAML frontmatter must not be flagged.
	raw := []byte("---\ntitle: Hello\n---\n\nbody\n")
	if got := atxFindings(t, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
	// And a real glued heading in the body IS flagged (frontmatter offset honored).
	raw = []byte("---\ntitle: Hello\n---\n\n#Body\n")
	got := atxFindings(t, raw)
	if len(got) != 1 || got[0].Line != 5 {
		t.Fatalf("got %+v, want 1 finding on line 5", got)
	}
}
