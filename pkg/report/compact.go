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

// Compact renders one flat `path:line:col [rule] severity message` line per
// finding. It is the guaranteed clickable / pipe-friendly format and the
// automatic fallback when stdout is not a terminal.
type Compact struct{ NoColor bool }

func (c Compact) color(s rule.Severity) string {
	if c.NoColor {
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

func (c Compact) reset() string {
	if c.NoColor {
		return ""
	}
	return ansiReset
}

func (c Compact) dim() string {
	if c.NoColor {
		return ""
	}
	return ansiDim
}

// Report writes each finding then a summary footer.
func (c Compact) Report(w io.Writer, findings []rule.Finding) error {
	for _, f := range findings {
		if _, err := fmt.Fprintf(
			w, "%s:%d:%d %s[%s]%s %s%s%s %s\n",
			f.Path, f.Line, f.Col,
			c.dim(), f.Rule, c.reset(),
			c.color(f.Severity), f.Severity, c.reset(),
			f.Message,
		); err != nil {
			return err
		}
	}
	errs, warns, infos := counts(findings)
	if _, err := fmt.Fprintf(w, "\n%d error(s), %d warning(s), %d info\n", errs, warns, infos); err != nil {
		return err
	}
	safe, unsafe := fixCounts(findings)
	if safe > 0 || unsafe > 0 {
		msg := fmt.Sprintf("%d fixable with --fix", safe)
		if unsafe > 0 {
			msg += fmt.Sprintf(", %d with --unsafe-fixes", unsafe)
		}
		if _, err := fmt.Fprintln(w, msg); err != nil {
			return err
		}
	}
	return nil
}
