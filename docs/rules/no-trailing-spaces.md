# no-trailing-spaces

> remove stray trailing spaces while preserving the two-space hard line break

- **Default severity:** warning
- **Fix:** safe autofix, applied by `doclint lint --fix` and `doclint fmt`

## How to fix

Trailing spaces at the end of a line are invisible and usually accidental. CommonMark gives exactly two trailing spaces a single meaning — a hard line break (<br>) — so this rule never flags a two-space run; that is intentional formatting. A single trailing space (an invisible stray that renders as nothing) and a whitespace-only line (no preceding content for a break to attach to) are unambiguous, so each carries a safe autofix that strips it — and because the fix is targeted, it never touches the two-space hard break. A run of three or more is flagged WITHOUT a fix: the renderer collapses it back to a two-space break, so whether the author meant a (sloppy) break or stray spaces is ambiguous and a human should decide. Lines inside a fenced code block are significant content and are ignored.

---

_Generated from `doclint` rule metadata — run `doclint docs` to refresh; do not edit by hand._
