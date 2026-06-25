# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `lint` command: report findings with `--format human|json`, exit non-zero on errors.
- `lint --fix` with safe/unsafe fix tiers (`--unsafe-fixes`) and `--diff` preview.
- `fmt` command: idempotent, fence-aware markdown spacing normalizer (`--check`/`--diff`).
- Built-in `details-blank-line` rule with a safe autofix.
- Built-in `table-column-count` rule: flags GFM table rows whose column count differs from the header.
- `fmt` aligns well-formed GFM table columns (shared per-column widths, preserved alignment colons); malformed tables are left untouched.
- Declarative custom rules in `.doclint.yaml`: `required`, `length`, `not_equal`, `match`, `deny`.
- Markdown (frontmatter) and data-file (YAML/TOML/JSON) linting.
- Inline suppression (`<!-- doclint-disable-next-line <rule> -->`) with unused-directive warnings.
- `list` and `explain` commands; discoverable `.doclint.yaml` configuration.
- Cross-platform release binaries via GoReleaser.
