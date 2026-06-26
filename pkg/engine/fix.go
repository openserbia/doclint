// Package engine discovers files, runs rules in parallel, applies inline
// suppression, and applies fixes or renders diffs.
package engine

import (
	"fmt"
	"sort"

	"github.com/pmezard/go-difflib/difflib"

	"github.com/openserbia/doclint/pkg/rule"
)

// ApplyEdits returns src with every edit applied. Edits must not overlap; they
// are applied last-to-first so earlier offsets stay valid during splicing.
func ApplyEdits(src []byte, edits []rule.TextEdit) ([]byte, error) {
	if len(edits) == 0 {
		return src, nil
	}
	sorted := make([]rule.TextEdit, len(edits))
	copy(sorted, edits)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start < sorted[j].Start })

	for i := 1; i < len(sorted); i++ {
		if sorted[i].Start < sorted[i-1].End {
			return nil, fmt.Errorf("overlapping edits at offset %d", sorted[i].Start)
		}
	}
	out := make([]byte, len(src))
	copy(out, src)
	for i := len(sorted) - 1; i >= 0; i-- {
		e := sorted[i]
		if e.Start < 0 || e.End > len(out) || e.Start > e.End {
			return nil, fmt.Errorf("edit out of range [%d:%d] (len %d)", e.Start, e.End, len(out))
		}
		out = append(out[:e.Start], append([]byte(e.NewText), out[e.End:]...)...)
	}
	return out, nil
}

// coalesceBlankInserts drops redundant blank-line insertions that target the
// same inter-line boundary. Two rules can each request a blank at one spot — e.g.
// blanks-around-headings (a blank after a heading) and blanks-around-lists (a
// blank before the immediately following list) both bracket the single newline
// between the heading and the list. Applying both stacks two blank lines where
// one is wanted (fmt's blank-run collapse hides this, but lint --fix does not).
// This keeps the first insertion per bracketed newline; every non-blank-insert
// edit passes through untouched and in original order.
func coalesceBlankInserts(src []byte, edits []rule.TextEdit) []rule.TextEdit {
	seen := make(map[int]bool)
	out := make([]rule.TextEdit, 0, len(edits))
	for _, e := range edits {
		if e.Start != e.End || e.NewText != "\n" {
			out = append(out, e)
			continue
		}
		key := bracketedNewline(src, e.Start)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, e)
	}
	return out
}

// bracketedNewline returns the offset of the existing newline a blank-line
// insertion at off sits next to: off when a newline follows (a line-end anchor),
// off-1 when a newline precedes (a line-start anchor), else off (a document
// edge). Insertions that bracket the same newline share a key, so they coalesce.
func bracketedNewline(src []byte, off int) int {
	if off >= 0 && off < len(src) && src[off] == '\n' {
		return off
	}
	if off-1 >= 0 && off-1 < len(src) && src[off-1] == '\n' {
		return off - 1
	}
	return off
}

// diffContext is the number of unchanged lines shown around each diff hunk.
const diffContext = 3

// UnifiedDiff renders a unified diff of before vs after for one file.
func UnifiedDiff(path string, before, after []byte) string {
	d := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(before)),
		B:        difflib.SplitLines(string(after)),
		FromFile: "a/" + path,
		ToFile:   "b/" + path,
		Context:  diffContext,
	}
	text, _ := difflib.GetUnifiedDiffString(d)
	return text
}
