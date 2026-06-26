# no-missing-space-atx

> require a space after the # of an ATX heading so it renders

- **Default severity:** error
- **Fix:** safe autofix, applied by `doclint lint --fix` and `doclint fmt`

## How to fix

An ATX heading is 1-6 '#' characters followed by a space (or tab) and the heading text. When the text is glued straight onto the hashes ("#Heading"), CommonMark and Goldmark (the parser Hugo uses) do not recognize a heading at all: the line renders as the literal text "#Heading" and the heading is silently lost. The fix inserts a single space between the hashes and the text, which makes the heading render and is idempotent (a spaced heading no longer matches). A digit immediately after the hashes ("#1") is left alone, since that is usually a hashtag/issue reference rather than a heading.

---

_Generated from `doclint` rule metadata — run `doclint docs` to refresh; do not edit by hand._
