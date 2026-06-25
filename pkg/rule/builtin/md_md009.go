package builtin

import (
	"strconv"
	"strings"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

// hardBreakSpaces is the exact number of trailing spaces CommonMark treats as an
// intentional hard line break (rendered as <br>). A run of exactly this length is
// deliberate authoring and is never flagged; a shorter run (a single stray space)
// or a longer run (which the renderer collapses back to a two-space break) is.
const hardBreakSpaces = 2

// NoTrailingSpaces flags trailing spaces that are almost certainly unintended
// (markdownlint MD009). A line ending in exactly two spaces is a markdown hard
// line break and is left untouched; everything else is suspect: a single trailing
// space is an invisible stray that nothing renders, three or more spaces collapse
// to the same two-space break (so the extras are noise the author probably did
// not mean), and a whitespace-only line has no preceding content for a break to
// attach to. The finding is emitted with no automatic fix (NoFix): the engine's
// fmt pass deliberately does not strip trailing whitespace, precisely because a
// blanket trim would destroy the two-space hard break this rule protects, so the
// only safe action is to surface the line for a human to inspect.
type NoTrailingSpaces struct{}

func (NoTrailingSpaces) Meta() rule.Meta {
	return rule.Meta{
		Name:        "no-trailing-spaces",
		Description: "flag stray trailing spaces while preserving the two-space hard line break",
		Detail: "Trailing spaces at the end of a line are invisible and usually " +
			"accidental. CommonMark gives exactly two trailing spaces a single " +
			"meaning — a hard line break (<br>) — so this rule never flags a " +
			"two-space run; that is intentional formatting. It does flag a single " +
			"trailing space (an invisible stray that renders as nothing) and a run " +
			"of three or more (which the renderer collapses back to a two-space " +
			"break, making the extra spaces meaningless noise). A whitespace-only " +
			"line is flagged for any trailing spaces, since with no preceding " +
			"content there is nothing a hard break could attach to. No automatic " +
			"fix is offered: the fmt pass refuses to strip trailing whitespace " +
			"because a blanket trim would silently delete the two-space hard break, " +
			"so the line is surfaced for a human to fix. Lines inside a fenced code " +
			"block are significant content and are ignored.",
		Severity: rule.Warning,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.NoFix,
	}
}

func (r NoTrailingSpaces) Check(doc *document.Document, report func(rule.Finding)) {
	for _, ln := range doc.Lines {
		if ln.InFence {
			continue
		}
		t := ln.Text
		n := len(t) - len(strings.TrimRight(t, " "))
		if n == 0 {
			continue
		}
		switch {
		case strings.TrimSpace(t) == "":
			report(r.finding(doc, ln, n, "whitespace-only line has "+
				countPhrase(n)+"; remove them (a blank line should be empty)"))
		case n == hardBreakSpaces:
			// Intentional hard line break (<br>); never flag, never edit.
		case n < hardBreakSpaces:
			report(r.finding(doc, ln, n,
				"line ends in a single stray trailing space; it renders as nothing — remove it"))
		default: // n >= 3
			report(r.finding(doc, ln, n, "line ends in "+countPhrase(n)+
				"; the renderer collapses them to a 2-space hard break, so the extras are likely unintended"))
		}
	}
}

// countPhrase renders a trailing-space count as a human phrase ("1 trailing
// space" / "4 trailing spaces") so messages read naturally.
func countPhrase(n int) string {
	if n < hardBreakSpaces { // n == 1: the only single-space case reaching here
		return "1 trailing space"
	}
	return strconv.Itoa(n) + " trailing spaces"
}

// finding builds the NoFix Warning at the first trailing space (1-based column).
func (r NoTrailingSpaces) finding(doc *document.Document, ln document.Line, n int, msg string) rule.Finding {
	return rule.Finding{
		Rule:     r.Meta().Name,
		Path:     doc.Path,
		Line:     ln.Num,
		Col:      len(ln.Text) - n + 1,
		Message:  msg,
		Severity: rule.Warning,
		Safety:   rule.NoFix,
	}
}
