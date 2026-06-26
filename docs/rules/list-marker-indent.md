# List item body indentation

`list-marker-indent`

> list item bodies must indent to the marker's content column

- **Default severity:** warning
- **Fix:** unsafe autofix, applied only with `doclint lint --fix --unsafe-fixes`

## How to fix

A list item's continuation and nested content must be indented to the marker's content column — len(marker)+1: 2 spaces under a "- " bullet, 3 under "1."–"9.", 4 under "10."+. When the body is indented less than that (a common foot-gun is a 2-space body under a single-digit "1. " item, which needs 3), CommonMark/Goldmark does not attach it to the item: the nested list escapes, an ordered list splits into single-item lists, and the numbering restarts (1. 1. 1. instead of 1. 2. 3.). The Unsafe fix (--fix --unsafe-fixes) re-indents the body to the content column: content outside the nested list (a leading paragraph, or a closing shortcode that de-indented out of it) is set to the content column, and the nested list block is shifted as a whole so its relative nesting is preserved. Separating the two is what stops an already-correct line from being over-indented (the v0.5.0 uniform shift's bug); it stays Unsafe because an inconsistent body can warrant a human's eye on the diff.

## Example

Flagged:

```markdown
1. {{< details "Doc" >}}
  - body under-indented (2 spaces under a "1. " item)
  {{< /details >}}
1. Next item — restarts, rendering "1." again
```

Fixed:

```markdown
1. {{< details "Doc" >}}
   - body at the content column (3 spaces)
   {{< /details >}}
1. Next item — renders as "2."
```

---

_Generated from `doclint` rule metadata — run `doclint docs` to refresh; do not edit by hand._
