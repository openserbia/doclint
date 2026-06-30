package builtin

import "github.com/openserbia/doclint/pkg/rule"

// Register adds every built-in rule to reg.
func Register(reg *rule.Registry) {
	reg.Register(DetailsBlankLine{})
	reg.Register(TableColumnCount{})
	reg.Register(NoMissingSpaceATX{})
	reg.Register(HeadingStartLeft{})
	reg.Register(BlanksAroundFences{})
	reg.Register(BlanksAroundThematicBreak{})
	reg.Register(BlanksAroundLists{})
	reg.Register(BlanksAroundHeadings{})
	reg.Register(FencedCodeLanguage{})
	reg.Register(NoAltText{})
	reg.Register(NoTrailingSpaces{})
	reg.Register(NoBrokenAnchor{})
	reg.Register(ListMarkerIndent{})
}
