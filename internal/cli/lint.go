package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/openserbia/doclint/pkg/engine"
	"github.com/openserbia/doclint/pkg/report"
	"github.com/openserbia/doclint/pkg/rule"
)

func newLintCmd(opts *Options) *cobra.Command {
	var (
		fix         bool
		unsafeFixes bool
		diff        bool
		maxWarn     int
	)
	cmd := &cobra.Command{
		Use:   "lint [paths...]",
		Short: "Report findings; --fix applies safe fixes, --diff lists files that would change",
		RunE: func(cmd *cobra.Command, args []string) error {
			if unsafeFixes {
				fix = true
			}
			cfg, reg, err := loadConfig(opts)
			if err != nil {
				return err
			}
			eng, err := engine.New(cfg, reg)
			if err != nil {
				return err
			}
			ctx := context.Background()

			if fix || diff {
				changed, err := eng.Fix(ctx, args, unsafeFixes, diff)
				if err != nil {
					return err
				}
				if !opts.Quiet {
					for _, p := range changed {
						if _, err := fmt.Fprintln(cmd.OutOrStdout(), p); err != nil {
							return err
						}
					}
					verb := "fixed"
					if diff {
						verb = "would change"
					}
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%d file(s) %s\n", len(changed), verb); err != nil {
						return err
					}
				}
				return nil
			}

			res, err := eng.Run(ctx, args)
			if err != nil {
				return err
			}
			if err := pickReporter(opts, cmd.Flags().Changed("format")).Report(cmd.OutOrStdout(), res.Findings); err != nil {
				return err
			}
			if shouldFail(res, maxWarn) {
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "apply safe fixes in place")
	cmd.Flags().BoolVar(&unsafeFixes, "unsafe-fixes", false, "also apply unsafe fixes (implies --fix)")
	cmd.Flags().BoolVar(&diff, "diff", false, "list files whose fixes would change them, without writing")
	cmd.Flags().IntVar(&maxWarn, "max-warnings", -1, "fail if warnings exceed N (-1 = never)")
	return cmd
}

func shouldFail(res *engine.Result, maxWarn int) bool {
	if res.ExitCode() != 0 {
		return true
	}
	if maxWarn < 0 {
		return false
	}
	warns := 0
	for _, f := range res.Findings {
		if f.Severity == rule.Warning {
			warns++
		}
	}
	return warns > maxWarn
}

func pickReporter(opts *Options, formatSet bool) report.Reporter {
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	// Never emit ANSI to something that isn't a terminal — keeps piped/CI logs
	// and redirected files free of escape codes.
	noColor := opts.NoColor || !isTTY
	switch opts.Format {
	case "json":
		return report.JSON{}
	case "compact":
		return report.Compact{NoColor: noColor}
	default:
		// The default human format auto-falls back to the flat compact format
		// when stdout is not an interactive terminal (pipes, CI, IDE log
		// capture) so machine consumers and grep stay happy. An explicit
		// --format human is always honored (but still color-free off-terminal).
		if !formatSet && !isTTY {
			return report.Compact{NoColor: noColor}
		}
		return report.Human{NoColor: noColor}
	}
}
