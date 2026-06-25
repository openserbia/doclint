package document

import (
	"reflect"
	"testing"
)

func TestSplitTableCells(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"bordered", "| a | b | c |", []string{"a", "b", "c"}},
		{"unbordered", "a | b | c", []string{"a", "b", "c"}},
		{"leading_only", "| a | b", []string{"a", "b"}},
		{"trailing_only", "a | b |", []string{"a", "b"}},
		{"escaped_pipe", `| a \| b | c |`, []string{`a \| b`, "c"}},
		{"empty_cells", "|  |  |", []string{"", ""}},
		{"single", "| only |", []string{"only"}},
		{"trailing_escaped", `| a | b\| |`, []string{"a", `b\|`}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SplitTableCells(c.in); !reflect.DeepEqual(got, c.want) {
				t.Errorf("SplitTableCells(%q) = %#v, want %#v", c.in, got, c.want)
			}
		})
	}
}

func TestTables_BasicThreeColumn(t *testing.T) {
	raw := []byte("intro\n\n| a | b | c |\n|---|---|---|\n| 1 | 2 | 3 |\n| 4 | 5 | 6 |\n\nafter\n")
	doc := &Document{Raw: raw, Lines: SplitLines(raw)}
	tables := Tables(doc)
	if len(tables) != 1 {
		t.Fatalf("got %d tables, want 1", len(tables))
	}
	tbl := tables[0]
	if tbl.Cols != 3 {
		t.Errorf("Cols = %d, want 3", tbl.Cols)
	}
	if tbl.HeaderIdx != 2 || tbl.SepIdx != 3 {
		t.Errorf("HeaderIdx=%d SepIdx=%d, want 2 and 3", tbl.HeaderIdx, tbl.SepIdx)
	}
	if !reflect.DeepEqual(tbl.RowIdxs, []int{4, 5}) {
		t.Errorf("RowIdxs = %v, want [4 5]", tbl.RowIdxs)
	}
}

func TestTables_Alignment(t *testing.T) {
	raw := []byte("| a | b | c | d |\n|:--|:-:|--:|---|\n| 1 | 2 | 3 | 4 |\n")
	doc := &Document{Raw: raw, Lines: SplitLines(raw)}
	tables := Tables(doc)
	if len(tables) != 1 {
		t.Fatalf("got %d tables, want 1", len(tables))
	}
	want := []string{"left", "center", "right", ""}
	if !reflect.DeepEqual(tables[0].Align, want) {
		t.Errorf("Align = %#v, want %#v", tables[0].Align, want)
	}
}

func TestTables_SkipsFenced(t *testing.T) {
	raw := []byte("```\n| a | b |\n|---|---|\n| 1 | 2 |\n```\n")
	doc := &Document{Raw: raw, Lines: SplitLines(raw)}
	if tables := Tables(doc); len(tables) != 0 {
		t.Fatalf("got %d tables inside fence, want 0", len(tables))
	}
}

func TestTables_StopsAtBlankAndNonPipe(t *testing.T) {
	raw := []byte("| a | b |\n|---|---|\n| 1 | 2 |\nnot a row\n")
	doc := &Document{Raw: raw, Lines: SplitLines(raw)}
	tables := Tables(doc)
	if len(tables) != 1 {
		t.Fatalf("got %d tables, want 1", len(tables))
	}
	if !reflect.DeepEqual(tables[0].RowIdxs, []int{2}) {
		t.Errorf("RowIdxs = %v, want [2]", tables[0].RowIdxs)
	}
}

func TestTables_RequiresHeaderWithPipe(t *testing.T) {
	// Separator with no pipe-bearing header above it is not a table.
	raw := []byte("plain heading\n---\n")
	doc := &Document{Raw: raw, Lines: SplitLines(raw)}
	if tables := Tables(doc); len(tables) != 0 {
		t.Fatalf("got %d tables, want 0", len(tables))
	}
}
