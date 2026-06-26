package config

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	known := []string{"blanks-around-lists", "details-blank-line"}

	cases := []struct {
		name    string
		cfg     *Config
		wantErr string // substring to find; "" means expect success
	}{
		{"valid", &Config{Default: "standard", Settings: map[string]RuleSetting{"details-blank-line": {Severity: "error"}}}, ""},
		{"bad preset", &Config{Default: "stadnard"}, `"default" must be one of`},
		{"unknown disable suggests", &Config{Disable: []string{"blanks-around-list"}}, `did you mean "blanks-around-lists"`},
		{"unknown setting", &Config{Settings: map[string]RuleSetting{"nope": {}}}, "unknown rule"},
		{"bad setting severity", &Config{Settings: map[string]RuleSetting{"details-blank-line": {Severity: "eror"}}}, "invalid severity"},
		{"empty custom id", &Config{Custom: []CustomRule{{Type: "required", Field: "x"}}}, "empty id"},
		{"custom id shadows builtin", &Config{Custom: []CustomRule{{ID: "details-blank-line", Type: "required", Field: "x"}}}, "duplicate id"},
		{"unknown custom type", &Config{Custom: []CustomRule{{ID: "x", Type: "bogus"}}}, "unknown type"},
		{"length needs field", &Config{Custom: []CustomRule{{ID: "x", Type: "length", Max: 10}}}, `needs a "field"`},
		{"length needs bound", &Config{Custom: []CustomRule{{ID: "x", Type: "length", Field: "a"}}}, "min"},
		{"not_equal needs two", &Config{Custom: []CustomRule{{ID: "x", Type: "not_equal", Fields: []string{"a"}}}}, "exactly 2"},
		{"match needs pattern", &Config{Custom: []CustomRule{{ID: "x", Type: "match", Field: "a"}}}, `needs a "pattern"`},
		{"bad regex", &Config{Custom: []CustomRule{{ID: "x", Type: "match", Field: "a", Pattern: "("}}}, "invalid pattern"},
		{"custom id usable as ref", &Config{Disable: []string{"my-rule"}, Custom: []CustomRule{{ID: "my-rule", Type: "required", Field: "a"}}}, ""},
	}
	for _, c := range cases {
		err := Validate(c.cfg, known)
		if c.wantErr == "" {
			if err != nil {
				t.Errorf("%s: unexpected error: %v", c.name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), c.wantErr) {
			t.Errorf("%s: want error containing %q, got %v", c.name, c.wantErr, err)
		}
	}
}
