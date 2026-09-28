package builtin

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
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
			"can be part of literal examples. So is markup, where these characters are " +
			"syntax: HTML tags and their attributes, the bodies of <style> and <script> " +
			"elements, Hugo shortcode tags, and the semicolon that closes an " +
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
	var markup markupState
	for _, ln := range doc.Lines {
		if ln.InFence {
			continue
		}
		spans := codeSpanRanges(ln.Text)
		entities := htmlEntity.FindAllStringIndex(ln.Text, -1)
		inMarkup := markup.mask(ln.Text)
		for i, symbol := range ln.Text {
			if !r.forbidden(symbol) || inMarkup[i] || insideCodeSpan(i, spans) || symbol == ';' && entityEnd(i, entities) {
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

// markupState follows, across lines, the markup whose characters are syntax
// rather than prose: an HTML tag with its attributes, the body of a <style> or
// <script> element, and a Hugo shortcode tag. Each may span several lines.
type markupState struct {
	inTag     bool   // inside <name ...>
	tagName   string // lowercase name of the tag being read ("/style" for a closer)
	shortcode string // closing delimiter (">}}" or "%}}") while inside a shortcode tag
	rawEnd    string // "</style" or "</script" while inside that element's body
}

// mask reports, per byte of text, whether it belongs to markup, and carries
// the state over to the next line.
func (s *markupState) mask(text string) []bool {
	m := make([]bool, len(text))
	lower := strings.ToLower(text)
	for i := 0; i < len(text); {
		var end int
		switch {
		case s.rawEnd != "":
			end = s.skipRaw(lower, i)
		case s.shortcode != "":
			end = s.skipShortcode(text, i)
		case s.inTag:
			end = s.skipTag(text, i)
		default:
			s.open(text, lower, i)
			if !s.inTag && s.shortcode == "" {
				i++
			}
			continue
		}
		for k := i; k < end; k++ {
			m[k] = true
		}
		i = end
	}
	return m
}

// skipRaw returns where the open <style> or <script> body stops in lower: at
// its closing tag, which is then read as a tag, or at the end of the line.
func (s *markupState) skipRaw(lower string, i int) int {
	j := strings.Index(lower[i:], s.rawEnd)
	if j < 0 {
		return len(lower)
	}
	s.rawEnd = ""
	return i + j
}

// skipShortcode returns the end of the open shortcode tag, or of the line.
func (s *markupState) skipShortcode(text string, i int) int {
	j := strings.Index(text[i:], s.shortcode)
	if j < 0 {
		return len(text)
	}
	end := i + j + len(s.shortcode)
	s.shortcode = ""
	return end
}

// skipTag returns the end of the open HTML tag, or of the line. Closing an
// opening <style> or <script> tag starts that element's raw body.
func (s *markupState) skipTag(text string, i int) int {
	j := strings.IndexByte(text[i:], '>')
	if j < 0 {
		return len(text)
	}
	end := i + j + 1
	s.inTag = false
	selfClosing := j > 0 && text[end-2] == '/'
	if (s.tagName == "style" || s.tagName == "script") && !selfClosing {
		s.rawEnd = "</" + s.tagName
	}
	return end
}

// open starts a shortcode or an HTML tag when one begins at text[i].
func (s *markupState) open(text, lower string, i int) {
	switch {
	case strings.HasPrefix(text[i:], "{{<"):
		s.shortcode = ">}}"
	case strings.HasPrefix(text[i:], "{{%"):
		s.shortcode = "%}}"
	case text[i] == '<' && i+1 < len(text) && tagStart(text[i+1]):
		s.inTag = true
		s.tagName = tagName(lower[i+1:])
	}
}

// tagStart reports whether c, right after '<', opens a tag (a name, a closer,
// or a comment/doctype) rather than a less-than sign in prose.
func tagStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '/' || c == '!'
}

// tagName returns the leading tag name of s, which starts right after '<'.
func tagName(s string) string {
	end := strings.IndexFunc(s, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '/' && r != '!'
	})
	if end < 0 {
		return s
	}
	return s[:end]
}
