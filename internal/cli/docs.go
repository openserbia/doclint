package cli

import (
	"fmt"
	"os"
	"path/filepath"

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
func newDocsCmd() *cobra.Command {
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
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "wrote %d rule pages to %s\n", len(rules), outDir)
			return err
		},
	}
	cmd.Flags().StringVar(&outDir, "out", filepath.Join("docs", "rules"), "output directory for the rule pages")
	return cmd
}

func rulePage(m rule.Meta) string {
	fix := "no automatic fix — surfaced for a human to resolve"
	switch m.Safety {
	case rule.NoFix:
		fix = "no automatic fix — surfaced for a human to resolve"
	case rule.Safe:
		fix = "safe autofix, applied by `doclint lint --fix` and `doclint fmt`"
	case rule.Unsafe:
		fix = "unsafe autofix, applied only with `doclint lint --fix --unsafe-fixes`"
	}
	return fmt.Sprintf(
		"# %s\n\n> %s\n\n- **Default severity:** %s\n- **Fix:** %s\n\n## How to fix\n\n%s\n\n---\n\n_Generated from `doclint` rule metadata — run `doclint docs` to refresh; do not edit by hand._\n",
		m.Name, m.Description, m.Severity, fix, m.Detail,
	)
}
