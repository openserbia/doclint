package builtin_test

import (
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func TestNoForbiddenSymbols(t *testing.T) {
	raw := []byte("---\ntitle: A · B\n---\nПривет — word · end\n`— ·`\n```text\n— ·\n```\n")
	doc, err := document.ParseMarkdown("test.md", raw)
	if err != nil {
		t.Fatal(err)
	}
	var got []rule.Finding
	(builtin.NoForbiddenSymbols{}).Check(doc, func(f rule.Finding) { got = append(got, f) })
	want := []struct {
		line, col int
		message   string
	}{
		{2, 10, "forbidden symbol '·'"},
		{4, 8, "forbidden symbol '—'"},
		{4, 15, "forbidden symbol '·'"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d findings, want %d: %+v", len(got), len(want), got)
	}
	for i, f := range got {
		if f.Rule != "no-forbidden-symbols" || f.Line != want[i].line || f.Col != want[i].col || f.Message != want[i].message || f.Severity != rule.Warning || f.Safety != rule.NoFix || len(f.Fixes) != 0 {
			t.Errorf("finding %d = %+v, want line %d col %d message %q with warning and no fix", i, f, want[i].line, want[i].col, want[i].message)
		}
	}
}
