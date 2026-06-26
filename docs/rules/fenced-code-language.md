# Code fence language

`fenced-code-language`

> fenced code blocks should specify a language for syntax highlighting

- **Default severity:** warning
- **Fix:** no automatic fix — surfaced for a human to resolve

## How to fix

A fenced code block (``` or ~~~) whose opening delimiter has an empty info string declares no language. Hugo's Chroma highlighter then has nothing to highlight, so the block renders as an unstyled plain preformatted box. The content is not lost — this is a quality/hygiene issue, not a correctness break — but a language tag (```go, ```bash, ```json …) makes code blocks readable. This rule reports each opening fence that omits a language. The correct language cannot be inferred from the code, so no automatic fix is offered; add the language by hand. Closing delimiters never carry an info string and are ignored.

## Example

Flagged:

```markdown
~~~
echo hello
~~~
```

Fixed:

```markdown
~~~bash
echo hello
~~~
```

---

_Generated from `doclint` rule metadata — run `doclint docs` to refresh; do not edit by hand._
