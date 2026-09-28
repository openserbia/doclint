package builtin

import (
	"fmt"
	"regexp"
	"slices"
	"unicode/utf8"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

// NoForbiddenSymbols flags em dashes, middle dots and semicolons in Markdown text,
// or the characters listed under settings.no-forbidden-symbols.symbols.
type NoForbiddenSymbols struct {
	symbols []rune // nil: defaultForbiddenSymbols
}

// defaultForbiddenSymbols is the list used when the config sets none.
var defaultForbiddenSymbols = []rune{'—', '·', ';'}

// htmlEntity matches a character reference such as &nbsp; or &#8212;, whose
// closing semicolon is syntax, not punctuation.
var htmlEntity = regexp.MustCompile(`&(?:[A-Za-z][A-Za-z0-9]*|#\d+|#[xX][0-9A-Fa-f]+);`)

// WithOptions returns the rule with the configured symbol list. Each entry must
// be exactly one character. An empty list flags nothing.
func (NoForbiddenSymbols) WithOptions(o rule.Options) (rule.Rule, error) {
	symbols := make([]rune, 0, len(o.Symbols))
	for _, s := range o.Symbols {
		if utf8.RuneCountInString(s) != 1 {
			return nil, fmt.Errorf("symbols: %q is not a single character", s)
		}
		r, _ := utf8.DecodeRuneInString(s)
		symbols = append(symbols, r)
	}
	return NoForbiddenSymbols{symbols: symbols}, nil
}

func (r NoForbiddenSymbols) forbidden(c rune) bool {
	symbols := r.symbols
	if symbols == nil {
		symbols = defaultForbiddenSymbols
	}
	return slices.Contains(symbols, c)
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
			"sentence by hand: there is no automatic replacement that preserves its meaning. " +
			"To use a different list, set `symbols` under `settings.no-forbidden-symbols` " +
			"in .doclint.yaml, one character per entry; it replaces the default list.",
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
			if !r.forbidden(symbol) || insideCodeSpan(i, spans) || symbol == ';' && entityEnd(i, entities) {
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
