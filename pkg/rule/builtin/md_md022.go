package builtin

import (
	"regexp"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

// underlineRe matches a setext heading underline once up to three leading spaces
// are allowed: a run of '=' (level 1) or '-' (level 2) with optional trailing
// whitespace and nothing else. It is the same shape CommonMark/Goldmark require,
// and it deliberately also matches "---"/"===", which is how a thematic-break or
// underline-looking line is recognized and excluded as setext heading text.
var underlineRe = regexp.MustCompile(`^ {0,3}(=+|-+)[ \t]*$`)

// thematicBreakRe matches a thematic break once up to three leading spaces are
// allowed: three or more of the same marker ('*', '-' or '_'), optionally spaced.
// Such a line is not paragraph text, so it cannot be the text of a setext heading.
var thematicBreakRe = regexp.MustCompile(`^ {0,3}((\*[ \t]*){3,}|(-[ \t]*){3,}|(_[ \t]*){3,})$`)

// BlanksAroundHeadings flags an ATX or setext heading that is not surrounded by a
// blank line above and below (markdownlint MD022). The surrounding blank is
// mostly structural hygiene, but a setext underline (and some list adjacencies)
// only parses as a heading when the blank is present. The fix inserts a single
// blank line on the offending side — content-neutral and idempotent. Severity is
// Warning so a boundary heuristic miss never hard-blocks a deploy.
//
// The above-blank insertion is anchored at the END of the line above the heading
// (textually identical to inserting at the heading's start, since the two offsets
// bracket the same newline) so it never coincides with heading-start-left's
// dedent edit at the heading's start — keeping the shared fmt pass free of a
// spurious ApplyEdits overlap. The setext above-check fires only when the line
// above is a structural block boundary (another heading, a fence, a list item, a
// thematic break); a plain paragraph line above is left alone, because a setext
// heading's text may span several lines and splitting it would change the heading.
type BlanksAroundHeadings struct{}

func (BlanksAroundHeadings) Meta() rule.Meta {
	return rule.Meta{
		Name:        "blanks-around-headings",
		Description: "headings should be surrounded by blank lines",
		Detail: "An ATX heading (\"# Heading\") or setext heading (a line of text " +
			"underlined by \"===\" or \"---\") should have a blank line both above " +
			"and below it. The surrounding blank is largely structural hygiene, but a " +
			"setext underline only parses as a heading when the text line above it is " +
			"a paragraph, and some list adjacencies likewise need the blank to render " +
			"as a heading at all. This rule reports each missing surrounding blank and " +
			"inserts one (a safe, idempotent, content-neutral fix). Fenced code and " +
			"frontmatter are skipped (so a YAML \"---\" is never mistaken for a setext " +
			"underline or thematic break), and the document's first and last lines are " +
			"exempt. The setext above-check is withheld when the line above is ordinary " +
			"paragraph text, since a setext heading's text can span multiple lines and " +
			"inserting a blank there would split the heading. The defect is a Warning: " +
			"heading boundaries are detected line-by-line, and a Warning keeps a rare " +
			"mis-detection from hard-blocking a deploy.",
		Severity: rule.Warning,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.Safe,
	}
}

func (r BlanksAroundHeadings) Check(doc *document.Document, report func(rule.Finding)) {
	lines := doc.Lines
	i := 0
	for i < len(lines) {
		ln := lines[i]
		// Fence-aware and frontmatter-aware: skip fenced code and any line that
		// begins inside the frontmatter block (before the body starts).
		if ln.InFence || ln.Start < doc.BodyOffset {
			i++
			continue
		}
		if isATXHeading(ln.Text) {
			r.checkATX(doc, lines, i, report)
			i++
			continue
		}
		if i+1 < len(lines) && isSetextHeading(doc, lines, i) {
			r.checkSetext(doc, lines, i, report)
			i += 2 // skip the underline so it is never re-read as heading text
			continue
		}
		i++
	}
}

// checkATX reports the missing blank above and/or below the ATX heading at index
// i. The line above the heading being the document start or frontmatter is exempt,
// as is a blank neighbour. Inserting a blank around an ATX heading never changes
// how the heading itself parses, so any non-blank neighbour is flagged.
func (r BlanksAroundHeadings) checkATX(doc *document.Document, lines []document.Line, i int, report func(rule.Finding)) {
	ln := lines[i]
	// A heading structurally nested in a list item is left alone: inserting a
	// blank would de-nest it (the same reason heading-start-left withholds its
	// dedent there).
	if w := leadingWhitespace(ln.Text); w > 0 && nestedInList(lines, i, w, doc.BodyOffset) {
		return
	}
	if i > 0 {
		prev := lines[i-1]
		if prev.Start >= doc.BodyOffset && !isBlank(prev.Text) {
			report(r.finding(doc, ln.Num, "missing blank line above heading",
				rule.TextEdit{Start: prev.End, End: prev.End, NewText: "\n"}))
		}
	}
	if i+1 < len(lines) && !isBlank(lines[i+1].Text) {
		report(r.finding(doc, ln.Num, "missing blank line below heading",
			rule.TextEdit{Start: ln.End, End: ln.End, NewText: "\n"}))
	}
}

// checkSetext reports the missing blank above the text line and/or below the
// underline of the setext heading whose text is at index i (and underline at
// i+1). The above-check fires only at a structural block boundary (see the type
// doc) so a multi-line setext paragraph is never split.
func (r BlanksAroundHeadings) checkSetext(doc *document.Document, lines []document.Line, i int, report func(rule.Finding)) {
	t := lines[i]
	underline := lines[i+1]
	// A setext heading nested in a list item is left alone for the same de-nesting
	// reason as the ATX case.
	if w := leadingWhitespace(t.Text); w > 0 && nestedInList(lines, i, w, doc.BodyOffset) {
		return
	}
	if i > 0 {
		prev := lines[i-1]
		if prev.Start >= doc.BodyOffset && !prev.InFence && isStructuralBoundary(prev.Text) {
			report(r.finding(doc, t.Num, "missing blank line above heading",
				rule.TextEdit{Start: prev.End, End: prev.End, NewText: "\n"}))
		}
	}
	if i+2 < len(lines) && !isBlank(lines[i+2].Text) {
		report(r.finding(doc, t.Num, "missing blank line below heading",
			rule.TextEdit{Start: underline.End, End: underline.End, NewText: "\n"}))
	}
}

// finding assembles a Warning finding carrying a single safe blank-insertion fix.
func (r BlanksAroundHeadings) finding(doc *document.Document, lineNum int, msg string, edit rule.TextEdit) rule.Finding {
	return rule.Finding{
		Rule:     r.Meta().Name,
		Path:     doc.Path,
		Line:     lineNum,
		Col:      1,
		Message:  msg,
		Severity: rule.Warning,
		Safety:   rule.Safe,
		Fixes:    []rule.TextEdit{edit},
	}
}

// isATXHeading reports whether text is an ATX heading: up to three leading spaces,
// then 1-6 '#' followed by a space, a tab, or end of line. A glued "#Heading" (no
// separator) is deliberately not a heading here — that is no-missing-space-atx's
// concern — and 7+ '#' is not a heading at all.
func isATXHeading(text string) bool {
	w := leadingWhitespace(text)
	if w > maxATXIndent {
		return false
	}
	return atxHeadingRe.MatchString(text[w:])
}

// isSetextHeading reports whether the line at index i is the text of a setext
// heading underlined by the next line: the underline must be a body line outside
// any fence that matches underlineRe, and the text line must read as a paragraph.
func isSetextHeading(doc *document.Document, lines []document.Line, i int) bool {
	underline := lines[i+1]
	if underline.InFence || underline.Start < doc.BodyOffset {
		return false
	}
	if !underlineRe.MatchString(underline.Text) {
		return false
	}
	return isSetextText(lines[i].Text)
}

// isSetextText reports whether text can be the text line of a setext heading: a
// non-blank paragraph line that is not itself an ATX heading, a list item, a fence
// delimiter, a thematic break, or an underline-looking line. (A '-' underline only
// forms a heading when the line above reads as a paragraph; otherwise the '-' line
// is a thematic break — these exclusions enforce exactly that.)
func isSetextText(text string) bool {
	if isBlank(text) {
		return false
	}
	if isATXHeading(text) || isListItem(text) {
		return false
	}
	return !isUnderlineOrBreak(text) && !fenceDelimRe.MatchString(text)
}

// isStructuralBoundary reports whether text is a non-blank block that cannot be a
// paragraph continuation: another heading, a list item, a fence delimiter, a
// thematic break, or an underline-looking line. Such a line directly above a
// setext heading means the heading's text is a single line, so a blank may safely
// be inserted between them.
func isStructuralBoundary(text string) bool {
	if isBlank(text) {
		return false
	}
	if isATXHeading(text) || isListItem(text) {
		return true
	}
	return isUnderlineOrBreak(text) || fenceDelimRe.MatchString(text)
}

// isUnderlineOrBreak reports whether text is a setext underline or a thematic
// break (a "===", "---", "***" or "___" style rule line).
func isUnderlineOrBreak(text string) bool {
	return underlineRe.MatchString(text) || thematicBreakRe.MatchString(text)
}

// isListItem reports whether text begins a list item.
func isListItem(text string) bool {
	_, ok := listContentIndent(text)
	return ok
}
