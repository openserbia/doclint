package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestWriteCompletionZsh(t *testing.T) {
	var buf bytes.Buffer
	if err := writeCompletion(NewRootCmd("t", "c", "d"), &buf, "zsh"); err != nil {
		t.Fatalf("writeCompletion: %v", err)
	}
	if !strings.Contains(buf.String(), "compdef") {
		t.Errorf("zsh completion script looks wrong:\n%.120s", buf.String())
	}
}

func TestInstallCompletionDeclineShowsManual(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetIn(strings.NewReader("n\n"))
	if err := installCompletion(cmd, NewRootCmd("t", "c", "d"), "zsh", false); err != nil {
		t.Fatalf("installCompletion: %v", err)
	}
	if !strings.Contains(out.String(), "source <(doclint completion zsh)") {
		t.Errorf("declining should print manual steps, got:\n%s", out.String())
	}
}

func TestInstallCompletionYesWritesFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := installCompletion(cmd, NewRootCmd("t", "c", "d"), "zsh", true); err != nil {
		t.Fatalf("installCompletion: %v", err)
	}
	target := filepath.Join(home, ".zsh", "completions", "_doclint")
	if _, err := os.Stat(target); err != nil {
		t.Errorf("completion not written to %s: %v", target, err)
	}
	if !strings.Contains(out.String(), "installed") {
		t.Errorf("expected a success message, got:\n%s", out.String())
	}
}
