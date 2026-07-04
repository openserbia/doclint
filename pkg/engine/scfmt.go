package engine

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/openserbia/doclint/pkg/document"
)

// scListMarkerRe matches a list-item marker (bullet or ordered) with up to three
// leading spaces, capturing the leading whitespace and the marker to compute the
// content column where list-continuation content should indent to.
var scListMarkerRe = regexp.MustCompile(`^( {0,3})([-*+]|\d{1,9}[.)]) `)

// ShortcodeIndentPass is a FormatPass that fixes shortcode indentation inside
// list items. Standalone shortcodes are emitted verbatim (no tree indentation).
type ShortcodeIndentPass struct{}

func (ShortcodeIndentPass) Name() string            { return "shortcode-indent" }
func (ShortcodeIndentPass) Apply(src []byte) []byte { return formatShortcodeIndent(src) }

const scIndentUnit = "  " // 2 spaces per shortcode nesting level (inside list blocks)

// formatShortcodeIndent fixes Hugo shortcode indentation inside list items.
// It is idempotent: running it twice produces the same output as running it once.
//
// Standalone shortcodes (not inside a list item's inline block) are emitted
// verbatim — no tree-based re-indentation is applied. The pass only re-indents
// when a block shortcode opens inline in a list item (e.g. "1. {{< details >}}"):
// the closer and any inner pure-tag shortcode lines are indented to the
// list-continuation column (len(leading) + len(marker) + 1) plus depth-based
// offset for nested block shortcodes.
//
// A line qualifies as a "pure shortcode tag line" when its trimmed form starts
// with "{{<" or "{{%". This excludes:
//   - Inline shortcodes inside a list item or prose ("- {{< video ... >}}")
//     because TrimSpace starts with "-", not "{{<".
//   - Shortcodes used as link targets ("[text]({{< relref ... >}})")
//     because TrimSpace starts with "[".
//   - Lines inside fenced code blocks (InFence=true).
//
// Depth is tracked internally (via a two-pass approach that collects block-opener
// names) so the pass can correctly match closers and pop the list-indent stack
// at the right time. Multi-line tag parameter blocks are always emitted as-is.
func formatShortcodeIndent(src []byte) []byte {
	lines := document.SplitLines(src)

	// Pass 1: collect every shortcode name that has an explicit closing tag.
	// Only these names are block openers; all others are treated as single tags.
	closedNames := scCollectClosedNames(lines)

	// Pass 2: fix shortcode indentation inside list items only.
	// Standalone shortcodes (not inside a list item) are emitted verbatim —
	// no tree-based re-indentation. Depth is still tracked internally so we
	// can correctly match closers to their openers and pop the list stack at
	// the right time.
	var out bytes.Buffer
	depth := 0
	inMultilineTag := false
	multilineIsBlock := false // whether the current multi-line opener is a block opener

	// listIndentStack tracks list-continuation indents for shortcodes opened
	// inline in list items (e.g. "1. {{< details >}}"). When the stack is
	// non-empty, pure-tag lines are re-indented to the list-continuation
	// column plus depth-based offset. Outside list-inline blocks, lines are
	// emitted verbatim.
	var listIndentStack []int

	// inList reports whether we are inside a list-inline block.
	inList := func() bool { return len(listIndentStack) > 0 }

	// indent returns the indentation string when inside a list-inline block.
	indent := func() string {
		if n := len(listIndentStack); n > 0 {
			return strings.Repeat(" ", listIndentStack[n-1]) + strings.Repeat(scIndentUnit, depth)
		}
		return ""
	}

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
			inMultilineTag, depth = scHandleMultilineContinuation(t, multilineIsBlock, depth)
			continue
		}

		// Not a pure shortcode tag line — emit verbatim, but track inline
		// block-openers in list items so the closer can be indented correctly.
		if !scIsPureTagLine(t) {
			out.WriteString(ln.Text)
			out.WriteByte('\n')
			if col := scInlineListOpenerCol(ln.Text, closedNames); col > 0 {
				listIndentStack = append(listIndentStack, col)
			}
			continue
		}

		// Outside a list-inline block: emit all pure-tag lines verbatim,
		// only tracking depth for structural correctness.
		if !inList() {
			out.WriteString(ln.Text)
			out.WriteByte('\n')
			// Still track depth so we don't confuse a later list-inline
			// closer with a standalone closer.
			if strings.Contains(t, "}}{{") {
				// Compound line: net zero.
			} else if scIsCloser(t) {
				if depth > 0 {
					depth--
				}
			} else if scIsSelfClosing(t) {
				// No depth change.
			} else if !strings.HasSuffix(t, ">}}") && !strings.HasSuffix(t, "%}}") {
				name := scTagName(t)
				inMultilineTag = true
				multilineIsBlock = closedNames[name]
			} else {
				name := scTagName(t)
				if closedNames[name] {
					depth++
				}
			}
			continue
		}

		// Inside a list-inline block: re-indent to list-continuation
		// column plus depth-based offset.

		// Compound line: net depth zero.
		if strings.Contains(t, "}}{{") {
			out.WriteString(indent())
			out.WriteString(t)
			out.WriteByte('\n')
			continue
		}

		switch {
		case scIsCloser(t):
			if depth == 0 {
				// This is the closer that matches the list-inline opener.
				n := len(listIndentStack)
				col := listIndentStack[n-1]
				listIndentStack = listIndentStack[:n-1]
				out.WriteString(strings.Repeat(" ", col))
				out.WriteString(t)
				out.WriteByte('\n')
			} else {
				depth--
				out.WriteString(indent())
				out.WriteString(t)
				out.WriteByte('\n')
			}

		case scIsSelfClosing(t):
			out.WriteString(indent())
			out.WriteString(t)
			out.WriteByte('\n')

		case !strings.HasSuffix(t, ">}}") && !strings.HasSuffix(t, "%}}"):
			// Multi-line opener inside a list block — emit as-is.
			name := scTagName(t)
			out.WriteString(ln.Text)
			out.WriteByte('\n')
			inMultilineTag = true
			multilineIsBlock = closedNames[name]

		default:
			name := scTagName(t)
			out.WriteString(indent())
			out.WriteString(t)
			out.WriteByte('\n')
			if closedNames[name] {
				depth++
			}
		}
	}

	return out.Bytes()
}

// scHandleMultilineContinuation processes a line inside a multi-line shortcode
// tag parameter block. Returns (stillInMultiline, updatedDepth).
func scHandleMultilineContinuation(trimmed string, isBlock bool, depth int) (stillOpen bool, newDepth int) {
	switch {
	case strings.HasSuffix(trimmed, "/>}}"):
		// Explicit self-closing — depth unchanged regardless.
		return false, depth
	case strings.HasSuffix(trimmed, ">}}") || strings.HasSuffix(trimmed, "%}}"):
		if isBlock {
			depth++
		}
		return false, depth
	default:
		return true, depth
	}
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

// scInlineListOpenerCol detects a block-opener shortcode embedded inline in a
// list item (e.g. "1. {{< details >}}") and returns the list-continuation
// indent column (len(leading) + len(marker) + 1). Returns 0 if the line is not
// a list item or does not contain a block-opener shortcode.
func scInlineListOpenerCol(text string, closedNames map[string]bool) int {
	m := scListMarkerRe.FindStringSubmatch(text)
	if m == nil {
		return 0
	}
	// Check whether this line contains an opening shortcode (angle or percent).
	scIdx := strings.Index(text, "{{< ")
	if scIdx < 0 {
		scIdx = strings.Index(text, "{{% ")
	}
	if scIdx < 0 {
		scIdx = strings.Index(text, "{{<")
	}
	if scIdx < 0 {
		scIdx = strings.Index(text, "{{%")
	}
	if scIdx < 0 {
		return 0
	}
	// Extract the shortcode name and check that it is a block opener.
	scPart := strings.TrimSpace(text[scIdx:])
	if scIsCloser(scPart) {
		return 0
	}
	name := scTagName(scPart)
	if name == "" || !closedNames[name] {
		return 0
	}
	// Also skip if the same line also closes this shortcode (compound line).
	if strings.Contains(text[scIdx:], "{{< /"+name) || strings.Contains(text[scIdx:], "{{% /"+name) {
		return 0
	}
	return len(m[1]) + len(m[2]) + 1
}
