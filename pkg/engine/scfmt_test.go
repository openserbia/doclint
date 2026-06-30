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

// TestSCFmt_SingleLevel: one outer shortcode containing one inner shortcode.
//
//	{{< tabs >}}
//	{{< tab "X" >}}
//	content
//	{{< /tab >}}
//	{{< /tabs >}}
func TestSCFmt_SingleLevel(t *testing.T) {
	in := "{{< tabs >}}\n{{< tab \"X\" >}}\ncontent\n{{< /tab >}}\n{{< /tabs >}}\n"
	want := "{{< tabs >}}\n  {{< tab \"X\" >}}\ncontent\n  {{< /tab >}}\n{{< /tabs >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_TwoLevel: uplatnica-form > uf-field (the common pattern in real content).
func TestSCFmt_TwoLevel(t *testing.T) {
	in := "" +
		"{{< uplatnica-form amount=\"400,00\" >}}\n" +
		"{{< uf-field slot=\"payer\" editable=\"true\" >}}\n" +
		"Имя и фамилия\n" +
		"{{< /uf-field >}}\n" +
		"{{< /uplatnica-form >}}\n"
	want := "" +
		"{{< uplatnica-form amount=\"400,00\" >}}\n" +
		"  {{< uf-field slot=\"payer\" editable=\"true\" >}}\n" +
		"Имя и фамилия\n" +
		"  {{< /uf-field >}}\n" +
		"{{< /uplatnica-form >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_ThreeLevel: uplatnica > uplatnica-form > uf-field (three levels,
// as found in state-insurance and electron-signature).
func TestSCFmt_ThreeLevel(t *testing.T) {
	in := "" +
		"{{< uplatnica >}}\n" +
		"{{< uplatnica-form amount=\"400,00\" >}}\n" +
		"{{< uf-field slot=\"payer\" editable=\"true\" >}}\n" +
		"Имя и фамилия\n" +
		"{{< /uf-field >}}\n" +
		"{{< /uplatnica-form >}}\n" +
		"{{< /uplatnica >}}\n"
	want := "" +
		"{{< uplatnica >}}\n" +
		"  {{< uplatnica-form amount=\"400,00\" >}}\n" +
		"    {{< uf-field slot=\"payer\" editable=\"true\" >}}\n" +
		"Имя и фамилия\n" +
		"    {{< /uf-field >}}\n" +
		"  {{< /uplatnica-form >}}\n" +
		"{{< /uplatnica >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_SiblingChildren: multiple children at the same depth (several
// uf-fields inside a single uplatnica-form).
func TestSCFmt_SiblingChildren(t *testing.T) {
	in := "" +
		"{{< uplatnica-form >}}\n" +
		"{{< uf-field slot=\"payer\" editable=\"true\" >}}\n" +
		"Имя\n" +
		"{{< /uf-field >}}\n" +
		"{{< uf-field slot=\"amount\" editable=\"true\" >}}\n" +
		"Сумма\n" +
		"{{< /uf-field >}}\n" +
		"{{< /uplatnica-form >}}\n"
	want := "" +
		"{{< uplatnica-form >}}\n" +
		"  {{< uf-field slot=\"payer\" editable=\"true\" >}}\n" +
		"Имя\n" +
		"  {{< /uf-field >}}\n" +
		"  {{< uf-field slot=\"amount\" editable=\"true\" >}}\n" +
		"Сумма\n" +
		"  {{< /uf-field >}}\n" +
		"{{< /uplatnica-form >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	scIdempotent(t, in)
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

// TestSCFmt_SelfClosingDepthNeutral: a self-closing shortcode ({{< tag />}})
// does not change the nesting depth.
func TestSCFmt_SelfClosingDepthNeutral(t *testing.T) {
	in := "" +
		"{{< uplatnica-form >}}\n" +
		"{{< link-card title=\"X\" href=\"/\" />}}\n" +
		"{{< /uplatnica-form >}}\n"
	want := "" +
		"{{< uplatnica-form >}}\n" +
		"  {{< link-card title=\"X\" href=\"/\" />}}\n" +
		"{{< /uplatnica-form >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_CompoundLine: opener immediately followed by closer on the same
// line (e.g. {{< uf-field >}}{{< /uf-field >}}) has net depth change of zero;
// it must be re-indented to the current depth but must not alter depth.
func TestSCFmt_CompoundLine(t *testing.T) {
	in := "" +
		"{{< uplatnica-form >}}\n" +
		"{{< uf-field slot=\"sifra\" editable=\"true\" control=\"select\" csv=\"sifra.csv\" >}}{{< /uf-field >}}\n" +
		"{{< /uplatnica-form >}}\n"
	want := "" +
		"{{< uplatnica-form >}}\n" +
		"  {{< uf-field slot=\"sifra\" editable=\"true\" control=\"select\" csv=\"sifra.csv\" >}}{{< /uf-field >}}\n" +
		"{{< /uplatnica-form >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_AlreadyCorrect: already-indented content must not change (idempotency
// with correct input).
func TestSCFmt_AlreadyCorrect(t *testing.T) {
	in := "" +
		"{{< tabs >}}\n" +
		"  {{< tab \"Online\" >}}\n" +
		"Content here.\n" +
		"  {{< /tab >}}\n" +
		"{{< /tabs >}}\n"
	if got := scfmt(in); got != in {
		t.Errorf("already-correct changed: got %q want %q", got, in)
	}
}

// TestSCFmt_StrayLeadingWhitespaceFixed: opener and closer that have stray
// leading whitespace (e.g. 1 or 3 spaces) are both normalised to the canonical
// depth × 2 indentation.
func TestSCFmt_StrayLeadingWhitespaceFixed(t *testing.T) {
	// Mirrors the real electron-signature problem: "{{< details >}}" at 1 space,
	// "{{< /details >}}" at 3 spaces.
	in := "" +
		"{{< step >}}\n" +
		" {{< details \"Note\" >}}\n" +
		"content\n" +
		"   {{< /details >}}\n" +
		"{{< /step >}}\n"
	want := "" +
		"{{< step >}}\n" +
		"  {{< details \"Note\" >}}\n" +
		"content\n" +
		"  {{< /details >}}\n" +
		"{{< /step >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_FencedCodeUntouched: shortcode-looking lines inside a fenced code
// block must never be modified.
func TestSCFmt_FencedCodeUntouched(t *testing.T) {
	in := "```\n{{< tabs >}}\n{{< /tabs >}}\n```\n"
	if got := scfmt(in); got != in {
		t.Errorf("fenced shortcodes changed: got %q want %q", got, in)
	}
}

// TestSCFmt_MultilineTagParamsUntouched: when a tag's parameters span multiple
// lines the opener and continuation lines are emitted verbatim, but the depth
// counter is still updated so children and the closer are placed correctly.
func TestSCFmt_MultilineTagParamsUntouched(t *testing.T) {
	in := "" +
		"{{< uplatnica\n" +
		"service=\"здравствена картица\"\n" +
		">}}\n" +
		"{{< uplatnica-caption >}}\n" +
		"Сумма за 2026 год\n" +
		"{{< /uplatnica-caption >}}\n" +
		"{{< /uplatnica >}}\n"
	// Opener + params emitted verbatim; uplatnica-caption and closer re-indented
	// at depth 1 (inside uplatnica).
	want := "" +
		"{{< uplatnica\n" +
		"service=\"здравствена картица\"\n" +
		">}}\n" +
		"  {{< uplatnica-caption >}}\n" +
		"Сумма за 2026 год\n" +
		"  {{< /uplatnica-caption >}}\n" +
		"{{< /uplatnica >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_MultilineTagSelfClosingDepthNeutral: a self-closing multi-line tag
// ({{< figure\n...\n/>}}) must not alter the depth counter.
func TestSCFmt_MultilineTagSelfClosingDepthNeutral(t *testing.T) {
	in := "" +
		"{{< step >}}\n" +
		"{{< figure\n" +
		"src=\"media/img.png\"\n" +
		"alt=\"Alt text\"\n" +
		"/>}}\n" +
		"{{< /step >}}\n"
	// figure is depth-neutral; {{< /step >}} stays at depth 0.
	want := "" +
		"{{< step >}}\n" +
		"{{< figure\n" +
		"src=\"media/img.png\"\n" +
		"alt=\"Alt text\"\n" +
		"/>}}\n" +
		"{{< /step >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

// TestSCFmt_PercentDelimiter: {{% %}} shortcodes are handled identically to
// {{< >}} shortcodes.
func TestSCFmt_PercentDelimiter(t *testing.T) {
	in := "{{% tabs %}}\n{{% tab \"X\" %}}\ncontent\n{{% /tab %}}\n{{% /tabs %}}\n"
	want := "{{% tabs %}}\n  {{% tab \"X\" %}}\ncontent\n  {{% /tab %}}\n{{% /tabs %}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_BlankLinesPreserved: blank lines between tags are content lines and
// must pass through unchanged.
func TestSCFmt_BlankLinesPreserved(t *testing.T) {
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
	want := "" +
		"{{< steps >}}\n" +
		"\n" +
		"  {{< step >}}\n" +
		"\n" +
		"Content.\n" +
		"\n" +
		"  {{< /step >}}\n" +
		"\n" +
		"{{< /steps >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_MixedProseAndShortcodes: shortcode tags interspersed with prose and
// other markdown are re-indented; the prose lines are left alone.
func TestSCFmt_MixedProseAndShortcodes(t *testing.T) {
	in := "" +
		"## Steps\n" +
		"\n" +
		"{{< steps >}}\n" +
		"{{< step >}}\n" +
		"**Prepare documents**\n" +
		"\n" +
		"- Passport\n" +
		"- Contract\n" +
		"{{< /step >}}\n" +
		"{{< /steps >}}\n"
	want := "" +
		"## Steps\n" +
		"\n" +
		"{{< steps >}}\n" +
		"  {{< step >}}\n" +
		"**Prepare documents**\n" +
		"\n" +
		"- Passport\n" +
		"- Contract\n" +
		"  {{< /step >}}\n" +
		"{{< /steps >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_RealElectronSignatureBlock reproduces the exact nesting from
// electron-signature/index.md that motivated this formatter:
//
//	{{< step >}}
//	 {{< details "Note" >}}        ← stray 1-space indent
//	{{< uplatnica ... >}}          ← 0 indent despite being inside step+details
//	{{< uplatnica-form ... >}}     ← 0 indent despite being inside step+details+uplatnica
//	  {{< uf-field ... >}}         ← 2 spaces (only level that was consistent)
//	  {{< /uf-field >}}
//	{{< /uplatnica-form >}}
//	{{< /uplatnica >}}
//	   {{< /details >}}            ← stray 3-space indent (different from opener)
//	{{< /step >}}
func TestSCFmt_RealElectronSignatureBlock(t *testing.T) {
	in := "" +
		"{{< step >}}\n" +
		" {{< details \"Пример счета на оплату\" >}}\n" +
		"{{< uplatnica service=\"электронная подпись\" taksa=\"KV.ELEKT\" >}}\n" +
		"{{< uplatnica-caption >}}\n" +
		"Точную сумму берите из счёта\n" +
		"{{< /uplatnica-caption >}}\n" +
		"{{< uplatnica-form amount=\"6.360,00\" >}}\n" +
		"  {{< uf-field slot=\"payer\" editable=\"true\" >}}\n" +
		"  Имя и фамилия\n" +
		"  {{< /uf-field >}}\n" +
		"{{< /uplatnica-form >}}\n" +
		"{{< /uplatnica >}}\n" +
		"   {{< /details >}}\n" +
		"{{< /step >}}\n"
	want := "" +
		"{{< step >}}\n" +
		"  {{< details \"Пример счета на оплату\" >}}\n" +
		"    {{< uplatnica service=\"электронная подпись\" taksa=\"KV.ELEKT\" >}}\n" +
		"      {{< uplatnica-caption >}}\n" +
		"Точную сумму берите из счёта\n" +
		"      {{< /uplatnica-caption >}}\n" +
		"      {{< uplatnica-form amount=\"6.360,00\" >}}\n" +
		"        {{< uf-field slot=\"payer\" editable=\"true\" >}}\n" +
		"  Имя и фамилия\n" +
		"        {{< /uf-field >}}\n" +
		"      {{< /uplatnica-form >}}\n" +
		"    {{< /uplatnica >}}\n" +
		"  {{< /details >}}\n" +
		"{{< /step >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_DepthNeverGoesNegative: a stray closer with no matching opener
// must not panic or produce negative depth.
func TestSCFmt_DepthNeverGoesNegative(t *testing.T) {
	in := "{{< /orphan >}}\n{{< /another >}}\n"
	// Both closers land at depth 0 (max(0, -1) = 0).
	want := "{{< /orphan >}}\n{{< /another >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}
