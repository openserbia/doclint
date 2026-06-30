package builtin

import (
	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

// BlanksAroundThematicBreak flags a thematic break (---, ***, ___ or their
// spaced variants such as "- - -") that is not surrounded by blank lines
// (markdownlint MD047-adjacent). Without the surrounding blanks some renderers
// fail to parse the marker as a block-level separator; the blank also prevents
// "---" from being re-read as a setext heading underline when content appears
// immediately before it. The fix inserts a single blank line on the offending
// side — content-neutral and idempotent. The document's first and last lines
// are exempt (a thematic break at the very start or end needs no surrounding
// blank). Setext heading underlines are excluded: a "---" line that immediately
// follows ordinary paragraph text is already handled by BlanksAroundHeadings.
// Exception: Hugo shortcode lines ({{< … >}} / {{% … %}}) and CommonMark
// attribute blocks ({.class}) look like paragraph text to a line-based scanner
// but are block-level constructs in Goldmark; a "---" following them is still a
// thematic break and is not exempt from this rule.
type BlanksAroundThematicBreak struct{}

func (BlanksAroundThematicBreak) Meta() rule.Meta {
	return rule.Meta{
		Name:        "blanks-around-thematic-break",
		Title:       "Blank lines around thematic breaks",
		Description: "thematic breaks (--- / *** / ___) should be surrounded by blank lines",
		Detail: "A thematic break (a line of three or more '-', '*', or '_' markers, " +
			"optionally spaced) needs a blank line above and below it. Without the " +
			"surrounding blank the marker can be re-parsed as a setext heading underline " +
			"(when '---' follows paragraph text) or silently merged with an adjacent " +
			"paragraph by some renderers. This rule reports each thematic break that is " +
			"missing a surrounding blank line and inserts one (a safe, idempotent fix). " +
			"The document's first and last lines are exempt. A '---' line that is a " +
			"genuine setext heading underline (i.e. it directly follows plain paragraph " +
			"text) is skipped — BlanksAroundHeadings handles it. Hugo shortcode lines " +
			"({{< … >}}) and CommonMark attribute blocks before a '---' are treated as " +
			"block-level constructs, not paragraph text, so the '---' is still flagged " +
			"as a thematic break.",
		Severity: rule.Warning,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.Safe,
		Example: rule.Example{
			Bad: `text
---
more`,
			Good: `text

---

more`,
		},
	}
}

func (r BlanksAroundThematicBreak) Check(doc *document.Document, report func(rule.Finding)) {
	lines := doc.Lines
	for i, ln := range lines {
		if ln.InFence || ln.Start < doc.BodyOffset {
			continue
		}
		if !isThematicBreakLine(doc, lines, i) {
			continue
		}
		if i > 0 {
			prev := lines[i-1]
			if prev.Start >= doc.BodyOffset && !isBlank(prev.Text) {
				report(r.finding(doc, ln, "missing blank line before thematic break",
					rule.TextEdit{Start: ln.Start, End: ln.Start, NewText: "\n"}))
			}
		}
		if i+1 < len(lines) && !isBlank(lines[i+1].Text) {
			report(r.finding(doc, ln, "missing blank line after thematic break",
				rule.TextEdit{Start: ln.End, End: ln.End, NewText: "\n"}))
		}
	}
}

// isThematicBreakLine reports whether the line at index i is a genuine thematic
// break. A "---" line that directly follows ordinary paragraph text is a setext
// heading underline and returns false (BlanksAroundHeadings handles it). Hugo
// shortcode and CommonMark attribute-block lines are block-level constructs even
// though isSetextText considers them paragraph-like; a "---" after them is still
// a thematic break. Lines using "***" or "___" can never be setext underlines and
// always return true when the other conditions are met.
func isThematicBreakLine(doc *document.Document, lines []document.Line, i int) bool {
	ln := lines[i]
	if !thematicBreakRe.MatchString(ln.Text) {
		return false
	}
	// Only "---"-style lines (those matching underlineRe) can double as setext
	// heading underlines. "***" and "___" are always thematic breaks.
	if underlineRe.MatchString(ln.Text) && i > 0 {
		prev := lines[i-1]
		if !prev.InFence && prev.Start >= doc.BodyOffset &&
			isSetextText(prev.Text) && !blockConstructRe.MatchString(prev.Text) {
			return false // setext heading underline; BlanksAroundHeadings handles it
		}
	}
	return true
}

// finding assembles a Warning finding carrying a single safe blank-insertion fix
// for the thematic break line ln.
func (r BlanksAroundThematicBreak) finding(doc *document.Document, ln document.Line, msg string, edit rule.TextEdit) rule.Finding {
	return rule.Finding{
		Rule:     r.Meta().Name,
		Path:     doc.Path,
		Line:     ln.Num,
		Col:      1,
		Message:  msg,
		Severity: rule.Warning,
		Safety:   rule.Safe,
		Fixes:    []rule.TextEdit{edit},
	}
}
