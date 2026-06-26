# blanks-around-lists

> lists should be surrounded by blank lines

- **Default severity:** warning
- **Fix:** safe autofix, applied by `doclint lint --fix` and `doclint fmt`

## How to fix

A list block needs a blank line before its first item and after its last item. When a list line is butted directly beneath a paragraph, CommonMark/Goldmark (the parser Hugo uses) folds it into that paragraph as a lazy continuation line and no list renders; when a paragraph is butted directly beneath the last item, it is absorbed into that item. This rule finds each maximal list region — a run of list-item lines, their indented continuation lines, and the single blank lines between items of a loose list — and reports a missing blank line on either edge, inserting one (a safe, idempotent fix). Frontmatter and fenced code are skipped. The defect is a Warning: the region boundaries are detected line-by-line, so an unusual lazy continuation could be mis-attributed, and a Warning keeps that from hard-blocking a deploy.

---

_Generated from `doclint` rule metadata — run `doclint docs` to refresh; do not edit by hand._
