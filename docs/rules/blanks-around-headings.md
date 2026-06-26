# blanks-around-headings

> headings should be surrounded by blank lines

- **Default severity:** warning
- **Fix:** safe autofix, applied by `doclint lint --fix` and `doclint fmt`

## How to fix

An ATX heading ("# Heading") or setext heading (a line of text underlined by "===" or "---") should have a blank line both above and below it. The surrounding blank is largely structural hygiene, but a setext underline only parses as a heading when the text line above it is a paragraph, and some list adjacencies likewise need the blank to render as a heading at all. This rule reports each missing surrounding blank and inserts one (a safe, idempotent, content-neutral fix). Fenced code and frontmatter are skipped (so a YAML "---" is never mistaken for a setext underline or thematic break), and the document's first and last lines are exempt. The setext above-check is withheld when the line above is ordinary paragraph text, since a setext heading's text can span multiple lines and inserting a blank there would split the heading. The defect is a Warning: heading boundaries are detected line-by-line, and a Warning keeps a rare mis-detection from hard-blocking a deploy.

---

_Generated from `doclint` rule metadata — run `doclint docs` to refresh; do not edit by hand._
