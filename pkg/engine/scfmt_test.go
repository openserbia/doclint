package engine

import (
	"testing"
)

// scfmt is a thin helper so table cells stay readable.
func scfmt(src string) string {
	return string(formatShortcodeIndent([]byte(src)))
}

// idempotent verifies that running the formatter twice produces the same output
// as running it once.
func scIdempotent(t *testing.T, src string) {
	t.Helper()
	once := scfmt(src)
	twice := scfmt(once)
	if once != twice {
		t.Errorf("not idempotent:\n once=%q\ntwice=%q", once, twice)
	}
}

// TestSCFmt_StandaloneVerbatim: standalone shortcodes (not inside a list item)
// are emitted verbatim — no tree re-indentation.
func TestSCFmt_StandaloneVerbatim(t *testing.T) {
	in := "" +
		"{{< tabs >}}\n" +
		"{{< tab \"X\" >}}\n" +
		"content\n" +
		"{{< /tab >}}\n" +
		"{{< /tabs >}}\n"
	if got := scfmt(in); got != in {
		t.Errorf("standalone changed: got %q want %q", got, in)
	}
}

// TestSCFmt_StandaloneNestedVerbatim: nested standalone shortcodes are not
// re-indented — whatever indentation the author wrote is preserved.
func TestSCFmt_StandaloneNestedVerbatim(t *testing.T) {
	in := "" +
		"{{< steps >}}\n" +
		"\n" +
		"{{< step >}}\n" +
		"\n" +
		"Content.\n" +
		"\n" +
		"{{< /step >}}\n" +
		"\n" +
		"{{< /steps >}}\n"
	if got := scfmt(in); got != in {
		t.Errorf("standalone nested changed: got %q want %q", got, in)
	}
}

// TestSCFmt_StandaloneAlreadyIndentedPreserved: shortcodes the author indented
// are left as-is — no stripping, no pushing.
func TestSCFmt_StandaloneAlreadyIndentedPreserved(t *testing.T) {
	in := "" +
		"{{< tabs >}}\n" +
		"  {{< tab \"Online\" >}}\n" +
		"Content here.\n" +
		"  {{< /tab >}}\n" +
		"{{< /tabs >}}\n"
	if got := scfmt(in); got != in {
		t.Errorf("already-indented changed: got %q want %q", got, in)
	}
}

// TestSCFmt_InlineInListNotTouched: a shortcode used inline inside a list item
// must never be re-indented, because TrimSpace of the line starts with "- ",
// not "{{<".
func TestSCFmt_InlineInListNotTouched(t *testing.T) {
	in := "- see {{< relref \"/guides/foo\" >}} for details\n"
	if got := scfmt(in); got != in {
		t.Errorf("inline-in-list changed: got %q want %q", got, in)
	}
}

// TestSCFmt_InlineRelrefNotTouched: relref used as a link target in prose must
// not be re-indented.
func TestSCFmt_InlineRelrefNotTouched(t *testing.T) {
	in := "[text]({{< relref \"/guides/foo\" >}} \"title\")\n"
	if got := scfmt(in); got != in {
		t.Errorf("inline relref changed: got %q want %q", got, in)
	}
}

// TestSCFmt_FencedCodeUntouched: shortcode-looking lines inside a fenced code
// block must never be modified.
func TestSCFmt_FencedCodeUntouched(t *testing.T) {
	in := "```\n{{< tabs >}}\n{{< /tabs >}}\n```\n"
	if got := scfmt(in); got != in {
		t.Errorf("fenced shortcodes changed: got %q want %q", got, in)
	}
}

// TestSCFmt_MultilineTagVerbatim: multi-line tag parameters are always
// emitted verbatim.
func TestSCFmt_MultilineTagVerbatim(t *testing.T) {
	in := "" +
		"{{< uplatnica\n" +
		"service=\"здравствена картица\"\n" +
		">}}\n" +
		"{{< uplatnica-caption >}}\n" +
		"Сумма за 2026 год\n" +
		"{{< /uplatnica-caption >}}\n" +
		"{{< /uplatnica >}}\n"
	if got := scfmt(in); got != in {
		t.Errorf("multiline tag changed: got %q want %q", got, in)
	}
}

// TestSCFmt_PercentDelimiterVerbatim: {{% %}} shortcodes pass through verbatim
// when not inside a list item.
func TestSCFmt_PercentDelimiterVerbatim(t *testing.T) {
	in := "{{% tabs %}}\n{{% tab \"X\" %}}\ncontent\n{{% /tab %}}\n{{% /tabs %}}\n"
	if got := scfmt(in); got != in {
		t.Errorf("percent delimiter changed: got %q want %q", got, in)
	}
}

// ---------- List-inline block tests (the main feature) ----------

// TestSCFmt_ListNestedShortcodePreservesIndent: when a shortcode opens inline
// in a list item ("1. {{< details >}}"), the closer is re-indented to the
// list-continuation column.
func TestSCFmt_ListNestedShortcodePreservesIndent(t *testing.T) {
	in := "" +
		"1. {{< details \"Заполненная форма\" >}}\n" +
		"   - Форму выдают на месте\n" +
		"   - Заполненный пример\n" +
		"   {{< /details >}}\n" +
		"2. {{< details \"Бели картон\" >}}\n" +
		"   - Что-то тут\n" +
		"   {{< /details >}}\n"
	// Both closers keep their 3-space list-continuation indent.
	if got := scfmt(in); got != in {
		t.Errorf("got:\n%s\nwant:\n%s", got, in)
	}
	scIdempotent(t, in)
}

// TestSCFmt_ListNestedAtColumnZero: closer at column 0 inside a list is
// re-indented to the list-continuation column (3 spaces for "1. ").
func TestSCFmt_ListNestedAtColumnZero(t *testing.T) {
	in := "" +
		"1. {{< details \"X\" >}}\n" +
		"   - Item\n" +
		"{{< /details >}}\n"
	want := "" +
		"1. {{< details \"X\" >}}\n" +
		"   - Item\n" +
		"   {{< /details >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_BulletListNestedIndent: closer inside a bullet-list item is
// re-indented to the 2-space content column (len("- ") = 2).
func TestSCFmt_BulletListNestedIndent(t *testing.T) {
	in := "" +
		"- {{< details \"X\" >}}\n" +
		"  - Nested item\n" +
		"{{< /details >}}\n"
	want := "" +
		"- {{< details \"X\" >}}\n" +
		"  - Nested item\n" +
		"  {{< /details >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_ListNestedMultipleItems: multiple list items each with an inline
// shortcode opener — each closer is re-indented to the correct column.
func TestSCFmt_ListNestedMultipleItems(t *testing.T) {
	in := "" +
		"1. {{< details \"Заполненная форма\" >}}\n" +
		"   - Форму выдают на месте\n" +
		"   - Заполненный пример\n" +
		"{{< /details >}}\n" +
		"2. {{< details \"Бели картон\" >}}\n" +
		"   - Что-то тут\n" +
		"{{< /details >}}\n"
	want := "" +
		"1. {{< details \"Заполненная форма\" >}}\n" +
		"   - Форму выдают на месте\n" +
		"   - Заполненный пример\n" +
		"   {{< /details >}}\n" +
		"2. {{< details \"Бели картон\" >}}\n" +
		"   - Что-то тут\n" +
		"   {{< /details >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_ListNestedSingleTag: a single-tag shortcode (no closer) inside a
// list-inline block shortcode is re-indented to the list-continuation column.
func TestSCFmt_ListNestedSingleTag(t *testing.T) {
	in := "" +
		"7. {{< details \"Пошлина\" >}}\n" +
		"{{< euprava-payment key=\"fee\" >}}\n" +
		"   {{< /details >}}\n"
	want := "" +
		"7. {{< details \"Пошлина\" >}}\n" +
		"   {{< euprava-payment key=\"fee\" >}}\n" +
		"   {{< /details >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_ListNestedBlockShortcodes: nested block shortcodes inside a
// list-inline opener get the list-continuation base plus depth-based indent.
func TestSCFmt_ListNestedBlockShortcodes(t *testing.T) {
	in := "" +
		"7. {{< details \"Пошлина\" >}}\n" +
		"{{< uplatnica >}}\n" +
		"{{< uplatnica-form amount=\"400\" >}}\n" +
		"{{< uf-field slot=\"payer\" >}}\n" +
		"Имя\n" +
		"{{< /uf-field >}}\n" +
		"{{< /uplatnica-form >}}\n" +
		"{{< /uplatnica >}}\n" +
		"   {{< /details >}}\n"
	want := "" +
		"7. {{< details \"Пошлина\" >}}\n" +
		"   {{< uplatnica >}}\n" +
		"     {{< uplatnica-form amount=\"400\" >}}\n" +
		"       {{< uf-field slot=\"payer\" >}}\n" +
		"Имя\n" +
		"       {{< /uf-field >}}\n" +
		"     {{< /uplatnica-form >}}\n" +
		"   {{< /uplatnica >}}\n" +
		"   {{< /details >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_DepthNeverGoesNegative: a stray closer with no matching opener
// must not panic or produce negative depth.
func TestSCFmt_DepthNeverGoesNegative(t *testing.T) {
	in := "{{< /orphan >}}\n{{< /another >}}\n"
	if got := scfmt(in); got != in {
		t.Errorf("got %q\nwant %q", got, in)
	}
}
