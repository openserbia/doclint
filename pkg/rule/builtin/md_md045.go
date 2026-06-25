package builtin

import (
	"regexp"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

// emptyAltImageRe matches an inline Markdown image whose alt text — the bracket
// before the URL — is empty or only spaces/tabs: ![](url) or ![ ](url). The URL
// part [^)]* never crosses a closing paren, so it stops at the image's own ).
var emptyAltImageRe = regexp.MustCompile(`!\[[ \t]*\]\([^)]*\)`)

// NoAltText flags an inline Markdown image with empty or whitespace-only alt
// text (markdownlint MD045). The image still renders, so this is not a
// correctness break, but missing alt text is a real accessibility defect
// (screen readers announce nothing) and an SEO defect on a public multilingual
// content site (search engines lose the image's textual signal). The alt text
// must describe the image in the page's language, which only a human can author,
// so the finding is emitted with no automatic fix (NoFix).
type NoAltText struct{}

func (NoAltText) Meta() rule.Meta {
	return rule.Meta{
		Name:        "no-alt-text",
		Description: "images should have non-empty alt text for accessibility and SEO",
		Detail: "An inline image written ![](url) or ![ ](url) has empty (or " +
			"whitespace-only) alt text. The image still renders, but a screen " +
			"reader announces nothing for it and search engines lose the textual " +
			"signal the alt attribute carries — a real accessibility and SEO " +
			"defect on a public multilingual content site. This rule reports each " +
			"such image at its '!'. Image syntax that appears inside an inline " +
			"code span (`![](url)`) or a fenced code block is illustrative, renders " +
			"no image, and is ignored. No automatic fix is offered: meaningful alt " +
			"text describing the image must be authored by a human in the page's " +
			"language.",
		Severity: rule.Warning,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.NoFix,
	}
}

func (r NoAltText) Check(doc *document.Document, report func(rule.Finding)) {
	for _, ln := range doc.Lines {
		if ln.InFence {
			continue
		}
		spans := codeSpanRanges(ln.Text)
		for _, m := range emptyAltImageRe.FindAllStringIndex(ln.Text, -1) {
			bang := m[0] // byte index of the leading '!'
			if insideCodeSpan(bang, spans) {
				continue
			}
			report(r.finding(doc, ln, bang))
		}
	}
}

// finding builds the NoFix Warning for an image missing alt text, located at the
// '!' (byte index bang, reported as a 1-based column).
func (r NoAltText) finding(doc *document.Document, ln document.Line, bang int) rule.Finding {
	return rule.Finding{
		Rule:     r.Meta().Name,
		Path:     doc.Path,
		Line:     ln.Num,
		Col:      bang + 1,
		Message:  "image is missing alt text",
		Severity: rule.Warning,
		Safety:   rule.NoFix,
	}
}

// codeSpanRanges returns the [start,end) byte ranges of inline code spans in
// text. A code span opens with a run of N backticks and closes with the next run
// of exactly N backticks (CommonMark); a backtick run with no matching close is
// literal text and yields no span. Ranges are used to suppress image matches
// that fall inside `code`.
func codeSpanRanges(text string) [][2]int {
	var spans [][2]int
	i := 0
	for i < len(text) {
		if text[i] != '`' {
			i++
			continue
		}
		runLen := backtickRunLen(text, i)
		end := closingBacktickRun(text, i+runLen, runLen)
		if end < 0 {
			i += runLen // unclosed run: literal, skip past it
			continue
		}
		spans = append(spans, [2]int{i, end})
		i = end
	}
	return spans
}

// backtickRunLen returns the number of consecutive '`' characters at text[i:].
func backtickRunLen(text string, i int) int {
	n := 0
	for i+n < len(text) && text[i+n] == '`' {
		n++
	}
	return n
}

// closingBacktickRun returns the end index (exclusive) of the first backtick run
// of exactly want characters at or after from, or -1 if none exists. Runs of a
// different length are code-span content and are skipped.
func closingBacktickRun(text string, from, want int) int {
	for j := from; j < len(text); j++ {
		if text[j] != '`' {
			continue
		}
		n := backtickRunLen(text, j)
		if n == want {
			return j + n
		}
		j += n - 1 // skip the rest of this non-matching run (loop adds the last 1)
	}
	return -1
}

// insideCodeSpan reports whether byte position pos lies within any span range.
func insideCodeSpan(pos int, spans [][2]int) bool {
	for _, s := range spans {
		if pos >= s[0] && pos < s[1] {
			return true
		}
	}
	return false
}
