package cli

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestInitCommand(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := runRoot("init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	b, err := os.ReadFile(".doclint.yaml")
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	if !strings.Contains(string(b), "default: standard") {
		t.Errorf("generated config missing the default preset:\n%s", b)
	}

	if err := runRoot("init"); err == nil {
		t.Error("init must refuse to overwrite an existing config without --force")
	}
	if err := runRoot("init", "--force"); err != nil {
		t.Errorf("init --force should overwrite: %v", err)
	}
}

func runRoot(args ...string) error {
	root := NewRootCmd("test", "commit", "date")
	root.SetArgs(args)
	root.SetOut(io.Discard)
	return root.Execute()
}
