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
			if (check || diff) && changed > 0 {
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "exit non-zero if any file would change")
	cmd.Flags().BoolVar(&diff, "diff", false, "print formatting changes as a diff")
	return cmd
}

func processFiles(files []string, check, diff bool, w io.Writer) (int, error) {
	changed := 0
	for _, p := range files {
		n, err := processFile(p, check, diff, w)
		if err != nil {
			return changed, err
		}
		changed += n
	}
	return changed, nil
}

func processFile(p string, check, diff bool, w io.Writer) (int, error) {
	raw, err := os.ReadFile(p) //nolint:gosec // discovered path
	if err != nil {
		return 0, err
	}
	doc, err := document.ParseMarkdown(p, raw)
	if err != nil {
		return 0, err
	}
	out := engine.Format(doc)
	if bytes.Equal(out, raw) {
		return 0, nil
	}
	if err := writeFmtOutput(p, raw, out, check, diff, w); err != nil {
		return 0, err
	}
	return 1, nil
}

func writeFmtOutput(p string, raw, out []byte, check, diff bool, w io.Writer) error {
	switch {
	case diff:
		_, err := fmt.Fprint(w, engine.UnifiedDiff(p, raw, out))
		return err
	case check:
		_, err := fmt.Fprintf(w, "would reformat %s\n", p)
		return err
	default:
		return os.WriteFile(p, out, fileMode) //nolint:gosec // discovered path
	}
}
