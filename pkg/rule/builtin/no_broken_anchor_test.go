package builtin_test

import (
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func anchorFindings(t *testing.T, raw string) []rule.Finding {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", []byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out []rule.Finding
	(builtin.NoBrokenAnchor{}).Check(doc, func(f rule.Finding) { out = append(out, f) })
	return out
}

func TestNoBrokenAnchor(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		broken int
	}{
		{"valid ascii", "## Getting Started\n\nSee [start](#getting-started).\n", 0},
		{"valid cyrillic", "## Зачем это нужно?\n\n[тут](#зачем-это-нужно)\n", 0},
		{"link with title", "## Real One\n\n[x](#real-one \"t\")\n", 0},
		{"broken", "## Real Heading\n\n[x](#missing-anchor)\n", 1},
		{"explicit id ok", "## Heading {#custom}\n\n[x](#custom)\n", 0},
		{"explicit id replaces natural", "## Heading {#custom}\n\n[x](#heading)\n", 1},
		{"duplicate suffix", "## Dup\n\n## Dup\n\n[a](#dup) [b](#dup-1)\n", 0},
		{"cross-page ignored", "## H\n\n[x](/other#frag)\n", 0},
		{"fenced ignored", "## H\n\n```\n[x](#nope)\n```\n", 0},
		{"shortcode anchor param", "{{< alert anchor=\"foo\" >}}x{{< /alert >}}\n\n[y](#foo)\n", 0},
		{"step ordinal", "{{< steps >}}\n{{< step >}}\na\n{{< /step >}}\n{{< step >}}\nb\n{{< /step >}}\n{{< /steps >}}\n\n[x](#step-2)\n", 0},
	}
	for _, c := range cases {
		if got := anchorFindings(t, c.raw); len(got) != c.broken {
			t.Errorf("%s: got %d broken, want %d: %+v", c.name, len(got), c.broken, got)
		}
	}
}
