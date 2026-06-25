# doclint — Design Spec

- **Date:** 2026-06-25
- **Status:** Approved (design); pending implementation plan
- **Repo:** `openserbia/doclint`

## 1. Summary

`doclint` is a fast, single-binary Go CLI that lints, auto-fixes, and formats a
Hugo site's **markdown content** and **data files** against built-in *and*
user-defined custom rules. It is the clean, scalable successor to ad-hoc rule
scripts that commonly accrete around a Hugo project — typically a mix of small
JS and shell linters with no shared config or output.

It runs as a **pre-deploy gate** (locally and, optionally, in CI) the way
`golangci-lint` guards Go code.

## 2. Motivation

A maturing Hugo site tends to grow one-off content checks: a JS script for a
markdown rendering gotcha, a shell script for frontmatter/SEO constraints, plus
a generic markdown style linter. They work but are unmaintainable: multiple
languages, ad-hoc string surgery, no shared config, no unified output, no
safe/unsafe fix distinction, and adding a rule means hand-writing a new script.

The goal is one tool where:

- simple rules are **declared in YAML** (no recompile),
- complex rules are **written in Go** against a small interface,
- everything shares one config, one fix engine, and one output format,
- and the architecture is seamed so a **data-file format** (now) and an **LSP**
  (later) are additive, not rewrites.

## 3. Goals / Non-goals

### Goals
- Lint markdown content (`content/**/*.md`) and Hugo data files
  (`data/**`, `config/**` — YAML/TOML/JSON).
- Hybrid rule model: declarative (config) + programmatic (Go).
- Autofix with a **safe/unsafe** safety tier (Ruff model).
- A deterministic, idempotent **formatter** (`fmt`) for spacing/whitespace.
- **Dry-run** by default: `lint` never mutates; `--diff` previews fixes.
- One discoverable config file; enable/disable/severity per rule.
- Inline suppression with warn-on-unused.
- Human + JSON output; non-zero exit for CI gating.
- Cross-platform binary releases via **GoReleaser**; maintained **CHANGELOG**.
- Format-agnostic core so new formats / an LSP are additive.

### Non-goals (v1)
- Linting Go **templates** (`layouts/**`) — needs a real template parser; out of scope.
- External HTTP **link checking** — already well served by dedicated tools; not duplicated.
- Generic markdown **style** rules (MD0xx) — keep delegating to an existing
  markdown style linter initially; absorb later only if it earns its keep.
- LSP server, SARIF/GitHub-annotation output, baseline files — **Phase 2**.

## 4. Prior art & build-vs-reuse

- **`gomarklint`** (Go, Hugo-focused, single binary, fast) is the closest tool,
  but its custom-rule support is undocumented/absent, it has no formatter, no
  documented autofix, frontmatter handling is "under review," and no JSON/SARIF.
  Our differentiator — a **custom-rule engine** with autofix + formatter — is
  exactly what it lacks. Build is justified.
- **`golangci-lint` v2** — adopt: separate `run`(lint) vs `fmt` commands where
  formatters also surface during lint; `linters.default: all|standard|none` +
  `enable`/`disable` + per-rule `settings`; config-relative paths; exclusion
  presets with warn-on-unused.
- **Ruff** — adopt: **safe** fixes by default, **unsafe** gated behind
  `--unsafe-fixes`, fix-safety overridable per rule.
- **Biome v2.4 / 2026 trend** — defer but design for: SARIF + JSON output,
  baseline files, LSP-over-stdio, GitHub Actions annotations.

## 5. Architecture

Format-agnostic core; markdown and data files are **plugins into** the engine,
not the engine itself.

```
cmd/doclint            # Cobra CLI entrypoint (thin)
internal/cli           # Cobra command wiring (root, lint, fmt, explain, list)
pkg/document           # Document{Format, Raw, Lines, Frontmatter/Data, Body, AST(lazy)}
                       # + parser registry keyed by Format
pkg/rule               # Rule interface, registry, declarative-rule interpreter,
                       # builtin programmatic rules
pkg/config             # .doclint.yaml schema + loader + discovery
pkg/engine             # discovery, parallel scheduling, suppression, fix applier,
                       # severity, exit codes — knows nothing about markdown
pkg/report             # reporters: human (colored), json
```

### 5.1 Document model

```go
type Format string // "markdown" | "data"

type Document struct {
    Path        string
    Format      Format
    Raw         []byte
    Lines       []Line          // fence-aware helpers (source view)
    Frontmatter map[string]any  // markdown: parsed frontmatter; data: whole file
    Body        []byte          // markdown content after frontmatter
    // ast built lazily, markdown only
}
```

Parsers register themselves by `Format`; the engine asks the registry to build
the right `Document` for each discovered path. Goldmark — the **same parser
Hugo uses** — guarantees the AST matches what Hugo renders. Data files reuse the
same parsed-map view as frontmatter, so declarative rules work on both.

### 5.2 Rule interface

```go
type Severity int // Info | Warning | Error

type Meta struct {
    Name        string     // stable id, e.g. "details-blank-line"
    Description string
    Detail      string     // long help for `explain`
    Severity    Severity   // default; overridable in config
    Formats     []Format   // which formats it applies to
    Safety      FixSafety  // Safe | Unsafe | NoFix
}

type Rule interface {
    Meta() Meta
    Check(doc *Document, report func(Finding))
}

type Finding struct {
    Rule     string
    Path     string
    Line     int
    Col      int
    Message  string
    Severity Severity
    Safety   FixSafety
    Fixes    []TextEdit // optional; byte-offset edits into Document.Raw
}
```

Fixes are first-class `TextEdit`s. `lint --fix`, `fmt`, `--diff`, and (Phase 2)
LSP quick-fixes are all the **same data** viewed differently.

## 6. Rule model (hybrid)

### 6.1 Declarative rules (YAML, no recompile)
Authored in the config `custom:` block; interpreted by a generic engine over the
parsed-map view. Covers the easy 80% — frontmatter + data-file constraints:

- `required` — key must exist and be non-empty (optionally skip drafts).
- `length` — value char length within `[min, max]` (e.g. SEO description 120–160).
- `not_equal` — `fieldA != fieldB` (e.g. `description != lead`).
- `match` / `deny` — value matches / must not match a regex.
- scoping: `glob` (e.g. `content/guides/**`) + optional `skip_drafts`.

### 6.2 Programmatic rules (Go)
Implement `Rule`; compiled in. Cover the hard 20%:

- `details-blank-line` — fence-aware scan: every literal `</summary>` must end its
  line and be followed by a blank line (else the inner markdown is swallowed as
  an HTML block and never renders). Exempts the `{{< details >}}` shortcode form.
  Emits a **safe** `TextEdit` that inserts the blank line / splits glued content.
- room for: shortcode validity, internal-link/asset existence, image alt-text.

## 7. CLI

Built on Cobra, mirroring `golangci-lint` v2 ergonomics.

| Command | Behavior |
|---|---|
| `doclint lint [paths…]` | Report findings; **never mutates** (the dry-run / "list all problems"); exit non-zero on Error. |
| `doclint lint --fix` | Apply **safe** fixes in place. |
| `doclint lint --fix --unsafe-fixes` | Also apply unsafe fixes. |
| `doclint lint --diff` | Print the patch fixes *would* make; write nothing. |
| `doclint fmt [paths…]` | Deterministic spacing/whitespace normalization (idempotent). |
| `doclint fmt --check` / `--diff` | Dry-run: exit non-zero if unformatted / show patch. |
| `doclint explain <rule>` | Rule docs, rationale, examples. |
| `doclint list` | Catalog of rules (built-in + custom) with status. |

Global flags: `--config`, `--format human|json`, `--no-color`, `--quiet`.

## 8. Formatter (`fmt`)

Deterministic, idempotent whitespace pass — the markdown analog of `gofmt`:
strip trailing whitespace, ensure a single final newline, collapse 3+ blank
lines, and apply the always-safe `</summary>` blank-line fix. Fence-aware (never
touches code-block interiors). Shares the `TextEdit` engine with `lint --fix`;
"format" fixes are just always-safe edits that `fmt` auto-applies.

## 9. Config — `.doclint.yaml`

Discoverable upward from CWD; paths resolve relative to the config file.

```yaml
# .doclint.yaml
default: standard            # all | standard | none
enable: [details-blank-line]
disable: [some-noisy-rule]

settings:
  details-blank-line:
    severity: error

ignore:
  - "node_modules/**"
  - "content/**/_index.md"

custom:
  - id: frontmatter-description-required
    type: required
    glob: "content/**/*.md"
    field: description
    skip_drafts: true
    severity: error

  - id: seo-description-length
    type: length
    glob: "content/guides/**/*.md"
    field: description
    min: 120
    max: 160
    severity: warning

  - id: description-not-lead
    type: not_equal
    glob: "content/**/*.md"
    fields: [description, lead]
    severity: warning
```

Inline suppression: `<!-- doclint-disable <rule> -->` (block) and
`<!-- doclint-disable-next-line <rule> -->`; engine warns on **unused**
suppressions.

## 10. Output & exit codes

- `human` (default): colored `path:line:col [rule] severity message`, summary footer.
- `json`: stable machine schema (array of `Finding`).
- Exit `0` clean, `1` on Error-severity findings, `2` on internal/config error.
  **Warnings are advisory** — they don't fail the build by default
  (`--max-warnings N` to tighten).

## 11. Distribution & releases

- **GoReleaser** (`.goreleaser.yaml`): cross-platform static binaries
  (linux/darwin amd64+arm64), archives, checksums, GitHub Release on tag `v*`.
  Optional Docker image to follow the org's container pattern (Phase 2).
- **Release workflow** (`.github/workflows/release.yml`): on tag push, run
  GoReleaser.
- **Consumption by a Hugo site:** pin a version via
  `go install github.com/openserbia/doclint/cmd/doclint@vX.Y.Z` (or download the
  released binary) and wire it into the site's task runner, replacing any
  ad-hoc lint scripts.

## 12. Changelog

- **`CHANGELOG.md`** in [Keep a Changelog](https://keepachangelog.com) format, SemVer.
- **Conventional Commits**.
- GoReleaser generates **grouped release notes** from commits between tags
  (features / fixes / others); the `Unreleased` section is promoted on release.

## 13. Testing

- **Golden tests per rule:** input `.md`/data fixture → expected findings, plus
  expected `--fix` / `fmt` output (`testdata/<rule>/{input,want,want.fixed}`).
- **Engine tests:** discovery, ignore globs, config precedence, inline
  suppression (including unused-suppression warnings), exit codes.
- **Idempotence test:** `fmt(fmt(x)) == fmt(x)` across the corpus.
- A representative Hugo `content/` + `data/` corpus as a smoke test in CI.

## 14. Phasing

- **Phase 1 (MVP):** format-agnostic core; markdown + data-file linting; the two
  example rules (`details-blank-line` + declarative frontmatter); `lint` /
  `--fix` / `fmt`; `.doclint.yaml`; human + JSON output; inline suppression;
  GoReleaser + CHANGELOG; golden tests.
- **Phase 2:** LSP server (`pkg/lsp`, thin engine adapter reading the same
  config); SARIF + GitHub-annotation reporters; baseline files; optional Docker
  image; richer built-in rules (shortcode/link/asset).

## 15. Repo conventions

Mirror the org's Go module conventions: `cmd/` + `pkg/` layout, Devbox +
Taskfile, the shared `.golangci.yml`, vendored deps. Drop the DB/migration
pieces a linter doesn't need.

## 16. Risks & open questions

- **"Universal linter framework" trap** — mitigated by shipping markdown-only
  semantics first; formats/LSP are additive packages behind stable seams.
- **Goldmark vs Hugo extensions** — Hugo enables specific Goldmark extensions
  (e.g. typographer, attributes). The markdown parser must mirror the target
  site's `markup` settings so the AST matches production. (Resolve when building
  `pkg/document` markdown parsing.)
- **`go install` from a private repo** — needs `GOPRIVATE`/auth on the runner;
  released-binary download is the fallback if that's friction.
- **Repo visibility** — defaulting **private**; trivial to open-source later.
