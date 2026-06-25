package builtin

import (
	"fmt"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

// TableColumnCount flags GFM table rows whose column count differs from the
// header. Ragged tables either render with dropped/empty cells or break the
// table entirely, so the mismatch is always a real defect. There is no safe
// autofix: the intended cell boundaries are ambiguous.
type TableColumnCount struct{}

func (TableColumnCount) Meta() rule.Meta {
	return rule.Meta{
		Name:        "table-column-count",
		Description: "require every table row to match the header's column count",
		Detail: "A GFM pipe table's column count is fixed by its header row. When a " +
			"data row has more or fewer cells, the renderer silently drops or pads " +
			"cells, so the table no longer means what the author wrote. This rule " +
			"reports each row whose cell count differs from the header. It emits no " +
			"fix because the correct cell boundaries cannot be inferred.",
		Severity: rule.Error,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.NoFix,
	}
}

func (t TableColumnCount) Check(doc *document.Document, report func(rule.Finding)) {
	for _, tbl := range document.Tables(doc) {
		idxs := make([]int, 0, len(tbl.RowIdxs)+1)
		idxs = append(idxs, tbl.HeaderIdx)
		idxs = append(idxs, tbl.RowIdxs...)
		for _, idx := range idxs {
			ln := doc.Lines[idx]
			got := len(document.SplitTableCells(ln.Text))
			if got == tbl.Cols {
				continue
			}
			report(rule.Finding{
				Rule:     t.Meta().Name,
				Path:     doc.Path,
				Line:     ln.Num,
				Col:      1,
				Message:  fmt.Sprintf("table row has %d columns; header defines %d", got, tbl.Cols),
				Severity: rule.Error,
				Safety:   rule.NoFix,
			})
		}
	}
}
