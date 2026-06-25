// Command doclint lints, autofixes and formats Hugo markdown content and data
// files against built-in and user-defined rules.
package main

import "fmt"

// Build-time version metadata (set via -ldflags by GoReleaser).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Replaced by Cobra Execute() in Task 13.
	fmt.Printf("doclint %s (commit %s, built %s)\n", version, commit, date)
}
