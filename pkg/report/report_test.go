package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/rule"
)

var sample = []rule.Finding{
	{Rule: "details-blank-line", Path: "a.md", Line: 3, Col: 1, Message: "missing blank line", Severity: rule.Error},
	{Rule: "seo-len", Path: "a.md", Line: 1, Col: 1, Message: "too short", Severity: rule.Warning},
}

// humanSample spans two files and mixes severities (pre-sorted by path,line,col
// as the engine emits them).
var humanSample = []rule.Finding{
	{Rule: "blanks-around-lists", Path: "a.md", Line: 3, Col: 1, Message: "missing blank line before list", Severity: rule.Warning},
	{Rule: "table-column-count", Path: "a.md", Line: 12, Col: 1, Message: "table row has 1 cell(s) but the table has 3", Severity: rule.Error},
	{Rule: "no-trailing-spaces", Path: "b.md", Line: 5, Col: 9, Message: "line ends in a single stray trailing space", Severity: rule.Warning},
}

func TestHuman(t *testing.T) {
	var buf bytes.Buffer
	if err := (Human{NoColor: true}).Report(&buf, humanSample); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if strings.Contains(out, "\x1b[") {
		t.Errorf("NoColor output must not contain ANSI escapes:\n%q", out)
	}

	// Each file path appears exactly once — as its group header.
	for _, path := range []string{"a.md", "b.md"} {
		if got := strings.Count(out, path); got != 1 {
			t.Errorf("path %q should appear once as a header, got %d:\n%s", path, got, out)
		}
	}

	// Every finding row keeps a literal line:col token (IDE clickability) plus
	// its message and rule name.
	for _, f := range humanSample {
		token := locToken(f.Line, f.Col)
		if !strings.Contains(out, token) {
			t.Errorf("missing literal line:col token %q:\n%s", token, out)
		}
		if !strings.Contains(out, f.Message) {
			t.Errorf("missing message %q:\n%s", f.Message, out)
		}
		if !strings.Contains(out, f.Rule) {
			t.Errorf("missing rule %q:\n%s", f.Rule, out)
		}
	}

	// Footer counts: 3 findings, 1 error, 2 warnings, across 2 files.
	for _, want := range []string{"3 problems", "1 error", "2 warnings", "across 2 files"} {
		if !strings.Contains(out, want) {
			t.Errorf("footer missing %q:\n%s", want, out)
		}
	}
}

func TestHumanEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := (Human{NoColor: true}).Report(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "✓ no problems") {
		t.Errorf("zero-findings output should report a clean line:\n%s", buf.String())
	}
}

func TestCompact(t *testing.T) {
	var buf bytes.Buffer
	if err := (Compact{NoColor: true}).Report(&buf, sample); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "a.md:3:1") || !strings.Contains(out, "[details-blank-line]") {
		t.Errorf("compact output missing location/rule:\n%s", out)
	}
	if !strings.Contains(out, "1 error") || !strings.Contains(out, "1 warning") {
		t.Errorf("compact output missing summary:\n%s", out)
	}
}

func TestJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := (JSON{}).Report(&buf, sample); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"severity": "error"`) ||
		!strings.Contains(buf.String(), `"rule": "details-blank-line"`) {
		t.Errorf("json missing fields:\n%s", buf.String())
	}
	if !json.Valid(buf.Bytes()) {
		t.Error("output is not valid JSON")
	}
}

func locToken(line, col int) string {
	return itoa(line) + ":" + itoa(col)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
