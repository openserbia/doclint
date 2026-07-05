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
//
// IndentWidth is the number of spaces added per shortcode nesting level inside a
// list-inline block (0 → the default of 2). Exclude names shortcodes whose whole
// subtree (opener through matching closer) is emitted verbatim — useful for
// shortcodes with carefully hand-aligned internals (e.g. uplatnica payment forms).
type ShortcodeIndentPass struct {
	IndentWidth int
	Exclude     map[string]bool
}

func (ShortcodeIndentPass) Name() string { return "shortcode-indent" }
func (p ShortcodeIndentPass) Apply(src []byte) []byte {
	return formatShortcodeIndent(src, p.IndentWidth, p.Exclude)
}

// scDefaultIndentWidth is the per-nesting-level indent used when IndentWidth is
// left at its zero value.
const scDefaultIndentWidth = 2

// formatShortcodeIndent fixes Hugo shortcode indentation inside list items.
// It is idempotent: running it twice produces the same output as running it once.
//
// Standalone shortcodes (not inside a list item's inline block) are emitted
// verbatim — no tree-based re-indentation is applied. The pass only re-indents
// when a block shortcode opens inline in a list item (e.g. "1. {{< details >}}"):
// the closer and any inner pure-tag shortcode lines are aligned to the item's
// actual continuation indent — the leading whitespace of the item's first direct
// prose line (see scListContentCol) — plus a depth-based offset for nested block
// shortcodes. When the item has no direct prose (only shortcode tags), the
// list-continuation column (len(leading) + len(marker) + 1) is used as the base.
// Aligning to sibling prose keeps a tag from being pushed past content the author
// already positioned (e.g. a col-0 block under a "10. " marker stays at col 0).
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
func formatShortcodeIndent(src []byte, indentWidth int, exclude map[string]bool) []byte {
	if indentWidth <= 0 {
		indentWidth = scDefaultIndentWidth
	}
	lines := document.SplitLines(src)
	s := &scState{
		lines: lines,
		// Pass 1: collect every shortcode name that has an explicit closing tag.
		// Only those names are block openers; all others are single tags.
		closedNames: scCollectClosedNames(lines),
		indentWidth: indentWidth,
	}
	// Pre-scan: mark every line inside an excluded shortcode's subtree (opener
	// through matching closer). Those lines are emitted verbatim, like fences.
	s.excluded = scExcludedLineSet(lines, exclude, s.closedNames)

	for i := range lines {
		s.step(i)
	}
	return s.out.Bytes()
}

// scState carries the mutable state of a single formatShortcodeIndent run. Only
// shortcodes opened inline in a list item are re-indented; everything else is
// emitted verbatim, with depth tracked internally so closers match their openers.
type scState struct {
	out              bytes.Buffer
	lines            []document.Line
	closedNames      map[string]bool
	excluded         []bool
	indentWidth      int
	depth            int
	inMultilineTag   bool
	multilineIsBlock bool
	// listIndentStack holds the continuation base column for each active
	// list-inline block; non-empty means we are inside one.
	listIndentStack []int
}

func (s *scState) inList() bool { return len(s.listIndentStack) > 0 }

// indent returns the indentation string for a re-indented tag inside a
// list-inline block: the innermost block's base column plus depth-based offset.
func (s *scState) indent() string {
	if n := len(s.listIndentStack); n > 0 {
		return strings.Repeat(" ", s.listIndentStack[n-1]+s.indentWidth*s.depth)
	}
	return ""
}

func (s *scState) emitVerbatim(text string) {
	s.out.WriteString(text)
	s.out.WriteByte('\n')
}

func (s *scState) emitIndented(prefix, body string) {
	s.out.WriteString(prefix)
	s.out.WriteString(body)
	s.out.WriteByte('\n')
}

// step processes line i, dispatching to the verbatim, multi-line, list-opener,
// standalone, or list-inline handler.
func (s *scState) step(i int) {
	ln := s.lines[i]
	// Fence interiors and excluded-shortcode subtrees are emitted verbatim.
	if ln.InFence || s.excluded[i] {
		s.emitVerbatim(ln.Text)
		return
	}
	t := strings.TrimSpace(ln.Text)

	// Inside a multi-line opener ({{< tag\n...\n>}}) — pass lines through until
	// the closer, then update depth if the opener is a block opener.
	if s.inMultilineTag {
		s.emitVerbatim(ln.Text)
		s.inMultilineTag, s.depth = scHandleMultilineContinuation(t, s.multilineIsBlock, s.depth)
		return
	}

	// Not a pure tag line — emit verbatim, but note an inline block-opener in a
	// list item so its subtree can be aligned to the item's continuation column.
	if !scIsPureTagLine(t) {
		s.emitVerbatim(ln.Text)
		if col := scInlineListOpenerCol(ln.Text, s.closedNames); col > 0 {
			s.listIndentStack = append(s.listIndentStack, scListContentCol(s.lines, i, col, s.closedNames))
		}
		return
	}

	if s.inList() {
		s.stepInList(ln.Text, t)
	} else {
		s.stepStandalone(ln.Text, t)
	}
}

// stepStandalone handles a pure-tag line outside any list-inline block: emit it
// verbatim, only tracking depth so a later list closer isn't misread.
func (s *scState) stepStandalone(text, t string) {
	s.emitVerbatim(text)
	switch scClassify(t) {
	case scSelfContained, scSelfClosing:
		// net-zero depth
	case scCloser:
		if s.depth > 0 {
			s.depth--
		}
	case scMultilineOpener:
		s.inMultilineTag = true
		s.multilineIsBlock = s.closedNames[scTagName(t)]
	case scOpener:
		if s.closedNames[scTagName(t)] {
			s.depth++
		}
	}
}

// stepInList handles a pure-tag line inside a list-inline block: re-indent it to
// the continuation base column plus depth-based offset.
func (s *scState) stepInList(text, t string) {
	switch scClassify(t) {
	case scSelfContained, scSelfClosing:
		s.emitIndented(s.indent(), t)
	case scCloser:
		if s.depth == 0 {
			// Closer matching the list-inline opener: pop and align to its base.
			n := len(s.listIndentStack)
			col := s.listIndentStack[n-1]
			s.listIndentStack = s.listIndentStack[:n-1]
			s.emitIndented(strings.Repeat(" ", col), t)
		} else {
			s.depth--
			s.emitIndented(s.indent(), t)
		}
	case scMultilineOpener:
		s.emitVerbatim(text) // params span lines; re-indenting would misalign them
		s.inMultilineTag = true
		s.multilineIsBlock = s.closedNames[scTagName(t)]
	case scOpener:
		s.emitIndented(s.indent(), t)
		if s.closedNames[scTagName(t)] {
			s.depth++
		}
	}
}

// scTagKind classifies a pure shortcode tag line by its structural effect.
type scTagKind int

const (
	scSelfContained   scTagKind = iota // opens and closes on the same line (net zero)
	scCloser                           // "{{< /name >}}"
	scSelfClosing                      // "{{< name />}}"
	scMultilineOpener                  // opener whose ">}}"/"%}}" is on a later line
	scOpener                           // single-line opener "{{< name … >}}"
)

// scHasTagClose reports whether a trimmed line contains a shortcode tag
// terminator (">}}" or "%}}") anywhere — meaning a tag opened on this line also
// closes on it. Its ABSENCE is what marks a multi-line opener. A plain suffix
// check is wrong for a complete tag with trailing content, e.g.
// "{{<figure … >}}</center>", which ends in "</center>" yet is not multi-line.
func scHasTagClose(trimmed string) bool {
	return strings.Contains(trimmed, ">}}") || strings.Contains(trimmed, "%}}")
}

// scClassify categorizes a pure shortcode tag line (scIsPureTagLine == true).
func scClassify(t string) scTagKind {
	switch {
	case scIsSelfContained(t):
		return scSelfContained
	case scIsCloser(t):
		return scCloser
	case scIsSelfClosing(t):
		return scSelfClosing
	case !scHasTagClose(t):
		return scMultilineOpener
	default:
		return scOpener
	}
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

// scIsSelfContained reports whether a pure-tag line both opens and closes the
// same block on a single line, giving it a net-zero effect on nesting depth.
// Examples:
//
//	"{{< alert >}}text{{< /alert >}}" → true  (text between open and close)
//	"{{< tabs >}}{{< /tabs >}}"       → true  (adjacent open and close)
//	"{{< details \"…{{< x >}}…\" >}}" → false (inner tag is a quoted param)
//
// It is name-aware: it only reports true when a closer for the SAME name as the
// opener appears later on the line, so a line with two distinct openers is not
// mistaken for net-zero.
func scIsSelfContained(trimmed string) bool {
	if scIsCloser(trimmed) {
		return false
	}
	name := scTagName(trimmed)
	if name == "" {
		return false
	}
	return strings.Contains(trimmed, "{{< /"+name) || strings.Contains(trimmed, "{{% /"+name)
}

// scLeadingSpaces returns the number of leading space characters in s.
func scLeadingSpaces(s string) int {
	return len(s) - len(strings.TrimLeft(s, " "))
}

// scExcludedLineSet marks every line that belongs to an excluded shortcode's
// subtree — from the opener (single- or multi-line) through its matching closer,
// inclusive, and everything nested in between. The main pass emits those lines
// verbatim, so an excluded shortcode (e.g. "uplatnica") keeps whatever internal
// alignment the author gave it. Because each subtree is balanced, skipping it
// wholesale leaves the enclosing block's nesting depth undisturbed.
//
// Names are matched via a stack of open block shortcodes; the active exclusion
// ends when the stack unwinds back to the depth at which it began.
//
//nolint:cyclop // linear tag scanner: complexity ≈ number of tag kinds handled; splitting fragments the state machine
func scExcludedLineSet(lines []document.Line, exclude, closedNames map[string]bool) []bool {
	marked := make([]bool, len(lines))
	if len(exclude) == 0 {
		return marked
	}
	var stack []string // names of currently-open block shortcodes
	excludedFrom := -1 // stack depth where the active excluded subtree began; -1 = none
	inMulti := false
	multiName := ""
	multiStart := -1

	for i := range lines {
		ln := lines[i]
		if ln.InFence {
			marked[i] = excludedFrom >= 0
			continue
		}
		t := strings.TrimSpace(ln.Text)

		if inMulti {
			marked[i] = excludedFrom >= 0
			if stillOpen, _ := scHandleMultilineContinuation(t, false, 0); !stillOpen {
				inMulti = false
				if !strings.HasSuffix(t, "/>}}") && closedNames[multiName] {
					stack = append(stack, multiName)
					if excludedFrom < 0 && exclude[multiName] {
						excludedFrom = len(stack) - 1
						for k := multiStart; k <= i; k++ {
							marked[k] = true
						}
					}
				}
			}
			continue
		}

		if !scIsPureTagLine(t) {
			marked[i] = excludedFrom >= 0
			continue
		}

		switch scClassify(t) {
		case scSelfContained:
			marked[i] = excludedFrom >= 0 || exclude[scTagName(t)]
		case scSelfClosing:
			marked[i] = excludedFrom >= 0
		case scCloser:
			marked[i] = excludedFrom >= 0
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
				if excludedFrom >= 0 && len(stack) == excludedFrom {
					excludedFrom = -1
				}
			}
		case scMultilineOpener:
			marked[i] = excludedFrom >= 0
			inMulti = true
			multiName = scTagName(t)
			multiStart = i
		case scOpener:
			name := scTagName(t)
			if closedNames[name] {
				stack = append(stack, name)
				if excludedFrom < 0 && exclude[name] {
					excludedFrom = len(stack) - 1
				}
			}
			marked[i] = excludedFrom >= 0
		}
	}
	return marked
}

// scListContentCol returns the column that shortcode tags directly inside a
// list-inline block (opened at lines[openerIdx]) should align to. It is the
// leading indent of the list item's first direct continuation prose line — the
// sibling content the author already positioned — so re-indented tags line up
// with it rather than with a theoretical marker-width column. When the item has
// no direct prose (only shortcode tags), it falls back to markerCol.
//
// "Direct" means at the block's top level: prose nested inside a child shortcode
// (localDepth > 0) is ignored. The scan stops at the block's matching closer.
//
//nolint:cyclop // linear tag scanner: complexity ≈ number of tag kinds handled; splitting fragments the state machine
func scListContentCol(lines []document.Line, openerIdx, markerCol int, closedNames map[string]bool) int {
	localDepth := 0
	inMulti := false
	multiBlock := false
	for j := openerIdx + 1; j < len(lines); j++ {
		ln := lines[j]
		if ln.InFence {
			continue
		}
		t := strings.TrimSpace(ln.Text)

		if inMulti {
			stillOpen, _ := scHandleMultilineContinuation(t, false, 0)
			if !stillOpen {
				inMulti = false
				if multiBlock && !strings.HasSuffix(t, "/>}}") {
					localDepth++
				}
			}
			continue
		}

		if t == "" {
			continue
		}
		if !scIsPureTagLine(t) {
			if localDepth == 0 {
				// First direct continuation prose of the list item.
				return scLeadingSpaces(ln.Text)
			}
			continue // nested content — not a direct sibling
		}
		switch scClassify(t) {
		case scSelfContained, scSelfClosing:
			// net-zero depth
		case scCloser:
			if localDepth == 0 {
				return markerCol // block's matching closer; no direct prose found
			}
			localDepth--
		case scMultilineOpener:
			inMulti = true
			multiBlock = closedNames[scTagName(t)]
		case scOpener:
			if closedNames[scTagName(t)] {
				localDepth++
			}
		}
	}
	return markerCol
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
