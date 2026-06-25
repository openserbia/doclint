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
