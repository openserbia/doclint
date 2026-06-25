# doclint

A fast, single-binary linter, autofixer, and formatter for a Hugo site's
**markdown content** and **data files**, driven by built-in and user-defined
custom rules. Run it as a pre-deploy gate — like `golangci-lint`, but for content.

## Why

Hugo sites accumulate ad-hoc content checks (a script for a rendering gotcha,
another for frontmatter/SEO). `doclint` replaces them with one tool: simple
rules are declared in YAML (no recompile), complex rules are built in, and
everything shares one config, one autofix engine, and one output format.

## Install

```bash
go install github.com/openserbia/doclint/cmd/doclint@latest
```

Or download a prebuilt binary from the Releases page.

## Usage

Point `doclint` at the directories you want linted (typically `content` and
`data`):

```bash
# Report findings (no changes); non-zero exit on errors
doclint lint content data

# Apply safe autofixes in place
doclint lint --fix content

# Also apply unsafe fixes (may change meaning)
doclint lint --fix --unsafe-fixes content

# List files whose fixes would change them, without writing
doclint lint --diff content

# Normalize markdown spacing (idempotent)
doclint fmt content
doclint fmt --check content   # CI gate: non-zero if any file would change

# Machine-readable output
doclint lint --format json content

# Discover rules
doclint list
doclint explain details-blank-line
```

`doclint` walks every file under the paths you pass, so scope it to your content
and data directories (or use `ignore` globs in config).

## Rules

### Built-in

- **details-blank-line** — a collapsible block written as raw HTML needs a blank
  line after the closing summary tag, or the parser (the same one Hugo uses)
  swallows the inner markdown as raw HTML and it never renders. `doclint` flags
  this and inserts the blank line (a safe fix). Flagged vs. fixed:

  ```markdown
  <details><summary>More</summary>
  - this list will NOT render
  ```

  ```markdown
  <details><summary>More</summary>

  - this list renders correctly
  ```

- **table-column-count** — every row of a GFM pipe table must have the same
  number of columns as its header. A ragged row makes the renderer silently drop
  or pad cells, so the table no longer says what the author wrote. Reported as an
  error (no autofix — the intended cell boundaries are ambiguous). `fmt`
  separately re-aligns the columns of *well-formed* tables.

- **no-missing-space-atx** (markdownlint MD018) — an ATX heading needs a space
  after its `#` run; `#Heading` (no space) is not a heading in CommonMark/Goldmark
  and renders as literal text, so the heading is silently lost. `doclint` flags
  this and inserts a single space (a safe fix); `fmt` applies it too. A digit
  right after the hashes (`#1`) is left alone as a likely hashtag. Flagged vs.
  fixed:

  ```markdown
  #Heading
  ```

  ```markdown
  # Heading
  ```

- **heading-start-left** (markdownlint MD023) — an ATX heading should start at
  the left margin. 1-3 leading spaces are merely cosmetic but still flagged; 4+
  leading spaces turn the line into an indented code block, so the heading is
  lost entirely. `doclint` flags this (a warning) and dedents the heading back to
  column 1 (a safe fix); `fmt` applies it too. The fix is withheld when the
  heading is structurally nested inside a list item, because dedenting would
  de-nest it. Flagged vs. fixed:

  ```markdown
    ## Indented heading
  ```

  ```markdown
  ## Indented heading
  ```

- **blanks-around-fences** (markdownlint MD031) — a fenced code block should have
  a blank line before its opening ` ``` ` (or `~~~`) and after its closing
  delimiter. A fence butted directly against a paragraph can fail to be
  recognized as a code block, so the content renders as prose. `doclint` flags
  this (a warning) and inserts the missing blank line (a safe fix); `fmt` applies
  it too. A fence at the very start or end of the file is exempt. Flagged vs.
  fixed (shown with `~~~` for the surrounding example fence):

  ~~~markdown
  text
  ```
  code
  ```
  more
  ~~~

  ~~~markdown
  text

  ```
  code
  ```

  more
  ~~~

- **blanks-around-lists** (markdownlint MD032) — a list block should have a blank
  line before its first item and after its last item. A list line butted directly
  under a paragraph is folded into it as a lazy continuation (so no list renders),
  and a paragraph butted directly under the last item is absorbed into that item.
  `doclint` flags this (a warning) and inserts the missing blank line (a safe fix);
  `fmt` applies it too. Fenced code and frontmatter are skipped, and a list at the
  very start or end of the file is exempt. Flagged vs. fixed:

  ```markdown
  Intro paragraph
  - first item
  - second item
  Outro paragraph
  ```

  ```markdown
  Intro paragraph

  - first item
  - second item

  Outro paragraph
  ```

- **blanks-around-headings** (markdownlint MD022) — an ATX heading (`# Heading`)
  or setext heading (a text line underlined by `===` or `---`) should have a blank
  line both above and below it. The surrounding blank is mostly structural
  hygiene, but a setext underline only parses as a heading when its text line is a
  paragraph, and some list adjacencies need the blank to render as a heading at
  all. `doclint` flags this (a warning) and inserts the missing blank line (a safe
  fix); `fmt` applies it too. Fenced code and frontmatter are skipped (so a YAML
  `---` is never mistaken for a setext underline), and the document's first and
  last lines are exempt. A heading nested inside a list item is left alone
  (dedenting/de-nesting risk), and the setext above-blank is only added at a
  structural boundary so a multi-line setext heading is never split. Flagged vs.
  fixed:

  ```markdown
  Intro paragraph
  # Heading
  Body text
  ```

  ```markdown
  Intro paragraph

  # Heading

  Body text
  ```

- **fenced-code-language** (markdownlint MD040) — an opening code fence
  (` ``` ` or `~~~`) should name a language (` ```go `, ` ```bash `, ` ```json `,
  …). Without one, Hugo's Chroma highlighter has nothing to highlight and the
  block renders as an unstyled plain code box — a quality/hygiene issue, not lost
  content. `doclint` flags this (a warning) with **no autofix**: the correct
  language cannot be inferred from the code, so add it by hand. Closing
  delimiters carry no info string and are ignored. Flagged (give it a language —
  use `text` if none fits):

  ~~~markdown
  ```
  echo hello
  ```
  ~~~

- **no-alt-text** (markdownlint MD045) — an inline image whose alt text is empty
  or only whitespace (`![](url)` or `![ ](url)`) should instead describe the
  image. The image still renders, but a screen reader announces nothing for it
  and search engines lose the textual signal the alt attribute carries — a real
  accessibility and SEO defect on a public multilingual content site. `doclint`
  flags this (a warning) at the `!`, with **no autofix**: meaningful alt text
  must be authored by a human in the page's language. Image syntax inside an
  inline code span (`` `![](url)` ``) or a fenced code block is illustrative,
  renders no image, and is ignored. Flagged (add a description):

  ```markdown
  ![](/images/skadarlija.jpg)
  ```

- **no-trailing-spaces** (markdownlint MD009) — a line should not end in stray
  trailing spaces. Exactly **two** trailing spaces are a markdown hard line break
  (`<br>`) and are deliberately left alone; everything else is suspect: a single
  trailing space is invisible and renders as nothing, three or more collapse back
  to the same two-space break (so the extras are meaningless), and a
  whitespace-only line has no content for a break to attach to. `doclint` flags
  these (a warning) with **no autofix**: the `fmt` pass refuses to strip trailing
  whitespace because a blanket trim would silently delete the two-space hard break
  this rule protects, so the line is surfaced for a human to fix. Lines inside a
  fenced code block are significant content and are ignored. Flagged (the `·`
  marks a trailing space) — `end·` and `tail···`, but not `break··`.

### Custom (declarative)

Define rules in `.doclint.yaml` with no recompile. Supported types: `required`,
`length`, `not_equal`, `match`, `deny` — scoped by a path `glob`, optionally
skipping drafts.

## Configuration

`doclint` discovers `.doclint.yaml` by walking up from the working directory:

```yaml
default: standard            # all | standard | none
enable: []                   # force-enable specific rules by name
disable: []                  # force-disable specific rules by name
settings:
  details-blank-line:
    severity: error
ignore:
  - "node_modules/**"
custom:
  - id: frontmatter-description-required
    type: required
    glob: "content/**/*.md"
    field: description
    skip_drafts: true
    severity: error
  - id: seo-description-length
    type: length
    glob: "content/**/*.md"
    field: description
    min: 120
    max: 160
    severity: warning
```

`enable` force-enables specific rules by name regardless of `default`; `disable` force-disables them.

### Inline suppression

```markdown
<!-- doclint-disable-next-line details-blank-line -->
```

Unused suppressions are reported as warnings.

## Autofix safety

Fixes are tagged **safe** or **unsafe** (inspired by Ruff). `lint --fix` applies
safe fixes only; `--unsafe-fixes` opts into the rest. Plain `lint` never mutates.

## Output and exit codes

`--format human` (default, colored) or `--format json`. Exit `0` when clean, `1`
on error-severity findings (warnings are advisory; use `--max-warnings N` to
tighten), `2` on a configuration or internal error.

## License

MIT — see [LICENSE](LICENSE).
