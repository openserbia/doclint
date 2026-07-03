package engine

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/config"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

// TestAllBuiltinRules_Regression runs a single markdown file that contains one
// violation per builtin rule. It asserts that every registered builtin rule
// fires at least once, so a future refactor that silently breaks a rule's
// detection fails this test.
//
// If you add a new builtin rule, add a corresponding violation to the fixture
// below and update wantRules.
func TestAllBuiltinRules_Regression(t *testing.T) {
	// Fixture: every section triggers exactly one rule.
	// Lines are annotated with the rule they trigger.
	fixture := strings.Join([]string{
		"---",
		"title: regression test",
		"---",
		"",
		// details-blank-line: missing blank after </summary>
		"<details><summary>click</summary>",
		"- item inside details",
		"</details>",
		"",
		// table-column-count: row has 3 cells, header has 2
		"| A | B |",
		"| - | - |",
		"| 1 | 2 | 3 |",
		"",
		// no-missing-space-atx: no space after #
		"##Glued heading",
		"",
		// heading-start-left: heading indented by 2 spaces
		"  ## Indented heading",
		"",
		// blanks-around-fences: fence not surrounded by blank lines
		"Text before fence.",
		"```go",
		"x := 1",
		"```",
		"Text after fence.",
		"",
		// blanks-around-thematic-break: *** not surrounded by blank lines
		// (--- would be parsed as setext heading underline, which the rule excludes)
		"",
		"Paragraph above thematic break.",
		"***",
		"Paragraph below thematic break.",
		"",
		// blanks-around-lists: list not surrounded by blank lines
		"Paragraph before list.",
		"- lonely list item",
		"Paragraph after list.",
		"",
		// blanks-around-headings: heading not surrounded by blank lines
		"Some text.",
		"## Heading without blank",
		"More text.",
		"",
		// fenced-code-language: fence with no language
		"",
		"```",
		"no language",
		"```",
		"",
		// no-alt-text: image with empty alt
		"![](https://example.com/img.png)",
		"",
		// no-trailing-spaces: trailing space
		"trailing space here \n",
		// no-broken-anchor: anchor that doesn't exist
		"[broken](#nonexistent-anchor)",
		"",
		// list-marker-indent: body indented to 2 instead of 3
		"1. Ordered item",
		"  body under-indented",
		"",
		// blanks-around-center: center not surrounded by blank lines
		"Text before center.",
		"<center>",
		"centered content",
		"</center>",
		"Text after center.",
		"",
	}, "\n")

	dir := t.TempDir()
	md := filepath.Join(dir, "regression.md")
	if err := os.WriteFile(md, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}

	reg := rule.NewRegistry()
	builtin.Register(reg)

	cfg := config.Default()
	eng, err := New(cfg, reg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res, err := eng.Run(context.Background(), []string{md})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Collect which rules fired.
	fired := map[string]bool{}
	for _, f := range res.Findings {
		fired[f.Rule] = true
	}

	// Every builtin rule must fire. If a rule is missing, the fixture needs
	// an additional violation, or the rule's detection is broken.
	wantRules := []string{
		"details-blank-line",
		"table-column-count",
		"no-missing-space-atx",
		"heading-start-left",
		"blanks-around-fences",
		"blanks-around-thematic-break",
		"blanks-around-lists",
		"blanks-around-headings",
		"fenced-code-language",
		"no-alt-text",
		"no-trailing-spaces",
		"no-broken-anchor",
		"list-marker-indent",
		"blanks-around-center",
	}

	sort.Strings(wantRules)
	var missing []string
	for _, r := range wantRules {
		if !fired[r] {
			missing = append(missing, r)
		}
	}
	if len(missing) > 0 {
		// Show all findings for debugging.
		var lines []string
		for _, f := range res.Findings {
			lines = append(lines, f.Rule)
		}
		sort.Strings(lines)
		t.Errorf("rules that did NOT fire: %v\nrules that fired: %v", missing, lines)
	}
}

// TestAllBuiltinRules_CleanFile_NoFindings verifies that a well-formed markdown
// file produces zero findings — a regression guard against false positives.
func TestAllBuiltinRules_CleanFile_NoFindings(t *testing.T) {
	clean := strings.Join([]string{
		"---",
		"title: clean file",
		"---",
		"",
		"## First heading",
		"",
		"A paragraph of text.",
		"",
		"- List item one",
		"- List item two",
		"",
		"## Second heading",
		"",
		"```go",
		"x := 1",
		"```",
		"",
		"| A | B |",
		"| - | - |",
		"| 1 | 2 |",
		"",
		"![Alt text](https://example.com/img.png)",
		"",
		"[link](#first-heading)",
		"",
	}, "\n")

	dir := t.TempDir()
	md := filepath.Join(dir, "clean.md")
	if err := os.WriteFile(md, []byte(clean), 0o600); err != nil {
		t.Fatal(err)
	}

	reg := rule.NewRegistry()
	builtin.Register(reg)

	cfg := config.Default()
	eng, err := New(cfg, reg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	res, err := eng.Run(context.Background(), []string{md})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(res.Findings) != 0 {
		var lines []string
		for _, f := range res.Findings {
			lines = append(lines, f.Rule+": "+f.Message)
		}
		t.Errorf("expected 0 findings on a clean file, got %d:\n  %s",
			len(res.Findings), strings.Join(lines, "\n  "))
	}
}
