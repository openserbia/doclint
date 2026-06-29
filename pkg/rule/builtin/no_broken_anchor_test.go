package builtin_test

import (
	"os"
	"path/filepath"
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

// anchorFindingsAt parses the file at path (already on disk under a content tree)
// and runs the rule, so cross-page resolution can find sibling pages.
func anchorFindingsAt(t *testing.T, path string) []rule.Finding {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // test path under a temp dir
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	doc, err := document.ParseMarkdown(path, raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out []rule.Finding
	(builtin.NoBrokenAnchor{}).Check(doc, func(f rule.Finding) { out = append(out, f) })
	return out
}

func TestNoBrokenAnchorCrossPage(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// Target pages: a leaf page, a leaf bundle (index.md) and a branch (_index.md).
	write("content/guides/banks/raiffeisen.md", "## Документы для открытия бизнес счёта\n\nbody\n")
	write("content/guides/banks/intesa/index.md", "## Открытие счёта\n")
	write("content/guides/banks/_index.md", "## Сравнение банков\n")

	// Valid fragment Hugo would generate for the raiffeisen heading.
	const validFrag = "документы-для-открытия-бизнес-счёта"
	// The user's broken anchor: "открытие" (not "открытия") and "счета" (no ё).
	const brokenFrag = "документы-для-открытие-бизнес-счета"

	cases := []struct {
		name   string
		body   string
		broken int
	}{
		{"valid cross-page", "[x](/guides/banks/raiffeisen/#" + validFrag + ")\n", 0},
		{"broken cross-page", "[x](/guides/banks/raiffeisen/#" + brokenFrag + ")\n", 1},
		{
			"broken inside option shortcode",
			"{{< option title=`[Бизнес счёт](/guides/banks/raiffeisen/#" + brokenFrag + " \"открытие\")` >}}\n", 1,
		},
		{"unresolvable page ignored", "[x](/guides/banks/sberbank/#whatever)\n", 0},
		{"leaf bundle resolves", "[x](/guides/banks/intesa/#открытие-счёта)\n", 0},
		{"branch index resolves", "[x](/guides/banks/#сравнение-банков)\n", 0},
		{"branch index broken", "[x](/guides/banks/#нет-такого)\n", 1},
		{"non-content asset ignored", "[img](/images/logo.png#frag)\n", 0},
		{"valid plus broken", "[a](/guides/banks/raiffeisen/#" + validFrag + ") [b](/guides/banks/raiffeisen/#" + brokenFrag + ")\n", 1},

		// Hugo {{< relref >}} shortcodes — Hugo validates the path but not the fragment.
		{"relref inside valid", "[x]({{< relref `/guides/banks/raiffeisen#" + validFrag + "` >}})\n", 0},
		{"relref inside broken", "[x]({{< relref `/guides/banks/raiffeisen#" + brokenFrag + "` >}})\n", 1},
		{"relref outside valid", "[x]({{< relref `/guides/banks/raiffeisen` >}}#" + validFrag + ")\n", 0},
		{"relref outside broken", "[x]({{< relref `/guides/banks/raiffeisen` >}}#" + brokenFrag + ")\n", 1},
		{"relref inside broken with title", "[x]({{< relref \"/guides/banks/raiffeisen#" + brokenFrag + "\" >}} \"Райф (банк)\")\n", 1},
		{"relref percent form broken", "[x]({{% relref \"/guides/banks/raiffeisen#" + brokenFrag + "\" %}})\n", 1},
		{"relref to missing page ignored", "[x]({{< relref `/guides/banks/sberbank#x` >}})\n", 0},
		{"relref no fragment ignored", "[x]({{< relref `/guides/banks/raiffeisen` >}})\n", 0},

		// relref outside a markdown link: inside an HTML href attribute, and bare.
		{"relref in html href valid", "<a href=\"{{< relref \"/guides/banks/raiffeisen#" + validFrag + "\" >}}\">x</a>\n", 0},
		{"relref in html href broken", "<a href=\"{{< relref \"/guides/banks/raiffeisen#" + brokenFrag + "\" >}}\">x</a>\n", 1},
		{"relref bare broken", "see {{< relref `/guides/banks/raiffeisen#" + brokenFrag + "` >}}\n", 1},
	}
	for _, c := range cases {
		src := filepath.Join(root, "content", "guides", "comparison.md")
		if err := os.WriteFile(src, []byte(c.body), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := anchorFindingsAt(t, src); len(got) != c.broken {
			t.Errorf("%s: got %d broken, want %d: %+v", c.name, len(got), c.broken, got)
		}
	}
}

// TestNoBrokenAnchorLanguageSubdir mirrors a multilingual Hugo layout where the
// default language's contentDir is content/<lang> and the URL carries no language
// prefix, so /guides/x/ is served from content/ru/guides/x/index.md.
func TestNoBrokenAnchorLanguageSubdir(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Target is a leaf bundle under the ru language dir; URL has no /ru/ prefix.
	write("content/ru/guides/banks/raiffeisen/index.md", "## Документы для открытия бизнес счета\n")
	write("content/ru/guides/banks/who-opens/index.md", "placeholder\n")
	src := filepath.Join(root, "content", "ru", "guides", "banks", "who-opens", "index.md")

	if err := os.WriteFile(src, []byte("[x](/guides/banks/raiffeisen/#документы-для-открытия-бизнес-счета)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := anchorFindingsAt(t, src); len(got) != 0 {
		t.Errorf("valid cross-page under content/ru: got %d broken, want 0: %+v", len(got), got)
	}

	if err := os.WriteFile(src, []byte("[x](/guides/banks/raiffeisen/#документы-для-открытие-бизнес-счета)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := anchorFindingsAt(t, src); len(got) != 1 {
		t.Errorf("broken cross-page under content/ru: got %d broken, want 1: %+v", len(got), got)
	}
}
