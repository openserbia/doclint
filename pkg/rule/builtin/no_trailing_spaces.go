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
		Title:       "Trailing whitespace",
		Description: "remove stray trailing spaces while preserving the two-space hard line break",
		Detail: "Trailing spaces at the end of a line are invisible and usually " +
			"accidental. CommonMark gives exactly two trailing spaces a single " +
			"meaning — a hard line break (<br>) — so this rule never flags a " +
			"two-space run; that is intentional formatting. A single trailing space " +
			"(an invisible stray that renders as nothing) and a whitespace-only line " +
			"(no preceding content for a break to attach to) are unambiguous, so each " +
			"carries a safe autofix that strips it — and because the fix is targeted, " +
			"it never touches the two-space hard break. A run of three or more is " +
			"flagged WITHOUT a fix: the renderer collapses it back to a two-space " +
			"break, so whether the author meant a (sloppy) break or stray spaces is " +
			"ambiguous and a human should decide. Lines inside a fenced code block " +
			"are significant content and are ignored.",
		Severity: rule.Warning,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.Safe,
		Example: rule.Example{
			Bad:  "first line has a stray trailing space \nsecond line",
			Good: "first line has a stray trailing space\nsecond line",
		},
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
			// Whitespace-only line: clear it entirely (safe — a blank line renders
			// the same, and nothing here can be a hard break).
			report(r.finding(doc, ln, n, "whitespace-only line has "+
				countPhrase(n)+"; remove them (a blank line should be empty)",
				rule.Safe, rule.TextEdit{Start: ln.Start, End: ln.End, NewText: ""}))
		case n == hardBreakSpaces:
			// Intentional hard line break (<br>); never flag, never edit.
		case n < hardBreakSpaces:
			// A single stray space — unambiguous (one space is never a break), so
			// strip it with a safe fix.
			report(r.finding(doc, ln, n,
				"line ends in a single stray trailing space; it renders as nothing — remove it",
				rule.Safe, rule.TextEdit{Start: ln.End - n, End: ln.End, NewText: ""}))
		default: // n >= 3: the renderer turns it into a break — ambiguous, flag only.
			report(r.finding(doc, ln, n, "line ends in "+countPhrase(n)+
				"; the renderer collapses them to a 2-space hard break, so the extras are likely unintended — fix by hand",
				rule.NoFix))
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

// finding builds a Warning at the first trailing space (1-based column) with the
// given fix safety and zero or one strip edit.
func (r NoTrailingSpaces) finding(doc *document.Document, ln document.Line, n int, msg string, safety rule.FixSafety, fixes ...rule.TextEdit) rule.Finding {
	return rule.Finding{
		Rule:     r.Meta().Name,
		Path:     doc.Path,
		Line:     ln.Num,
		Col:      len(ln.Text) - n + 1,
		Message:  msg,
		Severity: rule.Warning,
		Safety:   safety,
		Fixes:    fixes,
	}
}
