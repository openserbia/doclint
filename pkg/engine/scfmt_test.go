package engine

import (
	"testing"
)

// scfmt is a thin helper so table cells stay readable. It uses the default
// indent width (2) and no shortcode exclusions.
func scfmt(src string) string {
	return string(formatShortcodeIndent([]byte(src), 2, nil))
}

// scfmtCfg runs the pass with a custom indent width and optional excluded
// shortcode names.
func scfmtCfg(src string, width int, exclude ...string) string {
	var set map[string]bool
	if len(exclude) > 0 {
		set = map[string]bool{}
		for _, n := range exclude {
			set[n] = true
		}
	}
	return string(formatShortcodeIndent([]byte(src), width, set))
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

// TestSCFmt_SelfContainedLineNoCascade: a single-line block shortcode that
// opens and closes on the same line ("{{< alert >}}text{{< /alert >}}") has a
// net-zero effect on nesting depth. It must not leave stale depth/list state
// that wrongly indents later standalone shortcodes — a regression guard against
// the cascading-indentation bug.
func TestSCFmt_SelfContainedLineNoCascade(t *testing.T) {
	// "alert" is registered as a block name via the multi-line form up top,
	// then reused as a one-line compound. The trailing standalone shortcode and
	// the details block after the list must stay at column 0.
	in := "" +
		"{{< alert >}}\n" +
		"multi-line\n" +
		"{{< /alert >}}\n" +
		"\n" +
		"<details><summary>S</summary>\n" +
		"\n" +
		"  {{< alert icon=\"i\" >}}One-line compound.{{< /alert >}}\n" +
		"\n" +
		"- {{% tr-common %}}\n" +
		"- {{< details \"Inner\" >}}\n" +
		"  - foo\n" +
		"  {{< /details >}}\n" +
		"- {{% tr-apartment %}}\n" +
		"{{% tr-permanent-trivial %}}\n" +
		"\n" +
		"</details>\n" +
		"\n" +
		"{{< details \"Standalone\" >}}\n" +
		"{{< inner key=\"x\" >}}\n" +
		"{{< /details >}}\n"
	if got := scfmt(in); got != in {
		t.Errorf("self-contained line caused cascade:\ngot:\n%s\nwant:\n%s", got, in)
	}
	scIdempotent(t, in)
}

// TestSCFmt_RealPermanentResidenceOffline is a faithful regression test using
// the exact content that triggered the cascading-indentation bug in
// content/ru/guides/permanent-residence/offline/index.md.
//
// The trigger chain, all reproduced verbatim below:
//  1. A multi-line {{< alert >}}…{{< /alert >}} registers "alert" as a block name.
//  2. The "Работа" section reuses alert as a ONE-LINE compound
//     ("{{< alert … >}}текст{{< /alert >}}"). The old pass misread it as a bare
//     opener, corrupting the global depth counter.
//  3. That corruption cascaded: the trailing standalone "{{% tr-permanent-trivial-docs %}}"
//     and the later top-level "{{< details … >}}" block under the "Пошлина"
//     heading (which contains a nested {{< tax-sum >}} in its title) both got
//     spurious indentation.
//
// The whole block is already correctly indented, so the pass must leave it
// byte-for-byte unchanged.
func TestSCFmt_RealPermanentResidenceOffline(t *testing.T) {
	in := `{{< alert icon="🚧" context="warning">}}
- Если юридический адрес ИП в коворкинге, МУПу все равно для ПМЖ.
- Если юридический адрес ИП в другом городе, нужно перенести юр. адрес.
{{< /alert >}}

<details><summary>Документы по основанию - Работа</summary>

  {{< alert icon="💡" context="info">}}Этот тип заявления подходит если вы официально трудоустроены в штат в сербскую фирму на территории Сербии.{{< /alert >}}

- {{% tr-common-permanent-docs %}}
- {{% tr-funds-docs %}}
- {{< details "Документы по работе:" >}}
  - Рабочий договор где видно что вы официально трудоустроены
  - (Для Нового Сада) [Извод из АПР]({{< relref "/guides/business/changing-data#извод" >}} "выписку из реестра") в которой вы работаете
    - Не старше 30 дней
  {{< /details >}}
- {{% tr-apartment-docs "(Очень редко спрашивают) Документы на квартиру в которой проживаете" %}}
- {{% tr-education-docs "(Иногда просят) " %}}
{{% tr-permanent-trivial-docs %}}

</details>

### Пошлина за подачу на ПМЖ

{{< details "Оплата онлайн на eUprava ({{< tax-sum permanent_residence_zahtev_fee permanent_residence_approval_fee >}})" >}}
{{< euprava-payment key="permanent_residence_approval_fee" image="media/euprava-odobrenje-stalnog-nastanjenja.webp" >}}
{{< /details >}}
`
	if got := scfmt(in); got != in {
		t.Errorf("real offline/index.md content was re-indented:\ngot:\n%s\nwant:\n%s", got, in)
	}
	scIdempotent(t, in)
}

// TestSCFmt_AlignToSiblingContentColZero: when a list item's continuation prose
// is at column 0 (a consistent author layout under a wider marker like "10. "),
// the item's shortcode tags align to that prose column (0) — NOT the theoretical
// marker-width column (4). Real content from vozila/replacing-driving-license.
func TestSCFmt_AlignToSiblingContentColZero(t *testing.T) {
	in := "" +
		"10. {{< details \"Пошлина ({{< tax driving_license_replacement_fee >}})\" >}}\n" +
		"\n" +
		"Оплачивать пошлину заранее, до похода в МУП. Квитанцию заберут\n" +
		"{{< euprava-payment key=\"driving_license_replacement_fee\" >}}\n" +
		"{{< /details >}}\n"
	if got := scfmt(in); got != in {
		t.Errorf("col-0 sibling prose not respected:\ngot:\n%s\nwant:\n%s", got, in)
	}
	scIdempotent(t, in)
}

// TestSCFmt_AlignToSiblingContentIndented: the same block with a consistent
// 3-space continuation indent stays at 3 — the tags align to the prose, not to
// the marker-width column (4).
func TestSCFmt_AlignToSiblingContentIndented(t *testing.T) {
	in := "" +
		"10. {{< details \"Пошлина\" >}}\n" +
		"\n" +
		"   Оплачивать пошлину заранее\n" +
		"   {{< euprava-payment key=\"fee\" >}}\n" +
		"   {{< /details >}}\n"
	if got := scfmt(in); got != in {
		t.Errorf("3-space sibling prose not respected:\ngot:\n%s\nwant:\n%s", got, in)
	}
	scIdempotent(t, in)
}

// TestSCFmt_NoProseFallsBackToMarker: a tag-only list-inline block (no direct
// prose to align to) falls back to the marker-width continuation column.
func TestSCFmt_NoProseFallsBackToMarker(t *testing.T) {
	in := "" +
		"7. {{< details \"Пошлина\" >}}\n" +
		"{{< euprava-payment key=\"fee\" >}}\n" +
		"{{< /details >}}\n"
	want := "" +
		"7. {{< details \"Пошлина\" >}}\n" +
		"   {{< euprava-payment key=\"fee\" >}}\n" +
		"   {{< /details >}}\n"
	if got := scfmt(in); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	scIdempotent(t, in)
}

// TestSCFmt_IndentWidthConfigurable: the per-nesting-level indent width is
// configurable; width 4 puts a nested block at base+4 instead of base+2.
func TestSCFmt_IndentWidthConfigurable(t *testing.T) {
	in := "" +
		"7. {{< details \"Pay\" >}}\n" +
		"{{< uplatnica >}}\n" +
		"{{< uf-field slot=\"a\" >}}\n" +
		"{{< /uf-field >}}\n" +
		"{{< /uplatnica >}}\n" +
		"{{< /details >}}\n"
	// base = marker width 3 (no prose); depth adds 4 per level.
	want := "" +
		"7. {{< details \"Pay\" >}}\n" +
		"   {{< uplatnica >}}\n" +
		"       {{< uf-field slot=\"a\" >}}\n" +
		"       {{< /uf-field >}}\n" +
		"   {{< /uplatnica >}}\n" +
		"   {{< /details >}}\n"
	if got := scfmtCfg(in, 4); got != want {
		t.Errorf("width=4 got:\n%s\nwant:\n%s", got, want)
	}
	if scfmtCfg(want, 4) != want {
		t.Errorf("width=4 not idempotent")
	}
}

// TestSCFmt_ExcludeShortcodeSubtree: an excluded shortcode's whole subtree —
// including a multi-line opener and inner tags — is emitted verbatim, while the
// enclosing block's closer is still re-indented.
func TestSCFmt_ExcludeShortcodeSubtree(t *testing.T) {
	in := "" +
		"1. {{< details \"Pay\" >}}\n" +
		"{{< uplatnica\n" +
		"service=\"x\"\n" +
		">}}\n" +
		"{{< uf-field slot=\"a\" >}}text{{< /uf-field >}}\n" +
		"{{< /uplatnica >}}\n" +
		"{{< /details >}}\n"
	// uplatnica subtree stays verbatim (col 0); only the details closer moves.
	want := "" +
		"1. {{< details \"Pay\" >}}\n" +
		"{{< uplatnica\n" +
		"service=\"x\"\n" +
		">}}\n" +
		"{{< uf-field slot=\"a\" >}}text{{< /uf-field >}}\n" +
		"{{< /uplatnica >}}\n" +
		"   {{< /details >}}\n"
	if got := scfmtCfg(in, 2, "uplatnica"); got != want {
		t.Errorf("exclude got:\n%s\nwant:\n%s", got, want)
	}
	if scfmtCfg(want, 2, "uplatnica") != want {
		t.Errorf("exclude not idempotent")
	}
}

// TestSCFmt_CompleteTagWithTrailingContentNoCascade: a pure-tag line that is a
// COMPLETE shortcode tag followed by trailing content — e.g.
// "{{<figure … >}}</center>" — must not be mistaken for a multi-line opener.
// Misreading it swallows the next line ({{< /step >}}) as a fake tag terminator,
// corrupting depth so a later list-inline block never pops its stack, cascading
// spurious indentation onto subsequent standalone shortcodes. Real pattern from
// personal/police-certificate (a {{< steps >}} block with centered figures).
func TestSCFmt_CompleteTagWithTrailingContentNoCascade(t *testing.T) {
	in := "" +
		"{{< steps >}}\n" +
		"\n" +
		"{{< step >}}\n" +
		"{{<figure src=\"b.jpg\" width=250 >}}</center>\n" +
		"{{< /step >}}\n" +
		"\n" +
		"{{< /steps >}}\n" +
		"\n" +
		"1. {{< details \"X\" >}}\n" +
		"   content\n" +
		"   {{< /details >}}\n" +
		"\n" +
		"{{< steps >}}\n"
	// Everything is already correctly positioned; the trailing standalone
	// {{< steps >}} must stay at column 0 (no cascade).
	if got := scfmt(in); got != in {
		t.Errorf("complete-tag-with-trailing-content caused cascade:\ngot:\n%s\nwant:\n%s", got, in)
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
