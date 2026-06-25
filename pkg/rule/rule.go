// Package rule defines the Rule interface, severity/fix-safety vocabulary, and
// the Finding/TextEdit types every rule emits. Fixes are first-class byte-offset
// edits so lint --fix, fmt, and --diff all consume the same data.
package rule

import (
	"fmt"

	"github.com/openserbia/doclint/pkg/document"
)

// Severity orders findings from advisory to blocking.
type Severity int

const (
	Info Severity = iota
	Warning
	Error
)

func (s Severity) String() string {
	switch s {
	case Info:
		return "info"
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

// ParseSeverity converts a config string into a Severity.
func ParseSeverity(s string) (Severity, error) {
	switch s {
	case "info":
		return Info, nil
	case "warning":
		return Warning, nil
	case "error":
		return Error, nil
	default:
		return Info, fmt.Errorf("invalid severity %q", s)
	}
}

// FixSafety describes whether a fix preserves meaning.
type FixSafety int

const (
	NoFix  FixSafety = iota // no automatic fix
	Safe                    // applied by --fix and fmt
	Unsafe                  // applied only with --unsafe-fixes
)

// TextEdit replaces Raw[Start:End] with NewText. Offsets index Document.Raw.
type TextEdit struct {
	Start   int
	End     int
	NewText string
}

// Meta is a rule's static descriptor.
type Meta struct {
	Name        string
	Description string            // one line, shown by `list`
	Detail      string            // long help, shown by `explain`
	Severity    Severity          // default; config may override
	Formats     []document.Format // which formats this rule applies to
	Safety      FixSafety         // safety of fixes this rule emits
}

// AppliesTo reports whether the rule runs on the given format.
func (m Meta) AppliesTo(f document.Format) bool {
	for _, x := range m.Formats {
		if x == f {
			return true
		}
	}
	return false
}

// Finding is one reported issue.
type Finding struct {
	Rule     string     `json:"rule"`
	Path     string     `json:"path"`
	Line     int        `json:"line"`
	Col      int        `json:"col"`
	Message  string     `json:"message"`
	Severity Severity   `json:"severity"`
	Safety   FixSafety  `json:"-"`
	Fixes    []TextEdit `json:"-"`
}

// Rule inspects a Document and reports findings.
type Rule interface {
	Meta() Meta
	Check(doc *document.Document, report func(Finding))
}
