package builtin

import (
	"regexp"
	"strings"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

// centerOpenRe matches an opening <center> tag (optionally indented).
var centerOpenRe = regexp.MustCompile(`(?i)^\s*<center>\s*$`)

// centerCloseRe matches a closing </center> tag (optionally indented).
var centerCloseRe = regexp.MustCompile(`(?i)^\s*</center>\s*$`)

// BlanksAroundCenter flags a <center>…</center> block that is not surrounded by
// blank lines. Goldmark (the parser Hugo uses) treats <center> as an HTML block
// whose scope runs until the next blank line. When a list item or other markdown
// content is butted directly against </center> (or <center> directly against
// preceding content), the markdown is absorbed into the HTML block and never
// renders. The fix inserts a single blank line on the offending side —
// content-neutral and idempotent. The document's first and last lines are exempt.
type BlanksAroundCenter struct{}

func (BlanksAroundCenter) Meta() rule.Meta {
	return rule.Meta{
		Name:        "blanks-around-center",
		Title:       "Blank lines around <center> blocks",
		Description: "<center>…</center> blocks should be surrounded by blank lines",
		Detail: "A <center>…</center> block needs a blank line before the " +
			"opening <center> tag and after the closing </center> tag. Goldmark " +
			"(the parser Hugo uses) treats <center> as an HTML block that runs " +
			"until the next blank line. When markdown content (such as a list " +
			"item) is placed directly after </center> without a blank line, it " +
			"is absorbed into the HTML block and never renders as markdown. " +
			"Similarly, content directly before <center> can fail to separate " +
			"properly from the HTML block. This rule reports each missing " +
			"surrounding blank line and inserts one (a safe, idempotent fix). " +
			"The document's first and last lines are exempt.",
		Severity: rule.Warning,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.Safe,
		Example: rule.Example{
			Bad: `- list item
<center>
{{< figure src="img.png" >}}
</center>
- next item`,
			Good: `- list item

<center>
{{< figure src="img.png" >}}
</center>

- next item`,
		},
	}
}

func (r BlanksAroundCenter) Check(doc *document.Document, report func(rule.Finding)) {
	lines := doc.Lines
	for i, ln := range lines {
		if ln.InFence || ln.Start < doc.BodyOffset {
			continue
		}
		if centerOpenRe.MatchString(ln.Text) {
			r.checkOpen(doc, lines, i, report)
		}
		if centerCloseRe.MatchString(ln.Text) {
			r.checkClose(doc, lines, i, report)
		}
	}
}

// checkOpen reports a missing blank line before the opening <center> tag at
// index i (unless it is the document's first body line, which is exempt).
func (r BlanksAroundCenter) checkOpen(doc *document.Document, lines []document.Line, i int, report func(rule.Finding)) {
	if i == 0 {
		return
	}
	prev := lines[i-1]
	if prev.Start < doc.BodyOffset || strings.TrimSpace(prev.Text) == "" {
		return
	}
	ln := lines[i]
	report(r.finding(doc, ln, "missing blank line before <center>",
		rule.TextEdit{Start: ln.Start, End: ln.Start, NewText: "\n"}))
}

// checkClose reports a missing blank line after the closing </center> tag at
// index i (unless it is the document's last line, which is exempt).
func (r BlanksAroundCenter) checkClose(doc *document.Document, lines []document.Line, i int, report func(rule.Finding)) {
	if i+1 >= len(lines) || strings.TrimSpace(lines[i+1].Text) == "" {
		return
	}
	ln := lines[i]
	report(r.finding(doc, ln, "missing blank line after </center>",
		rule.TextEdit{Start: ln.End, End: ln.End, NewText: "\n"}))
}

// finding assembles a Warning finding with a single safe blank-insertion fix for
// the tag line ln.
func (r BlanksAroundCenter) finding(doc *document.Document, ln document.Line, msg string, edit rule.TextEdit) rule.Finding {
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
