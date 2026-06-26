package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

const (
	docsDirMode  = 0o755
	docsFileMode = 0o644
)

// newDocsCmd generates one reference page per built-in rule from its metadata,
// so the doc URLs printed by lint/explain resolve to real pages. It is a
// maintenance command (run in CI / before a release), hence hidden.
func newDocsCmd(opts *Options) *cobra.Command {
	var outDir string
	cmd := &cobra.Command{
		Use:    "docs",
		Short:  "Generate per-rule reference pages from rule metadata",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			reg := rule.NewRegistry()
			builtin.Register(reg)
			if err := os.MkdirAll(outDir, docsDirMode); err != nil {
				return err
			}
			rules := reg.All()
			for _, r := range rules {
				m := r.Meta()
				out := filepath.Join(outDir, m.Name+".md")
				if err := os.WriteFile(out, []byte(rulePage(m)), docsFileMode); err != nil { //nolint:gosec // generated docs
					return err
				}
			}
			if err := writeReadmeTable(rules); err != nil {
				return err
			}
			u := newUI(cmd.OutOrStdout(), opts.NoColor)
			u.ok(fmt.Sprintf("wrote %d rule %s", len(rules), plural(len(rules), "page")))
			u.item(outDir)
			return u.Err()
		},
	}
	cmd.Flags().StringVar(&outDir, "out", filepath.Join("docs", "rules"), "output directory for the rule pages")
	return cmd
}

func rulePage(m rule.Meta) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n`%s`\n\n> %s\n\n- **Default severity:** %s\n- **Fix:** %s\n\n## How to fix\n\n%s\n",
		ruleTitle(m), m.Name, m.Description, m.Severity, ruleFixLabel(m.Safety), m.Detail)
	if m.Example.Bad != "" {
		fmt.Fprintf(&b, "\n## Example\n\nFlagged:\n\n```markdown\n%s\n```\n\nFixed:\n\n```markdown\n%s\n```\n",
			m.Example.Bad, m.Example.Good)
	}
	b.WriteString("\n---\n\n_Generated from `doclint` rule metadata — run `doclint docs` to refresh; do not edit by hand._\n")
	return b.String()
}

// ruleTitle is the human-readable display name, falling back to the machine name.
func ruleTitle(m rule.Meta) string {
	if m.Title != "" {
		return m.Title
	}
	return m.Name
}

func ruleFixLabel(s rule.FixSafety) string {
	switch s {
	case rule.Safe:
		return "safe autofix, applied by `doclint lint --fix` and `doclint fmt`"
	case rule.Unsafe:
		return "unsafe autofix, applied only with `doclint lint --fix --unsafe-fixes`"
	default:
		return "no automatic fix — surfaced for a human to resolve"
	}
}

// readme markers delimit the generated rule table in README.md.
const (
	rulesStartMark = "<!-- rules:start -->"
	rulesEndMark   = "<!-- rules:end -->"
)

// writeReadmeTable rewrites the rule table between the markers in README.md (in
// the current directory). It is a no-op when there is no README or no markers,
// so running `doclint docs` outside the repo is harmless.
func writeReadmeTable(rules []rule.Rule) error {
	const readmePath = "README.md"
	body, err := os.ReadFile(readmePath)
	if err != nil {
		return nil //nolint:nilerr // no README here — nothing to update
	}
	content := string(body)
	s := strings.Index(content, rulesStartMark)
	e := strings.Index(content, rulesEndMark)
	if s < 0 || e < s {
		return nil // markers absent — leave the file alone
	}
	var t strings.Builder
	t.WriteString(rulesStartMark + "\n\n")
	t.WriteString("| Rule | Severity | Fix | Description |\n|---|---|---|---|\n")
	for _, r := range rules {
		m := r.Meta()
		fmt.Fprintf(&t, "| [%s](docs/rules/%s.md) (`%s`) | %s | %s | %s |\n",
			ruleTitle(m), m.Name, m.Name, m.Severity, fixTag(m.Safety), m.Description)
	}
	t.WriteString("\n" + rulesEndMark)
	out := content[:s] + t.String() + content[e+len(rulesEndMark):]
	return os.WriteFile(readmePath, []byte(out), docsFileMode) //nolint:gosec // repo README
}

func fixTag(s rule.FixSafety) string {
	switch s {
	case rule.Safe:
		return "safe"
	case rule.Unsafe:
		return "unsafe"
	default:
		return "—"
	}
}
