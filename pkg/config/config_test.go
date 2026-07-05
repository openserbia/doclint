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

func TestShortcodeIndentConfigDefaults(t *testing.T) {
	var c ShortcodeIndentConfig // all unset
	if !c.IsEnabled() {
		t.Errorf("unset Enabled should default to true")
	}
	if c.Width() != 2 {
		t.Errorf("unset IndentWidth should default to 2, got %d", c.Width())
	}
	if c.ExcludeSet() != nil {
		t.Errorf("empty Exclude should yield nil set")
	}
}

func TestShortcodeIndentConfigExplicit(t *testing.T) {
	no := false
	w := 4
	neg := -3
	c := ShortcodeIndentConfig{Enabled: &no, IndentWidth: &w, Exclude: []string{"uplatnica", "figure"}}
	if c.IsEnabled() {
		t.Errorf("explicit false Enabled should disable")
	}
	if c.Width() != 4 {
		t.Errorf("explicit IndentWidth 4, got %d", c.Width())
	}
	set := c.ExcludeSet()
	if !set["uplatnica"] || !set["figure"] || len(set) != 2 {
		t.Errorf("ExcludeSet = %v", set)
	}
	// Negative width clamps to the default.
	if (ShortcodeIndentConfig{IndentWidth: &neg}).Width() != 2 {
		t.Errorf("negative IndentWidth should clamp to default 2")
	}
}

func TestLoadFmtShortcodeIndent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".doclint.yaml")
	body := `
fmt:
  shortcode_indent:
    enabled: false
    indent_width: 3
    exclude:
      - uplatnica
`
	if err := os.WriteFile(cfgPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	sc := cfg.Fmt.ShortcodeIndent
	if sc.IsEnabled() {
		t.Errorf("enabled:false not parsed")
	}
	if sc.Width() != 3 {
		t.Errorf("indent_width:3 not parsed, got %d", sc.Width())
	}
	if !sc.ExcludeSet()["uplatnica"] {
		t.Errorf("exclude not parsed: %v", sc.Exclude)
	}
}
