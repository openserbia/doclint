package report

import (
	"fmt"
	"io"

	"github.com/openserbia/doclint/pkg/rule"
)

// ANSI colors keyed by severity; disabled when NoColor is set.
const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiYellow = "\033[33m"
	ansiBlue   = "\033[34m"
	ansiDim    = "\033[2m"
)

// Human renders colored `path:line:col [rule] severity message` lines.
type Human struct{ NoColor bool }

func (h Human) color(s rule.Severity) string {
	if h.NoColor {
		return ""
	}
	switch s {
	case rule.Error:
		return ansiRed
	case rule.Warning:
		return ansiYellow
	default:
		return ansiBlue
	}
}

func (h Human) reset() string {
	if h.NoColor {
		return ""
	}
	return ansiReset
}

func (h Human) dim() string {
	if h.NoColor {
		return ""
	}
	return ansiDim
}

// Report writes each finding then a summary footer.
func (h Human) Report(w io.Writer, findings []rule.Finding) error {
	for _, f := range findings {
		if _, err := fmt.Fprintf(
			w, "%s:%d:%d %s[%s]%s %s%s%s %s\n",
			f.Path, f.Line, f.Col,
			h.dim(), f.Rule, h.reset(),
			h.color(f.Severity), f.Severity, h.reset(),
			f.Message,
		); err != nil {
			return err
		}
	}
	errs, warns, infos := counts(findings)
	_, err := fmt.Fprintf(w, "\n%d error(s), %d warning(s), %d info\n", errs, warns, infos)
	return err
}
