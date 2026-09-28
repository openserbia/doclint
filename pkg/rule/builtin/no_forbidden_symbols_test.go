package builtin_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func TestNoForbiddenSymbols(t *testing.T) {
	raw := []byte("---\ntitle: A · B\n---\nПривет — word · end\n`— · ;`\n```text\n— · ;\n```\nOne; two&nbsp;three &#8212; &#x2014; &amp;\n")
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
		{9, 4, "forbidden symbol ';'"},
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

func TestNoForbiddenSymbolsWithOptions(t *testing.T) {
	raw := []byte("A — b; c! d¡\n`!`\n")
	doc, err := document.ParseMarkdown("test.md", raw)
	if err != nil {
		t.Fatal(err)
	}
	check := func(symbols []string) []rule.Finding {
		t.Helper()
		r, err := (builtin.NoForbiddenSymbols{}).WithOptions(rule.Options{Symbols: symbols})
		if err != nil {
			t.Fatal(err)
		}
		var got []rule.Finding
		r.Check(doc, func(f rule.Finding) { got = append(got, f) })
		return got
	}

	got := check([]string{"!", "¡"})
	if len(got) != 2 || got[0].Col != 9 || got[0].Message != "forbidden symbol '!'" || got[1].Col != 12 || got[1].Message != "forbidden symbol '¡'" {
		t.Errorf("custom list: got %+v, want '!' at col 9 and '¡' at col 12 only", got)
	}
	if got := check([]string{}); len(got) != 0 {
		t.Errorf("empty list: got %+v, want no findings", got)
	}
	for _, bad := range []string{"", "ab"} {
		if _, err := (builtin.NoForbiddenSymbols{}).WithOptions(rule.Options{Symbols: []string{bad}}); err == nil {
			t.Errorf("symbol %q: want an error", bad)
		}
	}
}

func TestNoForbiddenSymbolsSkipsMarkup(t *testing.T) {
	raw := []byte(strings.Join([]string{
		`<style>`,
		`.a{color:red;margin:0}`,
		`</style>`,
		`<div style="padding: 1rem;`,
		`  color: white;">Prose; flagged</div>`,
		`{{< uf-field options="53=a;153=b" >}}`,
		`{{< figure`,
		`    caption="x; y" >}}`,
		`<script>let a = 1;</script> after; flagged`,
		`a < b; flagged — too`,
		`<br/>ok · flagged`,
	}, "\n") + "\n")
	doc, err := document.ParseMarkdown("test.md", raw)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	(builtin.NoForbiddenSymbols{}).Check(doc, func(f rule.Finding) {
		got = append(got, fmt.Sprintf("%d:%d %s", f.Line, f.Col, f.Message))
	})
	want := []string{
		"5:23 forbidden symbol ';'",
		"9:34 forbidden symbol ';'",
		"10:6 forbidden symbol ';'",
		"10:16 forbidden symbol '—'",
		"11:9 forbidden symbol '·'",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
