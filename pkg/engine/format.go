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
// trailing newline, and applies the always-safe details-blank-line fix.
// Trailing-whitespace stripping is intentionally omitted (two trailing spaces
// are a markdown hard line break).
func Format(doc *document.Document) []byte {
	// 1. Apply the safe structural fix(es) first, on the raw bytes.
	// Intentional engine→builtin coupling: fmt always applies the safe
	// details-blank-line fix as part of normalization (see design spec §8).
	var fixes []rule.TextEdit
	(builtin.DetailsBlankLine{}).Check(doc, func(f rule.Finding) {
		if f.Safety == rule.Safe {
			fixes = append(fixes, f.Fixes...)
		}
	})
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
	return out
}
