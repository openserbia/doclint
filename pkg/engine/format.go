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
// columns of well-formed GFM tables. Trailing-whitespace stripping is
// intentionally omitted (two trailing spaces are a markdown hard line break).
func Format(doc *document.Document) []byte {
	// 1. Apply the safe structural fix(es) first, on the raw bytes.
	fixes := safeStructuralFixes(doc)
	raw := doc.Raw
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
// design spec §8): details-blank-line makes inner markdown render, and
// no-missing-space-atx inserts the single space that makes a glued "#Heading"
// render as a heading. Both rules are fence- and frontmatter-aware via Check,
// and each fix is idempotent, so the resulting pass stays idempotent. The fixes
// never overlap (different lines / positions), so ApplyEdits accepts them.
func safeStructuralFixes(doc *document.Document) []rule.TextEdit {
	var fixes []rule.TextEdit
	collect := func(f rule.Finding) {
		if f.Safety == rule.Safe {
			fixes = append(fixes, f.Fixes...)
		}
	}
	(builtin.DetailsBlankLine{}).Check(doc, collect)
	(builtin.NoMissingSpaceATX{}).Check(doc, collect)
	return fixes
}
