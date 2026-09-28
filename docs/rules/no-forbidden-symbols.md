# Forbidden symbols

`no-forbidden-symbols`

> avoid em dashes, middle dots and semicolons in Markdown text

- **Default severity:** warning
- **Fix:** no automatic fix — surfaced for a human to resolve

## How to fix

The em dash (—), middle dot (·) and semicolon (;) are forbidden in Markdown text, including frontmatter. Each occurrence is reported separately. Fenced and inline code are ignored because these characters can be part of literal examples, and so is the semicolon that closes an HTML character reference such as `&nbsp;`. Rewrite the surrounding sentence by hand: there is no automatic replacement that preserves its meaning.

## Example

Flagged:

```markdown
First point — second point · third point; fourth point
```

Fixed:

```markdown
First point, second point, third point, and fourth point
```

---

_Generated from `doclint` rule metadata — run `doclint docs` to refresh; do not edit by hand._
