// Package config loads .doclint.yaml: rule defaults/toggles, per-rule settings,
// ignore globs, and the declarative custom-rule block.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigName is the discovered file name.
const ConfigName = ".doclint.yaml"

// defaultPreset is the rule-preset used when none is specified.
const defaultPreset = "standard"

// RuleSetting overrides a rule's defaults.
type RuleSetting struct {
	Severity string `yaml:"severity"`
}

// CustomRule is one declarative rule (mirrors rule.DeclSpec; kept decoupled so
// config has no dependency on package rule's internals).
type CustomRule struct {
	ID         string   `yaml:"id"`
	Type       string   `yaml:"type"`
	Glob       string   `yaml:"glob"`
	Field      string   `yaml:"field"`
	Fields     []string `yaml:"fields"`
	Min        int      `yaml:"min"`
	Max        int      `yaml:"max"`
	Pattern    string   `yaml:"pattern"`
	SkipDrafts bool     `yaml:"skip_drafts"`
	Severity   string   `yaml:"severity"`
}

// Config is the parsed .doclint.yaml plus the directory it was loaded from.
type Config struct {
	Default  string                 `yaml:"default"`
	Enable   []string               `yaml:"enable"`
	Disable  []string               `yaml:"disable"`
	Settings map[string]RuleSetting `yaml:"settings"`
	Ignore   []string               `yaml:"ignore"`
	Paths    []string               `yaml:"paths"` // default lint/fmt targets when none given on the CLI
	Custom   []CustomRule           `yaml:"custom"`
	Fmt      FmtConfig              `yaml:"fmt"` // `fmt` command formatting-pass options

	Dir string `yaml:"-"` // directory of the config file (relative-path base)
}

// FmtConfig configures the `fmt` command's formatting passes.
type FmtConfig struct {
	ShortcodeIndent ShortcodeIndentConfig `yaml:"shortcode_indent"`
}

// Default values for the shortcode-indent pass when a field is left unset.
const (
	defaultShortcodeIndentEnabled = true
	defaultShortcodeIndentWidth   = 2
)

// ShortcodeIndentConfig configures the shortcode-indent formatting pass, which
// re-indents Hugo shortcode tags opened inline in a list item to align with the
// item's continuation content. Pointer fields distinguish "unset" (use the
// default) from an explicit value.
type ShortcodeIndentConfig struct {
	Enabled     *bool    `yaml:"enabled"`      // run the pass at all (default true)
	IndentWidth *int     `yaml:"indent_width"` // spaces per nesting level (default 2)
	Exclude     []string `yaml:"exclude"`      // shortcode names whose subtree is left verbatim
}

// IsEnabled reports whether the shortcode-indent pass should run.
func (c ShortcodeIndentConfig) IsEnabled() bool {
	if c.Enabled == nil {
		return defaultShortcodeIndentEnabled
	}
	return *c.Enabled
}

// Width returns the per-nesting-level indent width in spaces, clamped to a sane
// default when unset or negative.
func (c ShortcodeIndentConfig) Width() int {
	if c.IndentWidth == nil || *c.IndentWidth < 0 {
		return defaultShortcodeIndentWidth
	}
	return *c.IndentWidth
}

// ExcludeSet returns the excluded shortcode names as a set, or nil if none.
func (c ShortcodeIndentConfig) ExcludeSet() map[string]bool {
	if len(c.Exclude) == 0 {
		return nil
	}
	m := make(map[string]bool, len(c.Exclude))
	for _, n := range c.Exclude {
		m[n] = true
	}
	return m
}

// Default returns the built-in config used when no file is found.
func Default() *Config {
	return &Config{Default: defaultPreset, Settings: map[string]RuleSetting{}}
}

// Load reads and parses a config file.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path) //nolint:gosec // path is the discovered config file
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg := Default()
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if cfg.Settings == nil {
		cfg.Settings = map[string]RuleSetting{}
	}
	if cfg.Default == "" {
		cfg.Default = defaultPreset
	}
	cfg.Dir = filepath.Dir(path)
	return cfg, nil
}

// Discover walks up from start looking for ConfigName; returns "" if none.
func Discover(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, ConfigName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil // reached filesystem root
		}
		dir = parent
	}
}
