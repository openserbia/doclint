package builtin

import (
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

// NoForbiddenSymbols flags em dashes, middle dots and semicolons in Markdown text.
type NoForbiddenSymbols struct{}

// htmlEntity matches a character reference such as &nbsp; or &#8212;, whose
// closing semicolon is syntax, not punctuation.
var htmlEntity = regexp.MustCompile(`&(?:[A-Za-z][A-Za-z0-9]*|#\d+|#[xX][0-9A-Fa-f]+);`)

func forbiddenSymbol(r rune) bool {
	return r == '—' || r == '·' || r == ';'
}

func (NoForbiddenSymbols) Meta() rule.Meta {
	return rule.Meta{
		Name:        "no-forbidden-symbols",
		Title:       "Forbidden symbols",
		Description: "avoid em dashes, middle dots and semicolons in Markdown text",
		Detail: "The em dash (—), middle dot (·) and semicolon (;) are forbidden in " +
			"Markdown text, including frontmatter. Each occurrence is reported " +
			"separately. Fenced and inline code are ignored because these characters " +
			"can be part of literal examples, and so is the semicolon that closes an " +
			"HTML character reference such as `&nbsp;`. Rewrite the surrounding " +
			"sentence by hand: there is no automatic replacement that preserves its meaning.",
		Severity: rule.Warning,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.NoFix,
		Example: rule.Example{
			Bad:  "First point — second point · third point; fourth point",
			Good: "First point, second point, third point, and fourth point",
		},
	}
}

func (r NoForbiddenSymbols) Check(doc *document.Document, report func(rule.Finding)) {
	for _, ln := range doc.Lines {
		if ln.InFence {
			continue
		}
		spans := codeSpanRanges(ln.Text)
		entities := htmlEntity.FindAllStringIndex(ln.Text, -1)
		for i, symbol := range ln.Text {
			if !forbiddenSymbol(symbol) || insideCodeSpan(i, spans) || symbol == ';' && entityEnd(i, entities) {
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

// entityEnd reports whether byte offset i is the closing semicolon of one of
// the character references in entities.
func entityEnd(i int, entities [][]int) bool {
	for _, e := range entities {
		if i == e[1]-1 {
			return true
		}
	}
	return false
}
