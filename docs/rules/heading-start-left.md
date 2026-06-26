# Heading at the left margin

`heading-start-left`

> ATX headings should start at the left margin (no leading indentation)

- **Default severity:** warning
- **Fix:** safe autofix, applied by `doclint lint --fix` and `doclint fmt`

## How to fix

An ATX heading indented from the left margin is at best cosmetic clutter and at worst a lost heading. With 1-3 leading columns the heading still renders but markdownlint's MD023 flags the stray indent. With 4+ leading columns CommonMark/Goldmark (the parser Hugo uses) reparses the line as an indented code block, so the heading disappears and renders as monospaced code text instead. The fix removes the leading whitespace so the heading starts at column 1, and is idempotent (a left-aligned heading no longer matches). The fix is withheld when the heading is nested inside a list item, because there the indentation is structural — dedenting would pull the heading out of the list.

## Example

Flagged:

```markdown
  ## Indented heading
```

Fixed:

```markdown
## Indented heading
```

---

_Generated from `doclint` rule metadata — run `doclint docs` to refresh; do not edit by hand._
