package builtin

import (
	"regexp"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

// listItemRe matches a list-item line per CommonMark's marker grammar once up to
// three leading spaces are allowed: a bullet (-, *, +) or an ordered marker (1-9
// digits followed by '.' or ')'), in turn followed by a space, a tab, or end of
// line (an empty item). Four or more leading spaces make the line an indented
// code block, so they are excluded. It is matched only against non-fenced lines,
// so a "- x" written inside a code fence never registers as a list item.
var listItemRe = regexp.MustCompile(`^ {0,3}([-*+]|\d{1,9}[.)])([ \t]|$)`)

// BlanksAroundLists flags a list block that is not preceded and followed by a
// blank line (markdownlint MD032). A list line butted directly under a paragraph
// can be swallowed as a lazy paragraph continuation (the list never renders), and
// a paragraph butted directly under the last item can be absorbed into that item.
// The fix inserts a single blank line on the offending side — content-neutral and
// idempotent. Severity is Warning so a boundary mis-detection (the line-based
// region heuristic cannot see every CommonMark lazy-continuation nuance) never
// hard-blocks a deploy.
type BlanksAroundLists struct{}

func (BlanksAroundLists) Meta() rule.Meta {
	return rule.Meta{
		Name:        "blanks-around-lists",
		Description: "lists should be surrounded by blank lines",
		Detail: "A list block needs a blank line before its first item and after its " +
			"last item. When a list line is butted directly beneath a paragraph, " +
			"CommonMark/Goldmark (the parser Hugo uses) folds it into that paragraph " +
			"as a lazy continuation line and no list renders; when a paragraph is " +
			"butted directly beneath the last item, it is absorbed into that item. " +
			"This rule finds each maximal list region — a run of list-item lines, " +
			"their indented continuation lines, and the single blank lines between " +
			"items of a loose list — and reports a missing blank line on either edge, " +
			"inserting one (a safe, idempotent fix). Frontmatter and fenced code are " +
			"skipped. The defect is a Warning: the region boundaries are detected " +
			"line-by-line, so an unusual lazy continuation could be mis-attributed, " +
			"and a Warning keeps that from hard-blocking a deploy.",
		Severity: rule.Warning,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.Safe,
	}
}

func (r BlanksAroundLists) Check(doc *document.Document, report func(rule.Finding)) {
	lines := doc.Lines
	i := 0
	for i < len(lines) {
		ci, ok := listStart(doc, lines[i])
		if !ok {
			i++
			continue
		}
		e := regionEnd(lines, i, ci)
		r.checkBefore(doc, lines, i, report)
		r.checkAfter(doc, lines, e, report)
		i = e + 1
	}
}

// listStart reports whether ln begins a list region: it must be a body line
// (outside frontmatter) that is not inside a fenced code block and matches the
// list-item grammar. On success it returns the column where the item's content
// begins.
func listStart(doc *document.Document, ln document.Line) (int, bool) {
	if ln.InFence || ln.Start < doc.BodyOffset {
		return 0, false
	}
	return listItemContent(ln.Text)
}

// listItemContent reports whether text is a list-item line and, if so, the column
// at which the item's content begins (the marker end plus one separator). The
// content column is what continuation lines must be indented to.
func listItemContent(text string) (int, bool) {
	loc := listItemRe.FindStringSubmatchIndex(text)
	if loc == nil {
		return 0, false
	}
	// loc[markerEndIdx] is the byte offset just past the marker submatch; the item
	// content begins one separator column further right.
	const markerEndIdx = 3
	return loc[markerEndIdx] + listMarkerSpace, true
}

// regionEnd returns the index of the last line belonging to the list region that
// begins at index s (whose items' content starts at column ci). The region runs
// over further list-item lines, continuation lines indented to at least ci, and
// single interior blank lines that are themselves followed by more list content.
func regionEnd(lines []document.Line, s, ci int) int {
	e := s
	for j := s + 1; j < len(lines); j++ {
		ln := lines[j]
		if isBlank(ln.Text) {
			if !interiorBlank(lines, j, ci) {
				break
			}
			continue
		}
		if !inList(ln, ci) {
			break
		}
		e = j
	}
	return e
}

// interiorBlank reports whether the blank line at index j sits between two
// members of the same loose list: the next line must exist, be non-blank (a
// second consecutive blank ends the list), and itself belong to the list.
func interiorBlank(lines []document.Line, j, ci int) bool {
	if j+1 >= len(lines) {
		return false
	}
	next := lines[j+1]
	if isBlank(next.Text) {
		return false
	}
	return inList(next, ci)
}

// inList reports whether the non-blank line ln is part of a list region whose
// items' content begins at column ci: a (possibly nested) list-item line, or a
// line indented to at least ci (an item continuation, including indented fenced
// content). A fenced line is never read as a new list item.
func inList(ln document.Line, ci int) bool {
	if !ln.InFence {
		if _, ok := listItemContent(ln.Text); ok {
			return true
		}
	}
	return leadingWhitespace(ln.Text) >= ci
}

// checkBefore reports a missing blank line before the region whose first item is
// at index s. The document's first body line is exempt, as is a region whose
// preceding line is blank, frontmatter, or itself a list-item line.
func (r BlanksAroundLists) checkBefore(doc *document.Document, lines []document.Line, s int, report func(rule.Finding)) {
	if s == 0 {
		return
	}
	prev := lines[s-1]
	if prev.Start < doc.BodyOffset || isBlank(prev.Text) {
		return
	}
	if _, ok := listItemContent(prev.Text); ok {
		return
	}
	ln := lines[s]
	report(r.finding(doc, ln, "missing blank line before list",
		rule.TextEdit{Start: ln.Start, End: ln.Start, NewText: "\n"}))
}

// checkAfter reports a missing blank line after the region whose last line is at
// index e. The region's last line being the document's last line is exempt, as is
// a following blank line. regionEnd guarantees a present, non-blank following line
// does not belong to the list, so it is a paragraph that the final item absorbs.
func (r BlanksAroundLists) checkAfter(doc *document.Document, lines []document.Line, e int, report func(rule.Finding)) {
	if e+1 >= len(lines) || isBlank(lines[e+1].Text) {
		return
	}
	ln := lines[e]
	report(r.finding(doc, ln, "missing blank line after list",
		rule.TextEdit{Start: ln.End, End: ln.End, NewText: "\n"}))
}

// finding assembles a Warning finding carrying a single safe blank-insertion fix
// for the list-edge line ln.
func (r BlanksAroundLists) finding(doc *document.Document, ln document.Line, msg string, edit rule.TextEdit) rule.Finding {
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
