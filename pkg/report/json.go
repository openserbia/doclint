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
	}
	out := make([]wire, 0, len(findings))
	for _, f := range findings {
		out = append(out, wire{f.Rule, f.Path, f.Line, f.Col, f.Message, f.Severity.String()})
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
