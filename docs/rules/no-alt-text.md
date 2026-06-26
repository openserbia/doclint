# Image alt text

`no-alt-text`

> images should have non-empty alt text for accessibility and SEO

- **Default severity:** warning
- **Fix:** no automatic fix — surfaced for a human to resolve

## How to fix

An inline image written ![](url) or ![ ](url) has empty (or whitespace-only) alt text. The image still renders, but a screen reader announces nothing for it and search engines lose the textual signal the alt attribute carries — a real accessibility and SEO defect on a public multilingual content site. This rule reports each such image at its '!'. Image syntax that appears inside an inline code span (`![](url)`) or a fenced code block is illustrative, renders no image, and is ignored. No automatic fix is offered: meaningful alt text describing the image must be authored by a human in the page's language.

## Example

Flagged:

```markdown
![](/images/skadarlija.jpg)
```

Fixed:

```markdown
![Skadarlija street at dusk](/images/skadarlija.jpg)
```

---

_Generated from `doclint` rule metadata — run `doclint docs` to refresh; do not edit by hand._
