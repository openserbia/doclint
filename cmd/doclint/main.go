// Command doclint lints, autofixes and formats Hugo markdown content and data
// files against built-in and user-defined rules.
package main

import (
	"fmt"
	"os"

	"github.com/openserbia/doclint/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// exitErr is the exit code used when Execute returns an error.
const exitErr = 2

func main() {
	if err := cli.NewRootCmd(version, commit, date).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "doclint:", err)
		os.Exit(exitErr)
	}
}
