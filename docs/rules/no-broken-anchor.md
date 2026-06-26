# Valid in-page anchor links

`no-broken-anchor`

> in-page anchor links must point at a heading in the same page

- **Default severity:** warning
- **Fix:** no automatic fix — surfaced for a human to resolve

## How to fix

A markdown link to a pure fragment — [text](#some-heading) — jumps to the heading whose auto-generated id matches that fragment. Unlike a Hugo {{< relref >}} (which fails the build when its target is missing), a raw #fragment link renders fine even when no such heading exists: the jump silently does nothing. This rule computes every heading's id the way Hugo's default (GitHub-style) anchorize does — lowercase, drop punctuation, spaces to hyphens, Unicode-aware so Cyrillic headings work — honoring an explicit '{#custom-id}' and the -1/-2 suffixes Hugo adds to duplicate headings, then flags any in-page link whose fragment is not among them. No autofix: the intended target cannot be guessed. Cross-page 'page#frag' links and links inside fenced code blocks are not checked.

## Example

Flagged:

```markdown
## Overview

See the [overview](#overvew).
```

Fixed:

```markdown
## Overview

See the [overview](#overview).
```

---

_Generated from `doclint` rule metadata — run `doclint docs` to refresh; do not edit by hand._
