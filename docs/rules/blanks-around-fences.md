# blanks-around-fences

> fenced code blocks should be surrounded by blank lines

- **Default severity:** warning
- **Fix:** safe autofix, applied by `doclint lint --fix` and `doclint fmt`

## How to fix

A fenced code block (``` or ~~~) needs a blank line before its opening delimiter and after its closing delimiter. When a fence is butted directly against a preceding or following paragraph, CommonMark/Goldmark (the parser Hugo uses) can fail to recognize it as a code block, so the fenced content renders as ordinary prose instead of preformatted code. This rule reports each delimiter that is missing its surrounding blank line and inserts one (a safe fix); the insertion is content-neutral and idempotent. The document's first and last lines are exempt, since a fence at the very start or end of the file has no adjacent content to separate from.

---

_Generated from `doclint` rule metadata — run `doclint docs` to refresh; do not edit by hand._
