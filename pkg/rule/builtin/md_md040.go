package builtin

import (
	"strings"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

// FencedCodeLanguage flags an opening code fence whose info string is empty — no
// language identifier follows the ``` or ~~~ delimiter (markdownlint MD040). A
// missing language disables Chroma syntax highlighting: the block still renders
// as plain preformatted text, so this is a quality/hygiene issue, not a
// correctness break. The correct language cannot be inferred from the fenced
// content, so the finding is emitted with no automatic fix (NoFix).
type FencedCodeLanguage struct{}

func (FencedCodeLanguage) Meta() rule.Meta {
	return rule.Meta{
		Name:        "fenced-code-language",
		Description: "fenced code blocks should specify a language for syntax highlighting",
		Detail: "A fenced code block (``` or ~~~) whose opening delimiter has an empty " +
			"info string declares no language. Hugo's Chroma highlighter then has " +
			"nothing to highlight, so the block renders as an unstyled plain " +
			"preformatted box. The content is not lost — this is a quality/hygiene " +
			"issue, not a correctness break — but a language tag (```go, ```bash, " +
			"```json …) makes code blocks readable. This rule reports each opening " +
			"fence that omits a language. The correct language cannot be inferred " +
			"from the code, so no automatic fix is offered; add the language by hand. " +
			"Closing delimiters never carry an info string and are ignored.",
		Severity: rule.Warning,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.NoFix,
	}
}

func (r FencedCodeLanguage) Check(doc *document.Document, report func(rule.Finding)) {
	// Toggle `open` on each fence delimiter, mirroring document.SplitLines so this
	// rule's notion of opening vs closing delimiters agrees with Line.InFence. An
	// info string only exists on the OPENING delimiter; closing ones are skipped.
	open := false
	for _, ln := range doc.Lines {
		if !fenceDelimRe.MatchString(ln.Text) {
			continue
		}
		if !open && fenceInfoString(ln.Text) == "" {
			report(r.finding(doc, ln))
		}
		open = !open
	}
}

// fenceInfoString returns the info string of a fence delimiter line: the text
// after the leading whitespace and the run of fence characters (``` or ~~~),
// trimmed of surrounding whitespace. fenceDelimRe guarantees text begins with an
// optional indent followed by at least three identical fence chars.
func fenceInfoString(text string) string {
	rest := strings.TrimLeft(text, " \t")
	if rest == "" {
		return ""
	}
	fenceChar := rest[0] // '`' or '~', guaranteed by fenceDelimRe
	rest = strings.TrimLeft(rest, string(fenceChar))
	return strings.TrimSpace(rest)
}

// finding builds the NoFix Warning for an opening fence missing a language.
func (r FencedCodeLanguage) finding(doc *document.Document, ln document.Line) rule.Finding {
	return rule.Finding{
		Rule:     r.Meta().Name,
		Path:     doc.Path,
		Line:     ln.Num,
		Col:      1,
		Message:  "fenced code block is missing a language",
		Severity: rule.Warning,
		Safety:   rule.NoFix,
	}
}
