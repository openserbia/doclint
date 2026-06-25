// Package builtin holds programmatic (Go) rules.
package builtin

import (
	"strings"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

const closeTag = "</summary>"

// DetailsBlankLine enforces a blank line after a literal </summary>. Goldmark
// (the parser Hugo uses) treats <details><summary>…</summary> as an HTML block
// that runs until the next blank line; without it the inner markdown is
// swallowed as raw HTML and never renders.
type DetailsBlankLine struct{}

func (DetailsBlankLine) Meta() rule.Meta {
	return rule.Meta{
		Name:        "details-blank-line",
		Description: "require a blank line after </summary> so inner markdown renders",
		Detail: "Goldmark parses <details><summary>…</summary> as an HTML block " +
			"that ends at the next blank line. If content or markdown follows " +
			"</summary> on the same line or the very next line, it is captured as " +
			"raw HTML and never rendered. The fix inserts a blank line (and splits " +
			"any content glued onto the </summary> line).",
		Severity: rule.Error,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.Safe,
	}
}

func (d DetailsBlankLine) Check(doc *document.Document, report func(rule.Finding)) {
	lines := doc.Lines
	for i, ln := range lines {
		if ln.InFence || !strings.Contains(ln.Text, closeTag) {
			continue
		}
		cut := strings.LastIndex(ln.Text, closeTag) + len(closeTag)
		trailing := strings.TrimSpace(ln.Text[cut:])

		if trailing != "" {
			d.reportGluedContent(doc, ln, cut, trailing, report)
			continue
		}

		d.reportMissingBlankLine(doc, ln, i, lines, report)
	}
}

func (d DetailsBlankLine) reportGluedContent(
	doc *document.Document,
	ln document.Line,
	cut int,
	trailing string,
	report func(rule.Finding),
) {
	indent := ln.Text[:len(ln.Text)-len(strings.TrimLeft(ln.Text, " \t"))]
	insertAt := ln.Start + cut
	report(rule.Finding{
		Rule:     d.Meta().Name,
		Path:     doc.Path,
		Line:     ln.Num,
		Col:      cut + 1,
		Message:  "content must not follow </summary> on the same line; put it on its own line after a blank line",
		Severity: rule.Error,
		Safety:   rule.Safe,
		Fixes: []rule.TextEdit{{
			Start:   insertAt,
			End:     ln.End,
			NewText: "\n\n" + indent + trailing,
		}},
	})
}

func (d DetailsBlankLine) reportMissingBlankLine(
	doc *document.Document,
	ln document.Line,
	i int,
	lines []document.Line,
	report func(rule.Finding),
) {
	if i+1 >= len(lines) || strings.TrimSpace(lines[i+1].Text) == "" {
		return
	}
	report(rule.Finding{
		Rule:     d.Meta().Name,
		Path:     doc.Path,
		Line:     ln.Num,
		Col:      len(ln.Text) + 1,
		Message:  "missing blank line after </summary>; inner markdown will not render",
		Severity: rule.Error,
		Safety:   rule.Safe,
		Fixes: []rule.TextEdit{{
			Start:   ln.End,
			End:     ln.End,
			NewText: "\n",
		}},
	})
}
