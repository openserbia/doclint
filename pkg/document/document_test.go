package document

import "testing"

func TestSplitLines_TracksFenceState(t *testing.T) {
	raw := []byte("a\n```\nin fence\n```\nb\n")
	lines := SplitLines(raw)

	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5", len(lines))
	}
	want := []struct {
		text    string
		inFence bool
	}{
		{"a", false},
		{"```", false},     // fence delimiter is not "in fence"
		{"in fence", true}, // interior
		{"```", false},     // closing delimiter
		{"b", false},
	}
	for i, w := range want {
		if lines[i].Text != w.text || lines[i].InFence != w.inFence {
			t.Errorf("line %d = {%q, inFence=%v}, want {%q, %v}",
				i, lines[i].Text, lines[i].InFence, w.text, w.inFence)
		}
	}
	if lines[0].Num != 1 || lines[2].Start != 6 {
		t.Errorf("offsets wrong: line0.Num=%d line2.Start=%d", lines[0].Num, lines[2].Start)
	}
}
