package builtin

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

// listMarkerRe matches a list-item marker (bullet or ordered) with up to three
// leading spaces, capturing the leading whitespace and the marker so the content
// column — where the item's body must indent to — can be computed.
var listMarkerRe = regexp.MustCompile(`^( {0,3})([-*+]|\d{1,9}[.)]) `)

// ListMarkerIndent flags a list item whose body is indented to fewer columns than
// the marker requires, which makes the nested content escape the item and (for an
// ordered list) restarts the numbering.
type ListMarkerIndent struct{}

func (ListMarkerIndent) Meta() rule.Meta {
	return rule.Meta{
		Name:        "list-marker-indent",
		Title:       "List item body indentation",
		Description: "list item bodies must indent to the marker's content column",
		Detail: "A list item's continuation and nested content must be indented to " +
			"the marker's content column — len(marker)+1: 2 spaces under a \"- \" " +
			"bullet, 3 under \"1.\"–\"9.\", 4 under \"10.\"+. When the body is indented " +
			"less than that (a common foot-gun is a 2-space body under a single-digit " +
			"\"1. \" item, which needs 3), CommonMark/Goldmark does not attach it to the " +
			"item: the nested list escapes, an ordered list splits into single-item " +
			"lists, and the numbering restarts (1. 1. 1. instead of 1. 2. 3.). The " +
			"Unsafe fix (--fix --unsafe-fixes) re-indents the body to the content " +
			"column: content outside the nested list (a leading paragraph, or a " +
			"closing shortcode that de-indented out of it) is set to the content " +
			"column, and the nested list block is shifted as a whole so its relative " +
			"nesting is preserved. Separating the two is what stops an already-correct " +
			"line from being over-indented (the v0.5.0 uniform shift's bug); it stays " +
			"Unsafe because an inconsistent body can warrant a human's eye on the diff.",
		Severity: rule.Warning,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.Unsafe,
		Example: rule.Example{
			Bad: `1. {{< details "Doc" >}}
  - body under-indented (2 spaces under a "1. " item)
  {{< /details >}}
1. Next item — restarts, rendering "1." again`,
			Good: `1. {{< details "Doc" >}}
   - body at the content column (3 spaces)
   {{< /details >}}
1. Next item — renders as "2."`,
		},
	}
}

func (r ListMarkerIndent) Check(doc *document.Document, report func(rule.Finding)) {
	lines := doc.Lines
	i := 0
	for i < len(lines) {
		ln := lines[i]
		m := listMarkerRe.FindStringSubmatch(ln.Text)
		if m == nil || ln.InFence || ln.Start < doc.BodyOffset {
			i++
			continue
		}
		markerIndent := len(m[1])
		contentCol := markerIndent + len(m[2]) + 1
		end, base := bodyExtent(lines, i, markerIndent)
		if base > markerIndent && base < contentCol {
			r.flag(doc, lines, i, end, contentCol, contentCol-base, report)
		}
		i = end + 1
	}
}

// bodyExtent returns the index of the last body line of the item starting at i
// (the following lines indented strictly more than markerIndent, with single
// interior blank lines kept) and the smallest indent among its non-blank body
// lines (markerIndent when the item has no body).
func bodyExtent(lines []document.Line, i, markerIndent int) (end, base int) {
	end, base = i, -1
	for j := i + 1; j < len(lines); j++ {
		ln := lines[j]
		if isBlank(ln.Text) {
			if j+1 < len(lines) && !isBlank(lines[j+1].Text) && leadingWhitespace(lines[j+1].Text) > markerIndent {
				continue // interior blank inside the body
			}
			break
		}
		if leadingWhitespace(ln.Text) <= markerIndent {
			break // a sibling item or a de-indent ends the body
		}
		if ind := leadingWhitespace(ln.Text); base < 0 || ind < base {
			base = ind
		}
		end = j
	}
	if base < 0 {
		base = markerIndent
	}
	return end, base
}

// flag reports the under-indented item and an Unsafe fix that re-indents its body
// to the content column. Leading content (before the first nested list item) is
// set to the content column; the list block from the first list item onward is
// shifted by one delta so its relative nesting is preserved. Splitting the two is
// what keeps an already-correct leading paragraph from being over-indented.
// flag reports the under-indented item with an Unsafe fix from bodyReindentFixes.
func (r ListMarkerIndent) flag(doc *document.Document, lines []document.Line, start, end, contentCol, delta int, report func(rule.Finding)) {
	report(rule.Finding{
		Rule:     "list-marker-indent",
		Path:     doc.Path,
		Line:     lines[start].Num,
		Col:      1,
		Message:  fmt.Sprintf("list item body is under-indented; indent it %d more space(s) to the marker's content column", delta),
		Severity: rule.Warning,
		Safety:   rule.Unsafe,
		Fixes:    bodyReindentFixes(lines, start, end, contentCol),
	})
}

// bodyReindentFixes re-indents an item body (lines after start, through end) to
// contentCol. Content outside the nested list — a leading paragraph, or a trailing
// closing shortcode that de-indented out of it — is set to contentCol; the list
// block is shifted as a whole so its relative nesting is preserved. Separating the
// two keeps an already-correct line from being over-indented.
func bodyReindentFixes(lines []document.Line, start, end, contentCol int) []rule.TextEdit {
	firstList := firstListItem(lines, start+1, end)
	leadEnd := end // with no nested list, every body line is direct content
	if firstList >= 0 {
		leadEnd = firstList - 1
	}
	fixes := appendSetIndent(nil, lines, start+1, leadEnd, contentCol)
	if firstList < 0 {
		return fixes
	}
	listBase := leadingWhitespace(lines[firstList].Text)
	lastList := listBlockEnd(lines, firstList, end, listBase)
	if shift := contentCol - listBase; shift > 0 {
		pad := strings.Repeat(" ", shift)
		for j := firstList; j <= lastList; j++ {
			if !isBlank(lines[j].Text) {
				fixes = append(fixes, rule.TextEdit{Start: lines[j].Start, End: lines[j].Start, NewText: pad})
			}
		}
	}
	return appendSetIndent(fixes, lines, lastList+1, end, contentCol)
}

// firstListItem returns the index of the first list-item line in [from, to], or -1.
func firstListItem(lines []document.Line, from, to int) int {
	for j := from; j <= to; j++ {
		if !isBlank(lines[j].Text) && listMarkerRe.MatchString(lines[j].Text) {
			return j
		}
	}
	return -1
}

// listBlockEnd returns the last index of the run from firstList whose non-blank
// lines stay indented at least listBase (interior blanks don't end the run).
func listBlockEnd(lines []document.Line, firstList, end, listBase int) int {
	last := firstList
	for j := firstList; j <= end; j++ {
		if isBlank(lines[j].Text) {
			continue
		}
		if leadingWhitespace(lines[j].Text) < listBase {
			break
		}
		last = j
	}
	return last
}

// appendSetIndent appends a set-to-target edit for each non-blank line in [from,
// to] that is not already at target.
func appendSetIndent(fixes []rule.TextEdit, lines []document.Line, from, to, target int) []rule.TextEdit {
	for j := from; j <= to; j++ {
		if edit, ok := setIndent(lines[j], target); ok {
			fixes = append(fixes, edit)
		}
	}
	return fixes
}

// setIndent returns an edit that replaces a line's leading spaces with exactly
// target spaces, and false when the line is blank or already at target.
func setIndent(ln document.Line, target int) (rule.TextEdit, bool) {
	if isBlank(ln.Text) {
		return rule.TextEdit{}, false
	}
	cur := leadingWhitespace(ln.Text)
	if cur == target {
		return rule.TextEdit{}, false
	}
	return rule.TextEdit{Start: ln.Start, End: ln.Start + cur, NewText: strings.Repeat(" ", target)}, true
}
