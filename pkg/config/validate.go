package config

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Declarative-rule type names, mirrored from package rule. Config is decoupled
// from package rule (see CustomRule), so the vocabulary is restated here; it
// changes rarely and any drift is caught by the rule compiler at build time.
const (
	typeRequired = "required"
	typeLength   = "length"
	typeNotEqual = "not_equal"
	typeMatch    = "match"
	typeDeny     = "deny"
)

// notEqualFields is the exact number of fields a not_equal rule compares.
const notEqualFields = 2

// Suggestion tuning for didYouMean: a candidate is offered when its edit
// distance is within len(input)/suggestDivisor (but at least minThreshold).
const (
	suggestDivisor = 3
	minThreshold   = 2
)

var (
	validPresets    = []string{"all", defaultPreset, "none"}
	validSeverities = []string{"error", "warning", "info"}
	validCustomType = []string{typeRequired, typeLength, typeNotEqual, typeMatch, typeDeny}
)

// Validate checks a loaded config and returns the first problem as a clear,
// file-prefixed error. knownRules are the registered built-in rule names; custom
// rule IDs are added to the known set so enable/disable/settings may reference
// them. It runs at preflight, before the engine builds, so a malformed config
// fails fast with an actionable message instead of silently doing nothing.
func Validate(cfg *Config, knownRules []string) error {
	known := make(map[string]bool, len(knownRules)+len(cfg.Custom))
	for _, n := range knownRules {
		known[n] = true
	}
	// Custom rules are validated first; valid ids join the known set.
	if err := validateCustom(cfg, known); err != nil {
		return err
	}
	if cfg.Default != "" && !contains(validPresets, cfg.Default) {
		return cfgErr(`"default" must be one of [%s] (got %q)`, strings.Join(validPresets, ", "), cfg.Default)
	}
	for _, ref := range []struct {
		field string
		names []string
	}{{"enable", cfg.Enable}, {"disable", cfg.Disable}} {
		for _, n := range ref.names {
			if !known[n] {
				return cfgErr("%s references unknown rule %q%s", ref.field, n, didYouMean(n, sortedKeys(known)))
			}
		}
	}
	for name, set := range cfg.Settings {
		if !known[name] {
			return cfgErr("settings references unknown rule %q%s", name, didYouMean(name, sortedKeys(known)))
		}
		if set.Severity != "" && !contains(validSeverities, set.Severity) {
			return cfgErr("rule %q: invalid severity %q (valid: %s)", name, set.Severity, strings.Join(validSeverities, ", "))
		}
	}
	return nil
}

func validateCustom(cfg *Config, known map[string]bool) error {
	for i := range cfg.Custom {
		c := &cfg.Custom[i]
		if strings.TrimSpace(c.ID) == "" {
			return cfgErr("custom rule #%d has an empty id", i+1)
		}
		if known[c.ID] {
			return cfgErr("custom rule %q: duplicate id (also a built-in or an earlier custom rule)", c.ID)
		}
		if err := validateCustomRule(c); err != nil {
			return err
		}
		known[c.ID] = true
	}
	return nil
}

func validateCustomRule(c *CustomRule) error {
	if !contains(validCustomType, c.Type) {
		return cfgErr("custom rule %q: unknown type %q (valid: %s)", c.ID, c.Type, strings.Join(validCustomType, ", "))
	}
	if err := validateCustomFields(c); err != nil {
		return err
	}
	if c.Severity != "" && !contains(validSeverities, c.Severity) {
		return cfgErr("custom rule %q: invalid severity %q (valid: %s)", c.ID, c.Severity, strings.Join(validSeverities, ", "))
	}
	return nil
}

func validateCustomFields(c *CustomRule) error {
	switch c.Type {
	case typeRequired:
		if c.Field == "" {
			return cfgErr("custom rule %q: type %q needs a %q", c.ID, typeRequired, "field")
		}
	case typeLength:
		if c.Field == "" {
			return cfgErr("custom rule %q: type %q needs a %q", c.ID, typeLength, "field")
		}
		if c.Min <= 0 && c.Max <= 0 {
			return cfgErr("custom rule %q: type %q needs a min and/or max greater than 0", c.ID, typeLength)
		}
	case typeNotEqual:
		if len(c.Fields) != notEqualFields {
			return cfgErr("custom rule %q: type %q needs exactly 2 fields (got %d)", c.ID, typeNotEqual, len(c.Fields))
		}
	case typeMatch, typeDeny:
		return validatePatternRule(c)
	}
	return nil
}

func validatePatternRule(c *CustomRule) error {
	if c.Field == "" {
		return cfgErr("custom rule %q: type %q needs a %q", c.ID, c.Type, "field")
	}
	if c.Pattern == "" {
		return cfgErr("custom rule %q: type %q needs a %q", c.ID, c.Type, "pattern")
	}
	if _, err := regexp.Compile(c.Pattern); err != nil {
		return cfgErr("custom rule %q: invalid pattern: %v", c.ID, err)
	}
	return nil
}

func cfgErr(format string, a ...any) error {
	return fmt.Errorf(ConfigName+": "+format, a...)
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// didYouMean returns ` (did you mean "x"?)` when a candidate is within a small
// edit distance of got, else "". Keeps unknown-rule errors actionable for the
// common typo without suggesting wildly different names.
func didYouMean(got string, candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}
	best, bestDist := candidates[0], levenshtein(got, candidates[0])
	for _, c := range candidates[1:] {
		if d := levenshtein(got, c); d < bestDist {
			best, bestDist = c, d
		}
	}
	threshold := len([]rune(got))/suggestDivisor + 1
	if threshold < minThreshold {
		threshold = minThreshold
	}
	if bestDist <= threshold {
		return fmt.Sprintf(" (did you mean %q?)", best)
	}
	return ""
}

// levenshtein is the standard two-row edit distance over runes.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}
