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

func TestHuman(t *testing.T) {
	var buf bytes.Buffer
	if err := (Human{NoColor: true}).Report(&buf, sample); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "a.md:3:1") || !strings.Contains(out, "[details-blank-line]") {
		t.Errorf("human output missing location/rule:\n%s", out)
	}
	if !strings.Contains(out, "1 error") || !strings.Contains(out, "1 warning") {
		t.Errorf("human output missing summary:\n%s", out)
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
