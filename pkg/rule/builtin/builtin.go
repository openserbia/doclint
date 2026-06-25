package builtin

import "github.com/openserbia/doclint/pkg/rule"

// Register adds every built-in rule to reg.
func Register(reg *rule.Registry) {
	reg.Register(DetailsBlankLine{})
}
