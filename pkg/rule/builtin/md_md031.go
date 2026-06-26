package builtin

import (
	"regexp"
	"strings"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

// fenceDelimRe matches a fenced-code-block delimiter line once leading
// whitespace is allowed: ``` or ~~~ at the (optionally indented) line start.
// It mirrors document.SplitLines' own fence detection so this rule's view of
// where fences open and close agrees byte-for-byte with Line.InFence.
var fenceDelimRe = regexp.MustCompile("^[ \t]*(```|~~~)")

// BlanksAroundFences flags a fenced code block whose opening delimiter is
// immediately preceded by a non-blank line, or whose closing delimiter is
// immediately followed by a non-blank line (markdownlint MD031). A fence butted
// directly against a paragraph can fail to be recognized as a code block at all,
// so the content renders as prose. The fix inserts a single blank line on the
// offending side — content-neutral and idempotent. The document's first and last
// lines are exempt (a fence at the very start or end needs no separating blank).
type BlanksAroundFences struct{}

func (BlanksAroundFences) Meta() rule.Meta {
	return rule.Meta{
		Name:        "blanks-around-fences",
		Title:       "Blank lines around code fences",
		Description: "fenced code blocks should be surrounded by blank lines",
		Detail: "A fenced code block (``` or ~~~) needs a blank line before its " +
			"opening delimiter and after its closing delimiter. When a fence is " +
			"butted directly against a preceding or following paragraph, " +
			"CommonMark/Goldmark (the parser Hugo uses) can fail to recognize it as " +
			"a code block, so the fenced content renders as ordinary prose instead " +
			"of preformatted code. This rule reports each delimiter that is missing " +
			"its surrounding blank line and inserts one (a safe fix); the insertion " +
			"is content-neutral and idempotent. The document's first and last lines " +
			"are exempt, since a fence at the very start or end of the file has no " +
			"adjacent content to separate from.",
		Severity: rule.Warning,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.Safe,
		Example: rule.Example{
			Bad: `text
~~~
code
~~~
more`,
			Good: `text

~~~
code
~~~

more`,
		},
	}
}

func (r BlanksAroundFences) Check(doc *document.Document, report func(rule.Finding)) {
	lines := doc.Lines
	open := false
	for i := range lines {
		if !fenceDelimRe.MatchString(lines[i].Text) {
			continue
		}
		if open {
			r.checkClose(doc, lines, i, report)
		} else {
			r.checkOpen(doc, lines, i, report)
		}
		open = !open
	}
}

// checkOpen reports a missing blank line before the opening delimiter at index i
// (unless it is the document's first line, which is exempt).
func (r BlanksAroundFences) checkOpen(doc *document.Document, lines []document.Line, i int, report func(rule.Finding)) {
	if i == 0 || strings.TrimSpace(lines[i-1].Text) == "" {
		return
	}
	ln := lines[i]
	report(r.finding(doc, ln, "missing blank line before fenced code block",
		rule.TextEdit{Start: ln.Start, End: ln.Start, NewText: "\n"}))
}

// checkClose reports a missing blank line after the closing delimiter at index j
// (unless it is the document's last line, which is exempt).
func (r BlanksAroundFences) checkClose(doc *document.Document, lines []document.Line, j int, report func(rule.Finding)) {
	if j+1 >= len(lines) || strings.TrimSpace(lines[j+1].Text) == "" {
		return
	}
	ln := lines[j]
	report(r.finding(doc, ln, "missing blank line after fenced code block",
		rule.TextEdit{Start: ln.End, End: ln.End, NewText: "\n"}))
}

// finding assembles a Warning finding with a single safe blank-insertion fix for
// the delimiter line ln.
func (r BlanksAroundFences) finding(doc *document.Document, ln document.Line, msg string, edit rule.TextEdit) rule.Finding {
	return rule.Finding{
		Rule:     r.Meta().Name,
		Path:     doc.Path,
		Line:     ln.Num,
		Col:      1,
		Message:  msg,
		Severity: rule.Warning,
		Safety:   rule.Safe,
		Fixes:    []rule.TextEdit{edit},
	}
}
