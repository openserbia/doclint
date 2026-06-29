# Valid in-page anchor links

`no-broken-anchor`

> in-page anchor links must point at a heading in the same page

- **Default severity:** warning
- **Fix:** no automatic fix — surfaced for a human to resolve

## How to fix

A markdown link to a pure fragment — [text](#some-heading) — jumps to the heading whose auto-generated id matches that fragment. A raw #fragment link renders fine even when no such heading exists: the jump silently does nothing. This rule computes every heading's id the way Hugo's default (GitHub-style) anchorize does — lowercase, drop punctuation, spaces to hyphens, Unicode-aware so Cyrillic headings work — honoring an explicit '{#custom-id}' and the -1/-2 suffixes Hugo adds to duplicate headings, then flags any in-page link whose fragment is not among them. It also resolves cross-page links to the target markdown file under the nearest 'content' (or content/<lang>) directory — page.md, page/index.md or page/_index.md — and flags a fragment matching no heading there. Both raw site-absolute links — [text](/section/page/#frag) — and Hugo {{< relref "/section/page" >}} shortcodes (with the #frag inside the argument or appended after the close) are checked: a relref build-fails on a bad page path but NOT on a bad fragment, so the anchor still fails silently. Anything it cannot resolve to a local file — a missing page, an external URL, a relative or named-parameter relref, or a page whose URL is remapped via slug/url front matter — is left unchecked, so the rule never guesses. No autofix: the intended target cannot be guessed. Links inside fenced code blocks are not checked.

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
