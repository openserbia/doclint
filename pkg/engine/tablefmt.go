package engine

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"github.com/openserbia/doclint/pkg/document"
)

// minColWidth is the minimum rendered width of a table column, so even a
// single-character column keeps a readable delimiter (`---`, `:-:`).
const minColWidth = 3

// formatTables reformats every well-formed GFM table in raw with aligned
// columns: cells are left-justified to a shared per-column width and the
// delimiter row's alignment colons are preserved. Malformed tables (any row
// whose cell count differs from the header) are left untouched. The pass is
// idempotent: formatting already-aligned tables is a no-op.
func formatTables(raw []byte) []byte {
	doc := &document.Document{Raw: raw, Lines: document.SplitLines(raw)}
	tables := document.Tables(doc)
	if len(tables) == 0 {
		return raw
	}

	starts := make(map[int]document.Table, len(tables))
	for _, t := range tables {
		if wellFormedTable(doc, t) {
			starts[t.HeaderIdx] = t
		}
	}
	if len(starts) == 0 {
		return raw
	}

	var b bytes.Buffer
	for i := 0; i < len(doc.Lines); i++ {
		if t, ok := starts[i]; ok {
			b.WriteString(renderTable(doc, t))
			i = lastTableIdx(t)
			continue
		}
		b.WriteString(doc.Lines[i].Text)
		b.WriteByte('\n')
	}
	out := bytes.TrimRight(b.Bytes(), "\n")
	return append(out, '\n')
}

// wellFormedTable reports whether every row (delimiter and data) has exactly
// Cols cells. The header is Cols by construction.
func wellFormedTable(doc *document.Document, t document.Table) bool {
	if len(document.SplitTableCells(doc.Lines[t.SepIdx].Text)) != t.Cols {
		return false
	}
	for _, idx := range t.RowIdxs {
		if len(document.SplitTableCells(doc.Lines[idx].Text)) != t.Cols {
			return false
		}
	}
	return true
}

// lastTableIdx returns the Lines index of the table's final line.
func lastTableIdx(t document.Table) int {
	if n := len(t.RowIdxs); n > 0 {
		return t.RowIdxs[n-1]
	}
	return t.SepIdx
}

// renderTable produces the aligned text (header, delimiter, data rows) for one
// well-formed table, each line terminated by a newline.
func renderTable(doc *document.Document, t document.Table) string {
	idxs := make([]int, 0, len(t.RowIdxs)+1)
	idxs = append(idxs, t.HeaderIdx)
	idxs = append(idxs, t.RowIdxs...)

	cells := make([][]string, len(idxs))
	for i, idx := range idxs {
		cells[i] = document.SplitTableCells(doc.Lines[idx].Text)
	}
	widths := tableWidths(cells, t.Cols)

	var b strings.Builder
	b.WriteString(renderTableRow(cells[0], widths))
	b.WriteString(renderDelimiterRow(widths, t.Align))
	for _, row := range cells[1:] {
		b.WriteString(renderTableRow(row, widths))
	}
	return b.String()
}

// tableWidths computes the rendered width of each of cols columns: the larger
// of minColWidth and the widest cell (in runes) in that column.
func tableWidths(cells [][]string, cols int) []int {
	widths := make([]int, cols)
	for c := range cols {
		w := minColWidth
		for _, row := range cells {
			if c < len(row) {
				if n := utf8.RuneCountInString(row[c]); n > w {
					w = n
				}
			}
		}
		widths[c] = w
	}
	return widths
}

// renderTableRow renders one content row as `| cell | cell |`, right-padding
// each cell to its column width.
func renderTableRow(cells []string, widths []int) string {
	var b strings.Builder
	b.WriteByte('|')
	for c, w := range widths {
		cell := ""
		if c < len(cells) {
			cell = cells[c]
		}
		pad := w - utf8.RuneCountInString(cell)
		if pad < 0 {
			pad = 0
		}
		b.WriteByte(' ')
		b.WriteString(cell)
		b.WriteString(strings.Repeat(" ", pad))
		b.WriteString(" |")
	}
	b.WriteByte('\n')
	return b.String()
}

// renderDelimiterRow renders the `| --- | :-: |` separator row, restoring each
// column's alignment colons.
func renderDelimiterRow(widths []int, align []string) string {
	var b strings.Builder
	b.WriteByte('|')
	for c, w := range widths {
		a := ""
		if c < len(align) {
			a = align[c]
		}
		b.WriteByte(' ')
		b.WriteString(delimiterCell(w, a))
		b.WriteString(" |")
	}
	b.WriteByte('\n')
	return b.String()
}

// delimiterCell builds a single delimiter cell of the given width, encoding
// alignment with leading/trailing colons.
func delimiterCell(width int, align string) string {
	left := align == "left" || align == "center"
	right := align == "right" || align == "center"
	dashes := width
	if left {
		dashes--
	}
	if right {
		dashes--
	}
	cell := strings.Repeat("-", dashes)
	if left {
		cell = ":" + cell
	}
	if right {
		cell += ":"
	}
	return cell
}
