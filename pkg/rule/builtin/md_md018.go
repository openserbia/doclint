package builtin

import (
	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

const (
	// maxATXIndent is the most leading spaces an ATX heading may have before it
	// is instead parsed as an indented code block (CommonMark: 0-3 spaces).
	maxATXIndent = 3
	// maxATXHashes is the deepest ATX heading level (######); a run of 7+ '#'
	// is not a heading at all, so a missing space there is not this defect.
	maxATXHashes = 6
)

// NoMissingSpaceATX flags a line that looks like an ATX heading but glues the
// text directly to the hashes ("#Heading"). CommonMark/Goldmark require at
// least one space (or tab) after the '#' run; without it the line is not a
// heading and renders as literal text, so the intended heading is silently
// lost. The fix inserts exactly one space — content-neutral and idempotent.
type NoMissingSpaceATX struct{}

func (NoMissingSpaceATX) Meta() rule.Meta {
	return rule.Meta{
		Name:        "no-missing-space-atx",
		Description: "require a space after the # of an ATX heading so it renders",
		Detail: "An ATX heading is 1-6 '#' characters followed by a space (or tab) " +
			"and the heading text. When the text is glued straight onto the hashes " +
			"(\"#Heading\"), CommonMark and Goldmark (the parser Hugo uses) do not " +
			"recognize a heading at all: the line renders as the literal text " +
			"\"#Heading\" and the heading is silently lost. The fix inserts a single " +
			"space between the hashes and the text, which makes the heading render " +
			"and is idempotent (a spaced heading no longer matches). A digit " +
			"immediately after the hashes (\"#1\") is left alone, since that is " +
			"usually a hashtag/issue reference rather than a heading.",
		Severity: rule.Error,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.Safe,
	}
}

func (r NoMissingSpaceATX) Check(doc *document.Document, report func(rule.Finding)) {
	for _, ln := range doc.Lines {
		// Fence-aware and frontmatter-aware: skip fenced code and any line that
		// begins inside the frontmatter block (before the body starts).
		if ln.InFence || ln.Start < doc.BodyOffset {
			continue
		}
		indent, hashes, ok := atxMissingSpace(ln.Text)
		if !ok {
			continue
		}
		at := ln.Start + indent + hashes
		report(rule.Finding{
			Rule:     r.Meta().Name,
			Path:     doc.Path,
			Line:     ln.Num,
			Col:      indent + hashes + 1,
			Message:  "missing space after # in ATX heading; it will not render as a heading",
			Severity: rule.Error,
			Safety:   rule.Safe,
			Fixes: []rule.TextEdit{{
				Start:   at,
				End:     at,
				NewText: " ",
			}},
		})
	}
}

// atxMissingSpace reports whether text, after up to maxATXIndent leading spaces,
// is a run of 1..maxATXHashes '#' immediately followed by a heading-text byte
// that CommonMark requires to be separated by whitespace. It returns the indent
// width and the number of '#' so the caller can locate the insertion point.
// A digit right after the hashes is treated as a hashtag and not flagged.
func atxMissingSpace(text string) (indent, hashes int, ok bool) {
	i := 0
	for i < len(text) && text[i] == ' ' {
		i++
	}
	indent = i
	if indent > maxATXIndent {
		return 0, 0, false
	}

	for i < len(text) && text[i] == '#' {
		hashes++
		i++
	}
	if hashes == 0 || hashes > maxATXHashes {
		return 0, 0, false
	}

	// No following byte means the line is only hashes (no heading text).
	if i >= len(text) {
		return 0, 0, false
	}

	switch c := text[i]; {
	case c == ' ', c == '\t', c == '#':
		return 0, 0, false // already a valid marker, or still all hashes
	case c >= '0' && c <= '9':
		return 0, 0, false // "#1"/"#2026" hashtag, not a heading — suppress
	default:
		return indent, hashes, true
	}
}
