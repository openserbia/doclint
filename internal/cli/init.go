package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openserbia/doclint/pkg/config"
)

// configMode is the permission for a generated, user-editable project config.
const configMode = 0o644

// defaultConfigTemplate is the starter .doclint.yaml written by `doclint init`.
// It is intentionally heavily commented so a new project can discover the
// available knobs without reading the docs.
const defaultConfigTemplate = `# .doclint.yaml — doclint configuration.
# Rules reference: run ` + "`doclint list`" + ` (and ` + "`doclint explain <rule>`" + `).

# Which built-in rules are active before the enable/disable lists below:
#   all      — every built-in rule
#   standard — the curated default set (recommended)
#   none     — start empty and opt in via 'enable'
default: standard

# Toggle individual rules by name:
# enable: []
# disable:
#   - blanks-around-lists

# Default lint/fmt targets (relative to this file) used when you run doclint
# with no path arguments:
# paths:
#   - content
#   - data

# Override a rule's severity (error | warning | info):
settings:
  details-blank-line:
    severity: error

# Globs (relative to this file) to skip entirely:
# ignore:
#   - "**/node_modules/**"

# Project-specific declarative rules.
# Types: required | length | not_equal | match | deny.
custom:
  # Every non-draft page must have a non-empty 'description' (SEO / social cards):
  - id: frontmatter-description-required
    type: required
    glob: "content/**/*.md"
    field: description
    skip_drafts: true
    severity: error
`

func newInitCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write a starter .doclint.yaml in the current directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := config.ConfigName
			switch _, err := os.Stat(path); {
			case err == nil && !force:
				return fmt.Errorf("%s already exists (use --force to overwrite)", path)
			case err != nil && !errors.Is(err, os.ErrNotExist):
				return err
			}
			if err := os.WriteFile(path, []byte(defaultConfigTemplate), configMode); err != nil { //nolint:gosec // user-editable project config
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", path)
			return err
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing .doclint.yaml")
	return cmd
}
