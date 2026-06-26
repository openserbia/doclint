package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/engine"
)

const fileMode = 0o600

func newFmtCmd(opts *Options) *cobra.Command {
	var (
		check bool
		diff  bool
	)
	cmd := &cobra.Command{
		Use:   "fmt [paths...]",
		Short: "Normalize markdown spacing (idempotent); --check/--diff for dry-run",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, reg, err := loadConfig(opts)
			if err != nil {
				return err
			}
			eng, err := engine.New(cfg, reg)
			if err != nil {
				return err
			}
			files, err := eng.MarkdownFiles(resolveTargets(args, cfg))
			if err != nil {
				return err
			}
			changed, err := processFiles(files, check, diff, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			if err := reportFmt(cmd, opts, changed, check, diff); err != nil {
				return err
			}
			if (check || diff) && len(changed) > 0 {
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "exit non-zero if any file would change")
	cmd.Flags().BoolVar(&diff, "diff", false, "print formatting changes as a diff")
	return cmd
}

func processFiles(files []string, check, diff bool, w io.Writer) ([]string, error) {
	var changed []string
	for _, p := range files {
		ok, err := processFile(p, check, diff, w)
		if err != nil {
			return changed, err
		}
		if ok {
			changed = append(changed, p)
		}
	}
	return changed, nil
}

func processFile(p string, check, diff bool, w io.Writer) (bool, error) {
	raw, err := os.ReadFile(p) //nolint:gosec // discovered path
	if err != nil {
		return false, err
	}
	doc, err := document.ParseMarkdown(p, raw)
	if err != nil {
		return false, err
	}
	out := engine.Format(doc)
	if bytes.Equal(out, raw) {
		return false, nil
	}
	switch {
	case diff:
		if _, err := fmt.Fprint(w, engine.UnifiedDiff(p, raw, out)); err != nil {
			return false, err
		}
	case check:
		// the changed file is listed by reportFmt
	default:
		if err := os.WriteFile(p, out, fileMode); err != nil { //nolint:gosec // discovered path
			return false, err
		}
	}
	return true, nil
}

// reportFmt prints the fmt summary in doclint's common style: under --check each
// changed file is listed, and every mode ends with a status headline.
func reportFmt(cmd *cobra.Command, opts *Options, changed []string, check, diff bool) error {
	if opts.Quiet {
		return nil
	}
	u := newUI(cmd.OutOrStdout(), opts.NoColor)
	n := len(changed)
	if check {
		for _, p := range changed {
			u.item(p)
		}
	}
	switch {
	case (check || diff) && n == 0:
		u.ok("already formatted")
	case check || diff:
		u.warn(fmt.Sprintf("%d %s would change", n, plural(n, "file")))
	case n == 0:
		u.ok("already formatted")
	default:
		u.ok(fmt.Sprintf("formatted %d %s", n, plural(n, "file")))
	}
	return u.Err()
}
