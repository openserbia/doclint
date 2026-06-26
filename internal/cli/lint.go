package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/openserbia/doclint/pkg/cache"
	"github.com/openserbia/doclint/pkg/config"
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
		noCache     bool
		cacheDir    string
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
			if fix || diff {
				return runFix(cmd, eng, opts, args, unsafeFixes, diff)
			}
			return runLint(cmd, opts, eng, cfg, args, noCache, cacheDir, maxWarn)
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "apply safe fixes in place")
	cmd.Flags().BoolVar(&unsafeFixes, "unsafe-fixes", false, "also apply unsafe fixes (implies --fix)")
	cmd.Flags().BoolVar(&diff, "diff", false, "list files whose fixes would change them, without writing")
	cmd.Flags().IntVar(&maxWarn, "max-warnings", -1, "fail if warnings exceed N (-1 = never)")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "disable the lint result cache")
	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "cache directory (default: per-user cache dir)")
	return cmd
}

// runFix applies fixes (or previews them with --diff) and prints the changed files.
func runFix(cmd *cobra.Command, eng *engine.Engine, opts *Options, args []string, unsafe, diff bool) error {
	changed, err := eng.Fix(context.Background(), args, unsafe, diff)
	if err != nil {
		return err
	}
	if opts.Quiet {
		return nil
	}
	for _, p := range changed {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), p); err != nil {
			return err
		}
	}
	verb := "fixed"
	if diff {
		verb = "would change"
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "%d file(s) %s\n", len(changed), verb)
	return err
}

// runLint runs a read-only lint (memoizing per-file findings unless --no-cache),
// reports them, and exits non-zero on failure.
func runLint(cmd *cobra.Command, opts *Options, eng *engine.Engine, cfg *config.Config, args []string, noCache bool, cacheDir string, maxWarn int) error {
	var c *cache.Cache
	if !noCache {
		if dir := resolveCacheDir(cacheDir); dir != "" {
			c = cache.Open(dir)
			eng.UseCache(c, opts.Version, configHash(cfg))
		}
	}
	res, err := eng.Run(context.Background(), args)
	if err != nil {
		return err
	}
	if c != nil {
		if err := c.Save(); err != nil {
			return err
		}
	}
	if err := pickReporter(opts, cmd.Flags().Changed("format")).Report(cmd.OutOrStdout(), res.Findings); err != nil {
		return err
	}
	if shouldFail(res, maxWarn) {
		os.Exit(1)
	}
	return nil
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
