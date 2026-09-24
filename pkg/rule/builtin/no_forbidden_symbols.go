package builtin

import (
	"fmt"
	"unicode/utf8"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

// NoForbiddenSymbols flags em dashes and middle dots in Markdown text.
type NoForbiddenSymbols struct{}

func (NoForbiddenSymbols) Meta() rule.Meta {
	return rule.Meta{
		Name:        "no-forbidden-symbols",
		Title:       "Forbidden symbols",
		Description: "avoid em dashes and middle dots in Markdown text",
		Detail: "The em dash (—) and middle dot (·) are forbidden in Markdown text, " +
			"including frontmatter. Each occurrence is reported separately. " +
			"Fenced and inline code are ignored because these characters can be " +
			"part of literal examples. Rewrite the surrounding sentence by hand; " +
			"there is no automatic replacement that preserves its meaning.",
		Severity: rule.Warning,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.NoFix,
		Example: rule.Example{
			Bad:  "First point — second point · third point",
			Good: "First point, second point, and third point",
		},
	}
}

func (r NoForbiddenSymbols) Check(doc *document.Document, report func(rule.Finding)) {
	for _, ln := range doc.Lines {
		if ln.InFence {
			continue
		}
		spans := codeSpanRanges(ln.Text)
		for i, symbol := range ln.Text {
			if symbol != '—' && symbol != '·' || insideCodeSpan(i, spans) {
				continue
			}
			report(rule.Finding{
				Rule:     r.Meta().Name,
				Path:     doc.Path,
				Line:     ln.Num,
				Col:      utf8.RuneCountInString(ln.Text[:i]) + 1,
				Message:  fmt.Sprintf("forbidden symbol %q", symbol),
				Severity: rule.Warning,
				Safety:   rule.NoFix,
			})
		}
	}
}
