package builtin

import (
	"fmt"
	"regexp"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

const (
	// indentedCodeIndent is the leading-whitespace width at which an ATX heading
	// stops being a (cosmetically) indented heading and is instead parsed as an
	// indented code block, losing the heading entirely (CommonMark: 4+ columns).
	indentedCodeIndent = 4
	// listMarkerSpace is the single space a list marker is followed by; together
	// with the marker width it gives the column where list-item content begins.
	listMarkerSpace = 1
	// bulletMarkerWidth is the width of a one-character bullet ("-", "*", "+").
	bulletMarkerWidth = 1
	// orderedDelimWidth is the width of an ordered-list delimiter ("." or ")").
	orderedDelimWidth = 1
)

// atxHeadingRe matches a genuine ATX heading once leading whitespace is removed:
// 1-6 '#' followed by a space, a tab, or end of line. A '#' glued to text
// ("#Heading") is deliberately NOT matched here — that is no-missing-space-atx's
// (MD018) concern, not an indentation defect.
var atxHeadingRe = regexp.MustCompile(`^#{1,6}([ \t]|$)`)

// HeadingStartLeft flags an ATX heading that does not start at the left margin
// (markdownlint MD023). 1-3 leading columns are merely cosmetic but still
// reported; 4+ columns turn the line into an indented code block so the heading
// is lost entirely. The fix dedents the heading to column 1 — but only when the
// heading is top-level: if the indentation is structural list nesting (the
// heading sits inside a list item), dedenting would de-nest it and change
// meaning, so the finding is emitted with no fix.
type HeadingStartLeft struct{}

func (HeadingStartLeft) Meta() rule.Meta {
	return rule.Meta{
		Name:        "heading-start-left",
		Description: "ATX headings should start at the left margin (no leading indentation)",
		Detail: "An ATX heading indented from the left margin is at best cosmetic " +
			"clutter and at worst a lost heading. With 1-3 leading columns the " +
			"heading still renders but markdownlint's MD023 flags the stray indent. " +
			"With 4+ leading columns CommonMark/Goldmark (the parser Hugo uses) " +
			"reparses the line as an indented code block, so the heading disappears " +
			"and renders as monospaced code text instead. The fix removes the " +
			"leading whitespace so the heading starts at column 1, and is idempotent " +
			"(a left-aligned heading no longer matches). The fix is withheld when the " +
			"heading is nested inside a list item, because there the indentation is " +
			"structural — dedenting would pull the heading out of the list.",
		Severity: rule.Warning,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.Safe,
	}
}

func (r HeadingStartLeft) Check(doc *document.Document, report func(rule.Finding)) {
	lines := doc.Lines
	for i, ln := range lines {
		// Fence-aware and frontmatter-aware: skip fenced code and any line that
		// begins inside the frontmatter block (before the body starts).
		if ln.InFence || ln.Start < doc.BodyOffset {
			continue
		}
		w := leadingWhitespace(ln.Text)
		if w == 0 || !atxHeadingRe.MatchString(ln.Text[w:]) {
			continue
		}
		report(r.finding(doc, lines, i, w))
	}
}

// finding builds the Finding for an indented heading at line index i with
// leading-whitespace width w. The Safe dedent fix is attached only when the
// heading is top-level; a list-nested heading is reported with no fix.
func (r HeadingStartLeft) finding(doc *document.Document, lines []document.Line, i, w int) rule.Finding {
	ln := lines[i]
	f := rule.Finding{
		Rule:     r.Meta().Name,
		Path:     doc.Path,
		Line:     ln.Num,
		Col:      1,
		Message:  indentMessage(w),
		Severity: rule.Warning,
		Safety:   rule.NoFix,
	}
	if !nestedInList(lines, i, w, doc.BodyOffset) {
		f.Safety = rule.Safe
		f.Fixes = []rule.TextEdit{{Start: ln.Start, End: ln.Start + w, NewText: ""}}
	}
	return f
}

// indentMessage describes the defect, escalating when the indent is deep enough
// to demote the heading into an indented code block.
func indentMessage(w int) string {
	if w >= indentedCodeIndent {
		return fmt.Sprintf("heading indented %d columns becomes an indented code block "+
			"and is lost; start it at the left margin", w)
	}
	return fmt.Sprintf("heading indented %d column(s); ATX headings should start at the left margin", w)
}

// leadingWhitespace returns the number of leading space/tab bytes in text.
func leadingWhitespace(text string) int {
	i := 0
	for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
		i++
	}
	return i
}

// nestedInList reports whether the heading at line index idx (indented by w)
// sits inside a list item, by scanning upward for an open list-item marker whose
// content column is at most w. Frontmatter lines (before bodyOffset) and a
// root-level (unindented) non-list block both end the scan as "top-level".
func nestedInList(lines []document.Line, idx, w, bodyOffset int) bool {
	for j := idx - 1; j >= 0; j-- {
		ln := lines[j]
		if ln.Start < bodyOffset {
			return false // reached frontmatter: the heading is top-level
		}
		if ln.InFence || isBlank(ln.Text) {
			continue
		}
		if ci, ok := listContentIndent(ln.Text); ok {
			if ci <= w {
				return true // structural list nesting
			}
			continue // a more-indented marker; keep looking for a shallower one
		}
		if leadingWhitespace(ln.Text) == 0 {
			return false // a root-level non-list block resets list context
		}
	}
	return false
}

// listContentIndent returns the column where a list item's content begins and
// whether text is a list-item marker line at all (unordered "- ", "* ", "+ " or
// ordered "1. ", "2) "). A marker must be followed by a space/tab or end the
// line (an empty item); otherwise the line is ordinary text.
func listContentIndent(text string) (int, bool) {
	indent := leadingWhitespace(text)
	if indent >= len(text) {
		return 0, false
	}
	switch c := text[indent]; {
	case c == '-' || c == '*' || c == '+':
		if markerTerminated(text, indent+bulletMarkerWidth) {
			return indent + bulletMarkerWidth + listMarkerSpace, true
		}
	case c >= '0' && c <= '9':
		end := indent
		for end < len(text) && text[end] >= '0' && text[end] <= '9' {
			end++
		}
		if end < len(text) && (text[end] == '.' || text[end] == ')') &&
			markerTerminated(text, end+orderedDelimWidth) {
			return end + orderedDelimWidth + listMarkerSpace, true
		}
	}
	return 0, false
}

// markerTerminated reports whether position k ends a list marker: either the
// line ends there (an empty item) or a space/tab separates marker from content.
func markerTerminated(text string, k int) bool {
	return k >= len(text) || text[k] == ' ' || text[k] == '\t'
}

// isBlank reports whether text is empty or only whitespace.
func isBlank(text string) bool {
	return leadingWhitespace(text) == len(text)
}
