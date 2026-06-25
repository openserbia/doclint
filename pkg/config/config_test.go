package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndDiscover(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".doclint.yaml")
	body := `
default: standard
disable: [noisy]
settings:
  details-blank-line:
    severity: warning
ignore:
  - "node_modules/**"
custom:
  - id: desc-req
    type: required
    field: description
    skip_drafts: true
    severity: error
`
	if err := os.WriteFile(cfgPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "content")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	found, err := Discover(sub)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if found != cfgPath {
		t.Errorf("Discover = %q, want %q", found, cfgPath)
	}
	cfg, err := Load(found)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Default != "standard" || len(cfg.Custom) != 1 || cfg.Custom[0].ID != "desc-req" {
		t.Errorf("unexpected config: %+v", cfg)
	}
	if cfg.Settings["details-blank-line"].Severity != "warning" {
		t.Errorf("setting not parsed: %+v", cfg.Settings)
	}
}
