package engine

import (
	"bytes"
	"strings"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

// maxConsecutiveBlanks is the number of consecutive blank lines that Format
// permits outside fenced code blocks; any run longer than this is collapsed.
const maxConsecutiveBlanks = 1

// Format applies the deterministic, idempotent whitespace pass: it collapses
// 3+ consecutive blank lines (outside fenced code) to one, ensures a single
// trailing newline, applies the always-safe structural fixes, and aligns the
// columns of well-formed GFM tables. The structural fixes include
// no-trailing-spaces' targeted strip of a single stray trailing space and of a
// whitespace-only line; a deliberate two-space hard line break is never touched
// (the rule does not flag it) and an ambiguous 3+ run is left for a human.
func Format(doc *document.Document) []byte {
	// 1. Apply the safe structural fix(es) first, on the raw bytes.
	fixes := safeStructuralFixes(doc)
	raw := doc.Raw
	// Drop redundant same-boundary blank insertions before applying (e.g. a
	// heading's below-blank and the following list's before-blank), so the edits
	// are correct without relying on the later blank-run collapse.
	fixes = coalesceBlankInserts(raw, fixes)
	if len(fixes) > 0 {
		if out, err := ApplyEdits(raw, fixes); err == nil {
			raw = out
		}
	}

	// 2. Re-split (offsets changed) and collapse blank runs outside fences.
	lines := document.SplitLines(raw)
	var b bytes.Buffer
	blankRun := 0
	for _, ln := range lines {
		blank := strings.TrimSpace(ln.Text) == ""
		if blank && !ln.InFence {
			blankRun++
			if blankRun > maxConsecutiveBlanks {
				continue // keep at most one blank line
			}
		} else {
			blankRun = 0
		}
		b.WriteString(ln.Text)
		b.WriteByte('\n')
	}

	// 3. Single trailing newline.
	out := bytes.TrimRight(b.Bytes(), "\n")
	out = append(out, '\n')

	// 4. Align well-formed GFM tables (idempotent; malformed tables untouched).
	out = formatTables(out)
	return out
}

// safeStructuralFixes collects the always-safe, content-neutral fixes that fmt
// applies as part of normalization (intentional engine→builtin coupling, see
// design spec §8): details-blank-line makes inner markdown render,
// no-missing-space-atx inserts the single space that makes a glued "#Heading"
// render as a heading, heading-start-left dedents an indented ATX heading
// back to the left margin (only when it is top-level, never when it is nested in
// a list — the rule withholds the fix there), and blanks-around-fences inserts a
// blank line where a fenced code block is butted against adjacent prose. Every
// rule is fence- and frontmatter-aware via Check, and each fix is idempotent, so
// the resulting pass stays idempotent. The fixes never overlap (a glued
// "#Heading" line and a spaced indented heading line are mutually exclusive,
// dedent edits sit at the line start, and the blank-insertion edits sit at a
// fence line's start/end — distinct offsets), so ApplyEdits accepts them. When a
// closing fence is glued to the next opening fence, the two adjacent insertions
// produce a double blank that the subsequent blank-run collapse reduces to one.
//
// blanks-around-lists adds the same kind of blank-insertion where a list block is
// butted against an adjacent paragraph (so the list is not swallowed as a lazy
// continuation). Its edits sit at a list-edge line's start/end, distinct from the
// fence/heading offsets; when it and blanks-around-fences both insert at a
// list↔fence boundary the doubled blank is collapsed by the blank-run pass too. It
// is frontmatter-aware (it skips frontmatter so a YAML list is never touched).
//
// blanks-around-headings adds the same blank-insertion around an ATX or setext
// heading that is butted against adjacent content. Its below-blank sits at a
// heading/underline line's End and its above-blank at the previous line's End —
// the latter deliberately chosen so it never coincides with heading-start-left's
// dedent edit at the heading line's Start, so the two always-safe fixes never
// produce a spurious ApplyEdits overlap. Where it inserts at the same boundary as
// another blanks-around-* fix the doubled blank is collapsed by the blank-run pass.
func safeStructuralFixes(doc *document.Document) []rule.TextEdit {
	var fixes []rule.TextEdit
	collect := func(f rule.Finding) {
		if f.Safety == rule.Safe {
			fixes = append(fixes, f.Fixes...)
		}
	}
	(builtin.DetailsBlankLine{}).Check(doc, collect)
	(builtin.NoMissingSpaceATX{}).Check(doc, collect)
	(builtin.HeadingStartLeft{}).Check(doc, collect)
	(builtin.BlanksAroundFences{}).Check(doc, collect)
	(builtin.BlanksAroundLists{}).Check(doc, collect)
	(builtin.BlanksAroundHeadings{}).Check(doc, collect)
	(builtin.NoTrailingSpaces{}).Check(doc, collect)
	return fixes
}
