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
			"lists, and the numbering restarts (1. 1. 1. instead of 1. 2. 3.). The fix " +
			"re-indents the whole item body to the content column. It is Unsafe (only " +
			"applied with --unsafe-fixes) because shifting nested content — especially " +
			"across Hugo shortcodes — can warrant a human's review of the diff.",
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
			r.flag(doc, lines, i, end, contentCol-base, report)
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

// flag reports the under-indented item and a single Unsafe fix that shifts every
// non-blank body line right by delta, preserving the relative nesting.
func (r ListMarkerIndent) flag(doc *document.Document, lines []document.Line, start, end, delta int, report func(rule.Finding)) {
	pad := strings.Repeat(" ", delta)
	var fixes []rule.TextEdit
	for j := start + 1; j <= end; j++ {
		if isBlank(lines[j].Text) {
			continue
		}
		fixes = append(fixes, rule.TextEdit{Start: lines[j].Start, End: lines[j].Start, NewText: pad})
	}
	report(rule.Finding{
		Rule:     "list-marker-indent",
		Path:     doc.Path,
		Line:     lines[start].Num,
		Col:      1,
		Message:  fmt.Sprintf("list item body is under-indented; indent it %d more space(s) to the marker's content column", delta),
		Severity: rule.Warning,
		Safety:   rule.Unsafe,
		Fixes:    fixes,
	})
}
