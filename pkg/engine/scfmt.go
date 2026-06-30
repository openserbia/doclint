package engine

import (
	"bytes"
	"strings"

	"github.com/openserbia/doclint/pkg/document"
)

// ShortcodeIndentPass is a FormatPass that re-indents Hugo shortcode tag lines
// based on their nesting depth. Content lines between tags are never modified.
type ShortcodeIndentPass struct{}

func (ShortcodeIndentPass) Name() string            { return "shortcode-indent" }
func (ShortcodeIndentPass) Apply(src []byte) []byte { return formatShortcodeIndent(src) }

const scIndentUnit = "  " // 2 spaces per shortcode nesting level

// formatShortcodeIndent re-indents Hugo shortcode opening/closing-tag lines
// based on nesting depth. It is idempotent: running it twice produces the same
// output as running it once.
//
// A line qualifies as a "pure shortcode tag line" — eligible for re-indentation
// — when its trimmed form starts with "{{<" or "{{%". This excludes:
//   - Inline shortcodes inside a list item or prose ("- {{< video ... >}}")
//     because TrimSpace starts with "-", not "{{<".
//   - Shortcodes used as link targets ("[text]({{< relref ... >}})")
//     because TrimSpace starts with "[".
//   - Lines inside fenced code blocks (InFence=true).
//
// Content lines between tags — prose, list items, blank lines — are never
// modified. Markdown is indentation-sensitive (4+ leading spaces = code block),
// and content inside shortcodes often carries indentation that reflects the
// surrounding markdown structure (e.g. list-continuation indent). Re-indenting
// prose would silently corrupt rendering.
//
// Depth tracking uses a two-pass approach. The first pass collects every
// shortcode name that has an explicit closing tag ("{{< /name >}}") anywhere in
// the document. Only those names are treated as block openers that increase
// depth; all others (e.g. figure, video, link-card) are treated as self-contained
// single tags regardless of whether they use the self-closing "/>}}" syntax. This
// prevents single-tag shortcodes with multi-line parameter blocks from
// incorrectly inflating the depth counter.
//
// Multi-line tag parameter blocks (e.g. "{{< uplatnica\nkey="v"\n>}}") are
// emitted as-is: the opener and its continuation lines are left untouched, but
// the depth counter is still updated when the closing ">}}" line is reached (if
// the opener's name is a block opener), so that children and the closer are
// placed correctly.
func formatShortcodeIndent(src []byte) []byte {
	lines := document.SplitLines(src)

	// Pass 1: collect every shortcode name that has an explicit closing tag.
	// Only these names are block openers; all others are treated as single tags.
	closedNames := scCollectClosedNames(lines)

	// Pass 2: re-indent pure shortcode tag lines.
	var out bytes.Buffer
	depth := 0
	inMultilineTag := false
	multilineIsBlock := false // whether the current multi-line opener is a block opener

	for _, ln := range lines {
		// Fence interiors are always emitted verbatim.
		if ln.InFence {
			out.WriteString(ln.Text)
			out.WriteByte('\n')
			continue
		}

		t := strings.TrimSpace(ln.Text)

		// Inside a multi-line opener ({{< tag\n...params...\n>}}) — pass every
		// line through unchanged until the closing >}} (or />}} for self-closing),
		// then update depth if the opener is a block opener.
		if inMultilineTag {
			out.WriteString(ln.Text)
			out.WriteByte('\n')
			switch {
			case strings.HasSuffix(t, "/>}}"):
				// Explicit self-closing — depth unchanged regardless.
				inMultilineTag = false
			case strings.HasSuffix(t, ">}}") || strings.HasSuffix(t, "%}}"):
				inMultilineTag = false
				if multilineIsBlock {
					depth++
				}
			}
			continue
		}

		// Not a pure shortcode tag line — emit verbatim.
		if !scIsPureTagLine(t) {
			out.WriteString(ln.Text)
			out.WriteByte('\n')
			continue
		}

		// Compound line: opener immediately followed by a closer on the same line,
		// e.g. "{{< uf-field slot="sifra" >}}{{< /uf-field >}}".
		// Net depth change is zero; re-indent to current depth only.
		if strings.Contains(t, "}}{{") {
			out.WriteString(strings.Repeat(scIndentUnit, depth))
			out.WriteString(t)
			out.WriteByte('\n')
			continue
		}

		switch {
		case scIsCloser(t):
			depth = max(0, depth-1)
			out.WriteString(strings.Repeat(scIndentUnit, depth))
			out.WriteString(t)
			out.WriteByte('\n')

		case scIsSelfClosing(t):
			// Explicit "/>}}" — never increments depth.
			out.WriteString(strings.Repeat(scIndentUnit, depth))
			out.WriteString(t)
			out.WriteByte('\n')

		case !strings.HasSuffix(t, ">}}") && !strings.HasSuffix(t, "%}}"):
			// Multi-line opener: "{{<" starts the line but ">}}" is not on it yet.
			// Emit the first line as-is (re-indenting here would misalign the
			// parameter continuation lines), but remember whether this opener is a
			// block opener so depth can be updated when we see the closing ">}}".
			name := scTagName(t)
			out.WriteString(ln.Text)
			out.WriteByte('\n')
			inMultilineTag = true
			multilineIsBlock = closedNames[name]

		default:
			// Single-line opener: "{{< tag ... >}}"
			// Re-indent to current depth; increment depth only for block openers.
			name := scTagName(t)
			out.WriteString(strings.Repeat(scIndentUnit, depth))
			out.WriteString(t)
			out.WriteByte('\n')
			if closedNames[name] {
				depth++
			}
		}
	}

	return out.Bytes()
}

// scCollectClosedNames returns the set of shortcode names that have at least one
// explicit closing tag ("{{< /name >}}" or "{{% /name %}}") anywhere in lines.
// Fenced-code-block lines are skipped.
func scCollectClosedNames(lines []document.Line) map[string]bool {
	names := map[string]bool{}
	for _, ln := range lines {
		if ln.InFence {
			continue
		}
		t := strings.TrimSpace(ln.Text)
		if !scIsCloser(t) {
			continue
		}
		if name := scTagName(t); name != "" {
			names[name] = true
		}
	}
	return names
}

// scTagName extracts the shortcode name from a trimmed opener or closer line.
// Examples:
//
//	"{{< uplatnica-form amount=\"400\" >}}" → "uplatnica-form"
//	"{{< /uplatnica-form >}}"              → "uplatnica-form"
//	"{{<figure src=\"x.png\" >}}"          → "figure"
func scTagName(trimmed string) string {
	// Strip leading delimiter.
	s := trimmed
	switch {
	case strings.HasPrefix(s, "{{< "):
		s = s[4:]
	case strings.HasPrefix(s, "{{% "):
		s = s[4:]
	case strings.HasPrefix(s, "{{<"):
		s = s[3:]
	case strings.HasPrefix(s, "{{%"):
		s = s[3:]
	default:
		return ""
	}
	// Strip optional closer slash.
	s = strings.TrimLeft(s, " \t")
	s = strings.TrimPrefix(s, "/")
	s = strings.TrimLeft(s, " \t")
	// Name ends at the first whitespace, /, >, %, or }.
	if idx := strings.IndexAny(s, " \t/>%}"); idx >= 0 {
		return s[:idx]
	}
	return s
}

// scIsPureTagLine reports whether a trimmed line is unambiguously a shortcode
// tag (no non-whitespace content precedes "{{<" or "{{%" on the source line).
func scIsPureTagLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "{{<") || strings.HasPrefix(trimmed, "{{%")
}

// scIsCloser reports whether a trimmed shortcode tag line is a closing tag
// ("{{< /name >}}" or "{{% /name %}}").
func scIsCloser(trimmed string) bool {
	if strings.HasPrefix(trimmed, "{{< /") || strings.HasPrefix(trimmed, "{{% /") {
		return true
	}
	// No-space variant: "{{</name>}}" or "{{%/name%}}"
	return len(trimmed) > 3 &&
		(strings.HasPrefix(trimmed, "{{<") || strings.HasPrefix(trimmed, "{{%")) &&
		trimmed[3] == '/'
}

// scIsSelfClosing reports whether a trimmed shortcode tag line is explicitly
// self-closing ("{{< tag />}}").
func scIsSelfClosing(trimmed string) bool {
	return strings.HasSuffix(trimmed, "/>}}")
}
