# Blank line after </summary>

`details-blank-line`

> require a blank line after </summary> so inner markdown renders

- **Default severity:** error
- **Fix:** safe autofix, applied by `doclint lint --fix` and `doclint fmt`

## How to fix

Goldmark parses <details><summary>…</summary> as an HTML block that ends at the next blank line. If content or markdown follows </summary> on the same line or the very next line, it is captured as raw HTML and never rendered. The fix inserts a blank line (and splits any content glued onto the </summary> line).

## Example

Flagged:

```markdown
<details><summary>More</summary>
- this item is swallowed as raw HTML
```

Fixed:

```markdown
<details><summary>More</summary>

- this item renders as a list
```

---

_Generated from `doclint` rule metadata — run `doclint docs` to refresh; do not edit by hand._
