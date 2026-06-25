// Package document turns a file's bytes into a Document: raw content,
// fence-aware source lines, and a parsed frontmatter/data map. It is the
// format-agnostic substrate every rule reads from.
package document

import "regexp"

// Format identifies how a file is parsed and which rules apply to it.
type Format string

const (
	Markdown Format = "markdown"
	Data     Format = "data"
)

// Line is one source line with byte offsets and fenced-code-block state.
type Line struct {
	Num     int    // 1-based line number
	Text    string // line content without the trailing newline
	Start   int    // byte offset of the line start in Document.Raw
	End     int    // byte offset just past Text (before the newline)
	InFence bool   // true when the line is INSIDE a fenced code block
}

// Document is the parsed view of a single file.
type Document struct {
	Path        string
	Format      Format
	Raw         []byte
	Lines       []Line
	Frontmatter map[string]any // markdown: parsed frontmatter; data: whole file
	Body        []byte         // markdown content after frontmatter (nil for data)
	BodyOffset  int            // byte offset in Raw where Body begins (0 for data)
}

var fenceRe = regexp.MustCompile("^[ \\t]*(```|~~~)")

// SplitLines splits raw into fence-aware Lines. A fence delimiter line toggles
// the in-fence state but is itself reported with InFence=false; only the lines
// strictly between an opening and closing delimiter are InFence=true.
func SplitLines(raw []byte) []Line {
	var lines []Line
	inFence := false
	start := 0
	num := 1
	for i := 0; i <= len(raw); i++ {
		if i < len(raw) && raw[i] != '\n' {
			continue
		}
		text := string(raw[start:i])
		isFence := fenceRe.MatchString(text)
		ln := Line{Num: num, Text: text, Start: start, End: i, InFence: inFence && !isFence}
		lines = append(lines, ln)
		if isFence {
			inFence = !inFence
		}
		start = i + 1
		num++
		if i == len(raw) {
			break
		}
	}
	// A trailing newline produces a final empty line; drop it so line counts
	// match editors (which don't show a phantom line after the last newline).
	if len(lines) > 1 && lines[len(lines)-1].Text == "" && len(raw) > 0 && raw[len(raw)-1] == '\n' {
		lines = lines[:len(lines)-1]
	}
	return lines
}
