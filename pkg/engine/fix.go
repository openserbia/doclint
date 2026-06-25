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
