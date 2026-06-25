package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newListCmd(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List built-in and custom rules",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, reg, err := loadConfig(opts)
			if err != nil {
				return err
			}
			for _, r := range reg.All() {
				m := r.Meta()
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%-24s %-8s %s\n", m.Name, m.Severity, m.Description); err != nil {
					return err
				}
			}
			for i := range cfg.Custom {
				c := &cfg.Custom[i]
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%-24s %-8s custom (%s)\n", c.ID, c.Severity, c.Type); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
