// Package report renders findings as human-readable or JSON output.
package report

import (
	"io"

	"github.com/openserbia/doclint/pkg/rule"
)

// Reporter writes findings to w.
type Reporter interface {
	Report(w io.Writer, findings []rule.Finding) error
}

func counts(findings []rule.Finding) (errors, warnings, infos int) {
	for _, f := range findings {
		switch f.Severity {
		case rule.Error:
			errors++
		case rule.Warning:
			warnings++
		case rule.Info:
			infos++
		}
	}
	return
}
