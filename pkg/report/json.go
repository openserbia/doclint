package report

import (
	"encoding/json"
	"io"

	"github.com/openserbia/doclint/pkg/rule"
)

// JSON renders findings as a JSON array (stable machine schema).
type JSON struct{}

// Report marshals findings; severity is rendered via its String form.
func (j JSON) Report(w io.Writer, findings []rule.Finding) error {
	type wire struct {
		Rule     string `json:"rule"`
		Path     string `json:"path"`
		Line     int    `json:"line"`
		Col      int    `json:"col"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
		Fixable  bool   `json:"fixable"`
		Fix      string `json:"fix"` // safe | unsafe | none
	}
	out := make([]wire, 0, len(findings))
	for _, f := range findings {
		out = append(out, wire{
			Rule: f.Rule, Path: f.Path, Line: f.Line, Col: f.Col,
			Message: f.Message, Severity: f.Severity.String(),
			Fixable: f.Safety == rule.Safe || f.Safety == rule.Unsafe,
			Fix:     fixString(f.Safety),
		})
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func fixString(s rule.FixSafety) string {
	switch s {
	case rule.Safe:
		return "safe"
	case rule.Unsafe:
		return "unsafe"
	default:
		return "none"
	}
}
