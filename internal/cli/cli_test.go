package cli

import (
	"bytes"
	"testing"
)

func TestListCmd_ShowsBuiltinRule(t *testing.T) {
	root := NewRootCmd("test", "t", "t")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("details-blank-line")) {
		t.Errorf("list output missing built-in rule:\n%s", out.String())
	}
}
