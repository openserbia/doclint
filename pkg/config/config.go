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

	Dir string `yaml:"-"` // directory of the config file (relative-path base)
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
