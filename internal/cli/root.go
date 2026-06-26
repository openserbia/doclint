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
	Version    string // doclint version, for the cache key
}

// NewRootCmd builds the command tree. version/commit/date come from main.
func NewRootCmd(version, commit, date string) *cobra.Command {
	opts := &Options{Version: version}
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

	_ = root.RegisterFlagCompletionFunc("format", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"human", "compact", "json"}, cobra.ShellCompDirectiveNoFileComp
	})

	// Replace Cobra's bare-script `completion` command with our friendlier one
	// (interactive install or manual steps); the hidden __complete machinery that
	// actually drives tab-completion is unaffected.
	root.CompletionOptions.DisableDefaultCmd = true

	root.AddCommand(newLintCmd(opts))
	root.AddCommand(newFmtCmd(opts))
	root.AddCommand(newExplainCmd())
	root.AddCommand(newListCmd(opts))
	root.AddCommand(newInitCmd())
	root.AddCommand(newCompletionCmd(root))
	root.AddCommand(newDocsCmd())
	root.AddCommand(newCacheCmd())
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
	if err := config.Validate(cfg, registryRuleNames(reg)); err != nil {
		return nil, nil, err
	}
	return cfg, reg, nil
}

// registryRuleNames returns the built-in rule names in registration order.
func registryRuleNames(reg *rule.Registry) []string {
	all := reg.All()
	names := make([]string, len(all))
	for i, r := range all {
		names[i] = r.Meta().Name
	}
	return names
}
