# doclint

[![CI](https://github.com/openserbia/doclint/actions/workflows/ci.yml/badge.svg)](https://github.com/openserbia/doclint/actions/workflows/ci.yml)
[![Release](https://github.com/openserbia/doclint/actions/workflows/release.yml/badge.svg)](https://github.com/openserbia/doclint/actions/workflows/release.yml)
[![Latest release](https://img.shields.io/github/v/release/openserbia/doclint?sort=semver)](https://github.com/openserbia/doclint/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/openserbia/doclint.svg)](https://pkg.go.dev/github.com/openserbia/doclint)
[![Go Report Card](https://goreportcard.com/badge/github.com/openserbia/doclint)](https://goreportcard.com/report/github.com/openserbia/doclint)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

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

# Output formats: human (grouped, default) | compact (flat, CI/grep) | json
doclint lint --format compact content
doclint lint --format json content

# Scaffold a starter .doclint.yaml in the current directory (--force to overwrite)
doclint init

# Discover rules (explain tab-completes rule names)
doclint list
doclint explain details-blank-line

# Shell completion (bash|zsh|fish|powershell) — interactive install, or pipe the script
doclint completion zsh             # in a terminal: offers to install, or shows manual steps
source <(doclint completion zsh)   # or load the raw script directly
```

`doclint` walks every file under the paths you pass, so scope it to your content
and data directories (or use `ignore` globs in config).

## Rules

### Built-in

<!-- rules:start -->

| Rule | Severity | Fix | Description |
|---|---|---|---|
| [Blank line after </summary>](docs/rules/details-blank-line.md) (`details-blank-line`) | error | safe | require a blank line after </summary> so inner markdown renders |
| [Consistent table columns](docs/rules/table-column-count.md) (`table-column-count`) | error | — | require every table row to match the header's column count |
| [Space after heading hashes](docs/rules/no-missing-space-atx.md) (`no-missing-space-atx`) | error | safe | require a space after the # of an ATX heading so it renders |
| [Heading at the left margin](docs/rules/heading-start-left.md) (`heading-start-left`) | warning | safe | ATX headings should start at the left margin (no leading indentation) |
| [Blank lines around code fences](docs/rules/blanks-around-fences.md) (`blanks-around-fences`) | warning | safe | fenced code blocks should be surrounded by blank lines |
| [Blank lines around lists](docs/rules/blanks-around-lists.md) (`blanks-around-lists`) | warning | safe | lists should be surrounded by blank lines |
| [Blank lines around headings](docs/rules/blanks-around-headings.md) (`blanks-around-headings`) | warning | safe | headings should be surrounded by blank lines |
| [Code fence language](docs/rules/fenced-code-language.md) (`fenced-code-language`) | warning | — | fenced code blocks should specify a language for syntax highlighting |
| [Image alt text](docs/rules/no-alt-text.md) (`no-alt-text`) | warning | — | images should have non-empty alt text for accessibility and SEO |
| [Trailing whitespace](docs/rules/no-trailing-spaces.md) (`no-trailing-spaces`) | warning | safe | remove stray trailing spaces while preserving the two-space hard line break |
| [Valid in-page anchor links](docs/rules/no-broken-anchor.md) (`no-broken-anchor`) | warning | — | in-page anchor links must point at a heading in the same page |
| [List item body indentation](docs/rules/list-marker-indent.md) (`list-marker-indent`) | warning | unsafe | list item bodies must indent to the marker's content column |

<!-- rules:end -->

Each rule has a full rationale and a before/after example in [docs/rules/](docs/rules/), generated by the `doclint docs` command.
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
paths:                       # default lint/fmt targets when none are passed on the CLI
  - content
  - data
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

`--format human` (default — findings grouped by file, colored, each row
click-to-jump in editors), `--format compact` (one flat `path:line:col` line per
finding, for CI and grep), or `--format json`. The default `human` format
auto-falls back to `compact` when stdout is not a terminal, so piped/CI output
stays parseable and color-free. Each auto-fixable finding is marked (`*` safe,
`~` unsafe), the summary reports how many are `fixable with --fix`, and the human
output ends with a `learn how to fix:` list linking each rule that fired to its
reference page (also printed by `explain`, and a `doc_url` field in JSON). A
malformed `.doclint.yaml` fails preflight with a clear, actionable message. Exit
`0` when clean, `1` on error-severity findings
(warnings are advisory; use `--max-warnings N` to tighten), `2` on a
configuration or internal error.

## License

MIT — see [LICENSE](LICENSE).
