// Package cli wires the doclint Cobra command tree over the engine.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/openserbia/doclint/pkg/config"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

// Options holds resolved global flags.
type Options struct {
	ConfigPath string
	Format     string
	NoColor    bool
	Quiet      bool
}

// NewRootCmd builds the command tree. version/commit/date come from main.
func NewRootCmd(version, commit, date string) *cobra.Command {
	opts := &Options{}
	root := &cobra.Command{
		Use:           "doclint",
		Short:         "Lint, autofix and format Hugo markdown content and data files",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version + " (commit " + commit + ", built " + date + ")",
	}
	root.PersistentFlags().StringVar(&opts.ConfigPath, "config", "", "path to .doclint.yaml (default: discovered)")
	root.PersistentFlags().StringVar(&opts.Format, "format", "human", "output format: human|compact|json (human auto-falls back to compact when piped)")
	root.PersistentFlags().BoolVar(&opts.NoColor, "no-color", false, "disable colored output")
	root.PersistentFlags().BoolVar(&opts.Quiet, "quiet", false, "suppress non-finding output")

	root.AddCommand(newLintCmd(opts))
	root.AddCommand(newFmtCmd(opts))
	root.AddCommand(newExplainCmd())
	root.AddCommand(newListCmd(opts))
	return root
}

// loadConfig discovers/loads config and builds the registry of built-ins.
func loadConfig(opts *Options) (*config.Config, *rule.Registry, error) {
	reg := rule.NewRegistry()
	builtin.Register(reg)

	path := opts.ConfigPath
	if path == "" {
		found, err := config.Discover(".")
		if err != nil {
			return nil, nil, err
		}
		path = found
	}
	if path == "" {
		return config.Default(), reg, nil
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, nil, err
	}
	return cfg, reg, nil
}
