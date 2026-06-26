package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func newExplainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "explain <rule>",
		Short: "Show a rule's rationale and examples",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			reg := rule.NewRegistry()
			builtin.Register(reg)
			return registryRuleNames(reg), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := rule.NewRegistry()
			builtin.Register(reg)
			r, ok := reg.Get(args[0])
			if !ok {
				return fmt.Errorf("unknown rule %q", args[0])
			}
			m := r.Meta()
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s [%s]\n\n%s\n\n%s\n", m.Name, m.Severity, m.Description, m.Detail)
			return err
		},
	}
}
