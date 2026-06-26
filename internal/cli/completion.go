package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const completionDirMode = 0o755

const (
	shellBash       = "bash"
	shellZsh        = "zsh"
	shellFish       = "fish"
	shellPowerShell = "powershell"
)

func newCompletionCmd(root *cobra.Command) *cobra.Command {
	var printOnly, assumeYes bool
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate or install the shell completion script",
		Long: `Generate the shell completion script, or install it interactively.

Run in a terminal this offers to install completion to the conventional location
for your shell, or prints manual steps if you decline. When the output is piped
or redirected it prints the raw script instead, so these still work:

  source <(doclint completion zsh)
  doclint completion zsh > ~/.zsh/completions/_doclint`,
		ValidArgs: []string{shellBash, shellZsh, shellFish, shellPowerShell},
		Args:      cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := detectShell(args)
			if shell == "" {
				return errors.New("could not detect your shell; pass one of: bash, zsh, fish, powershell")
			}
			// Piped/redirected or --print: emit the raw script so `source <(…)`
			// and `> file` keep working untouched.
			if printOnly || !term.IsTerminal(int(os.Stdout.Fd())) {
				return writeCompletion(root, cmd.OutOrStdout(), shell)
			}
			return installCompletion(cmd, root, shell, assumeYes)
		},
	}
	cmd.Flags().BoolVar(&printOnly, "print", false, "print the script to stdout instead of installing")
	cmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "install without prompting")
	return cmd
}

func detectShell(args []string) string {
	if len(args) == 1 {
		return args[0]
	}
	switch sh := filepath.Base(os.Getenv("SHELL")); sh {
	case shellBash, shellZsh, shellFish:
		return sh
	default:
		return "" // powershell and unknowns must be named explicitly
	}
}

func writeCompletion(root *cobra.Command, w io.Writer, shell string) error {
	switch shell {
	case shellBash:
		return root.GenBashCompletionV2(w, true)
	case shellZsh:
		return root.GenZshCompletion(w)
	case shellFish:
		return root.GenFishCompletion(w, true)
	case shellPowerShell:
		return root.GenPowerShellCompletionWithDesc(w)
	default:
		return fmt.Errorf("unsupported shell %q (use bash, zsh, fish or powershell)", shell)
	}
}

// installCompletion offers to write the completion script to the conventional
// location for shell, or prints manual instructions when the user declines or no
// auto-install path is known (powershell).
func installCompletion(cmd, root *cobra.Command, shell string, assumeYes bool) error {
	out := cmd.OutOrStdout()
	target, activate, ok := completionTarget(shell)
	if !ok {
		return printManual(out, shell)
	}
	if !assumeYes {
		if _, err := fmt.Fprintf(out, "Install %s completion for doclint to:\n  %s\n\nProceed? [y/N] ", shell, target); err != nil {
			return err
		}
		if !readYes(cmd.InOrStdin()) {
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
			return printManual(out, shell)
		}
	}
	if err := os.MkdirAll(filepath.Dir(target), completionDirMode); err != nil {
		return err
	}
	f, err := os.Create(target) //nolint:gosec // path is $HOME + a fixed name
	if err != nil {
		return err
	}
	if err := writeCompletion(root, f, shell); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "\n✓ installed: %s\n\n%s\n", target, activate)
	return err
}

// completionTarget returns the conventional install path for shell and the
// activation hint printed after writing. ok is false when there is no good
// auto-install location, so the caller falls back to manual instructions.
func completionTarget(shell string) (path, activate string, ok bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", false
	}
	switch shell {
	case shellZsh:
		if isDir(filepath.Join(home, ".oh-my-zsh")) {
			return filepath.Join(home, ".oh-my-zsh", "completions", "_doclint"),
				"Restart your shell (or run `exec zsh`) — Oh My Zsh loads it automatically.", true
		}
		return filepath.Join(home, ".zsh", "completions", "_doclint"),
			"Add this to ~/.zshrc once, then restart your shell:\n" +
				"  fpath=(~/.zsh/completions $fpath)\n" +
				"  autoload -U compinit && compinit", true
	case shellBash:
		return filepath.Join(home, ".local", "share", "bash-completion", "completions", "doclint"),
			"Restart your shell (requires the bash-completion package).", true
	case shellFish:
		return filepath.Join(home, ".config", "fish", "completions", "doclint.fish"),
			"Restart your shell — fish loads it automatically.", true
	default:
		return "", "", false // powershell: profile editing, manual only
	}
}

func printManual(w io.Writer, shell string) error {
	steps, ok := manualSteps[shell]
	if !ok {
		return fmt.Errorf("unsupported shell %q (use bash, zsh, fish or powershell)", shell)
	}
	_, err := fmt.Fprintf(w, "To enable %s completion, run:\n\n%s\n", shell, steps)
	return err
}

var manualSteps = map[string]string{
	shellZsh: "  # one-off (current shell):\n" +
		"  source <(doclint completion zsh)\n\n" +
		"  # persistent:\n" +
		"  doclint completion zsh > ~/.zsh/completions/_doclint\n" +
		"  # then ensure ~/.zsh/completions is on $fpath and run `compinit`",
	shellBash: "  # one-off (current shell):\n" +
		"  source <(doclint completion bash)\n\n" +
		"  # persistent:\n" +
		"  doclint completion bash > ~/.local/share/bash-completion/completions/doclint",
	shellFish: "  # persistent:\n" +
		"  doclint completion fish > ~/.config/fish/completions/doclint.fish",
	shellPowerShell: "  # add to your PowerShell profile ($PROFILE):\n" +
		"  doclint completion powershell | Out-String | Invoke-Expression",
}

func readYes(r io.Reader) bool {
	s := bufio.NewScanner(r)
	if !s.Scan() {
		return false
	}
	a := strings.ToLower(strings.TrimSpace(s.Text()))
	return a == "y" || a == "yes"
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
