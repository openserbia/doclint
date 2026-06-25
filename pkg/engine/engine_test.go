package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/openserbia/doclint/pkg/config"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func TestEngine_RunFindsAndFixes(t *testing.T) {
	dir := t.TempDir()
	md := filepath.Join(dir, "post.md")
	if err := os.WriteFile(md, []byte("<details><summary>x</summary>\n- item\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reg := rule.NewRegistry()
	builtin.Register(reg)

	cfg := config.Default()
	eng, err := New(cfg, reg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res, err := eng.Run(context.Background(), []string{dir})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// The fixture lists "- item" directly under </summary>, so two rules fire on
	// the same gap: details-blank-line (error) and blanks-around-lists (warning),
	// the latter because the list is butted against the preceding line. Their fixes
	// both insert the same separating blank.
	byRule := map[string]rule.Finding{}
	for _, f := range res.Findings {
		byRule[f.Rule] = f
	}
	if _, ok := byRule["details-blank-line"]; !ok || len(res.Findings) != 2 {
		t.Fatalf("findings = %+v", res.Findings)
	}
	if _, ok := byRule["blanks-around-lists"]; !ok {
		t.Fatalf("expected blanks-around-lists finding, got %+v", res.Findings)
	}
	if res.ExitCode() != 1 {
		t.Errorf("exit = %d, want 1 (one error)", res.ExitCode())
	}

	fixed, err := eng.Fix(context.Background(), []string{dir}, false /*unsafe*/, true /*dryRun*/)
	if err != nil {
		t.Fatalf("Fix: %v", err)
	}
	if len(fixed) != 1 {
		t.Fatalf("expected 1 changed file, got %d", len(fixed))
	}
}
