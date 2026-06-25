package document

import (
	"regexp"
	"strings"
)

// Table describes one GFM pipe table located in a Document. All index fields
// (HeaderIdx, SepIdx, RowIdxs) are positions into Document.Lines, not line
// numbers, so callers can read the underlying Line for offsets and text.
type Table struct {
	Cols      int      // column count, taken from the header row
	HeaderIdx int      // Lines index of the header row
	SepIdx    int      // Lines index of the delimiter (separator) row
	RowIdxs   []int    // Lines indices of the data rows
	Align     []string // per-column alignment: "", "left", "center", "right"
}

// tableSepRe matches a GFM table delimiter row such as `|---|:--:|--:|`.
var tableSepRe = regexp.MustCompile(`^\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$`)

// isTableSeparator reports whether text is a GFM delimiter row.
func isTableSeparator(text string) bool {
	s := strings.TrimSpace(text)
	return strings.Contains(s, "-") && tableSepRe.MatchString(s)
}

// SplitTableCells splits one table row into trimmed cell values. It strips a
// single leading and trailing pipe if present and splits on unescaped pipes,
// preserving `\|` escapes inside cell text.
func SplitTableCells(text string) []string {
	s := strings.TrimSpace(text)
	s = strings.TrimPrefix(s, "|")
	s = strings.TrimSuffix(s, "|")

	var (
		cells []string
		cur   strings.Builder
	)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			cur.WriteByte(c)
			cur.WriteByte(s[i+1])
			i++
			continue
		}
		if c == '|' {
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	cells = append(cells, strings.TrimSpace(cur.String()))
	return cells
}

// parseAlign derives per-column alignment from a delimiter row's colons.
func parseAlign(sep string) []string {
	cells := SplitTableCells(sep)
	align := make([]string, len(cells))
	for i, c := range cells {
		c = strings.TrimSpace(c)
		left := strings.HasPrefix(c, ":")
		right := strings.HasSuffix(c, ":")
		switch {
		case left && right:
			align[i] = "center"
		case left:
			align[i] = "left"
		case right:
			align[i] = "right"
		default:
			align[i] = ""
		}
	}
	return align
}

// Tables finds every GFM pipe table in d. A table is a delimiter row whose
// preceding non-fenced line contains a pipe (the header), followed by zero or
// more data rows up to the next blank or non-pipe line. Fenced lines are
// skipped.
func Tables(d *Document) []Table {
	var tables []Table
	lines := d.Lines
	for i := 0; i < len(lines); {
		t, next, ok := tableAt(lines, i)
		if !ok {
			i++
			continue
		}
		tables = append(tables, t)
		i = next
	}
	return tables
}

// tableAt tries to read a table whose delimiter row is at index i. It returns
// the table, the next line index to resume scanning from, and whether a table
// was found.
func tableAt(lines []Line, i int) (Table, int, bool) {
	ln := lines[i]
	if ln.InFence || i == 0 || !isTableSeparator(ln.Text) {
		return Table{}, 0, false
	}
	hdr := lines[i-1]
	if hdr.InFence || !strings.Contains(hdr.Text, "|") {
		return Table{}, 0, false
	}

	rows := gatherRows(lines, i+1)
	t := Table{
		Cols:      len(SplitTableCells(hdr.Text)),
		HeaderIdx: i - 1,
		SepIdx:    i,
		RowIdxs:   rows,
		Align:     parseAlign(ln.Text),
	}
	next := i + 1
	if n := len(rows); n > 0 {
		next = rows[n-1] + 1
	}
	return t, next, true
}

// gatherRows collects consecutive data-row indices starting at from, stopping
// at the first fenced, blank, or non-pipe line.
func gatherRows(lines []Line, from int) []int {
	var rows []int
	for j := from; j < len(lines); j++ {
		r := lines[j]
		if r.InFence || strings.TrimSpace(r.Text) == "" || !strings.Contains(r.Text, "|") {
			break
		}
		rows = append(rows, j)
	}
	return rows
}
