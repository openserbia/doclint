# doclint MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `doclint` — a fast, single-binary Go CLI that lints, auto-fixes, and formats a Hugo site's markdown content and data files against built-in and user-defined rules, as a pre-deploy gate.

**Architecture:** Format-agnostic engine core. `pkg/document` turns a file into a `Document` (raw bytes, fence-aware lines, parsed frontmatter/data map). `pkg/rule` defines a `Rule` interface plus a declarative-rule interpreter and built-in Go rules; findings carry first-class `TextEdit` fixes tagged Safe/Unsafe. `pkg/engine` discovers files, runs rules in parallel, applies suppression, and applies fixes / diffs. `pkg/report` renders human/JSON. `internal/cli` wires Cobra commands; `cmd/doclint` is a thin entrypoint.

**Tech Stack:** Go 1.26 (vendored), Cobra, `adrg/frontmatter`, `yaml.v3`, `pelletier/go-toml/v2`, `bmatcuk/doublestar/v4`, `pmezard/go-difflib`, `golang.org/x/sync`, `google/go-cmp` (test). Devbox + Taskfile + shared `.golangci.yml`. GoReleaser for releases.

**Conventions:** Conventional Commits. **NEVER** add a `Co-Authored-By` trailer. All committed content is **generic** — no private project or infra names. Module path: `github.com/openserbia/doclint`.

---

## File Structure

```
doclint/
├── go.mod, go.sum
├── devbox.json
├── .golangci.yml                     # copied verbatim from the org baseline
├── .goreleaser.yaml
├── Taskfile.yml
├── .gitignore
├── README.md
├── CHANGELOG.md
├── LICENSE
├── .doclint.yaml                     # dogfood config for this repo's own docs
├── .github/
│   ├── workflows/ci.yml
│   ├── workflows/release.yml
│   └── dependabot.yml
├── cmd/doclint/main.go               # thin: build Cobra root, Execute
├── internal/cli/
│   ├── root.go                       # root cmd + persistent flags + version
│   ├── lint.go                       # lint (+ --fix/--unsafe-fixes/--diff)
│   ├── fmtcmd.go                     # fmt (+ --check/--diff)
│   ├── explain.go                    # explain <rule>
│   └── list.go                       # list
├── pkg/document/
│   ├── document.go                   # Format, Line, Document, SplitLines
│   ├── registry.go                   # parser registry
│   ├── markdown.go                   # markdown parser (frontmatter + body)
│   ├── data.go                       # YAML/TOML/JSON data-file parser
│   └── *_test.go
├── pkg/rule/
│   ├── rule.go                       # Severity, FixSafety, TextEdit, Meta, Finding, Rule
│   ├── registry.go                   # rule registry
│   ├── declarative.go                # declarative-rule interpreter
│   ├── builtin/details.go            # details-blank-line rule
│   └── *_test.go
├── pkg/config/
│   ├── config.go                     # .doclint.yaml schema + load + discover
│   └── config_test.go
├── pkg/engine/
│   ├── fix.go                        # ApplyEdits + UnifiedDiff
│   ├── suppress.go                   # inline suppression + unused detection
│   ├── format.go                     # fmt whitespace normalizer
│   ├── engine.go                     # discovery, classify, run, fix/diff
│   └── *_test.go
└── pkg/report/
    ├── report.go                     # Reporter interface
    ├── human.go
    ├── json.go
    └── *_test.go
```

**Package dependency direction (no cycles):** `document` ← `rule` ← {`config`, `report`, `engine`} ← `internal/cli` ← `cmd`. `engine` also imports `config`, `report`, `document`.

---

## Task 1: Repo scaffold

**Files:**
- Create: `go.mod`, `devbox.json`, `.golangci.yml`, `Taskfile.yml`, `.gitignore`, `README.md`, `CHANGELOG.md`, `LICENSE`, `cmd/doclint/main.go`

- [ ] **Step 1: Create `go.mod`**

```
module github.com/openserbia/doclint

go 1.26
```

- [ ] **Step 2: Create `devbox.json`** (mirror the org pin set; drop DB tools)

```json
{
  "$schema": "https://raw.githubusercontent.com/jetify-com/devbox/0.13.6/.schema/devbox.schema.json",
  "packages": {
    "github:openserbia/go-flake#go_1_26_4":            "",
    "github:openserbia/go-flake#golangci-lint_2_12_2": "",
    "github:openserbia/go-flake#gofumpt_0_10_0":       "",
    "github:openserbia/go-flake#govulncheck_1_3_0":    "",
    "go-task": "latest",
    "gci": "latest",
    "goreleaser": "latest"
  },
  "env": {
    "GOPRIVATE": "github.com/openserbia/*",
    "GOROOT": "$DEVBOX_PROJECT_ROOT/.devbox/nix/profile/default/share/go"
  }
}
```

- [ ] **Step 3: Create `.golangci.yml`** — copy the org baseline verbatim from `openserbia/go-template/.golangci.yml` (version "2", `modules-download-mode: vendor`, the full linter/formatter set). Do not modify; append service overrides only if a later task needs them.

- [ ] **Step 4: Create `Taskfile.yml`** (CLI variant — no migrate/docker include yet)

```yaml
# https://taskfile.dev
version: '3'

vars:
  ROOT_REPO: github.com/openserbia
  PACKAGE_NAME:
    sh: grep -m 1 "^module" go.mod | awk '{print $2}'

tasks:
  deps:
    desc: Download, tidy, vendor deps + govulncheck
    sources: [go.mod, go.sum]
    cmds:
      - go mod download
      - go mod tidy
      - go mod vendor
      - govulncheck ./...
    generates: [vendor/modules.txt]
    method: timestamp

  fmt:
    desc: Format go code
    deps: [deps]
    cmds:
      - gci write -s standard -s default -s "prefix({{.ROOT_REPO}})" -s "prefix({{.PACKAGE_NAME}})" --skip-generated cmd internal pkg
      - gofumpt -l -w .

  lint:
    desc: Run go linters
    deps: [fmt]
    cmds:
      - golangci-lint run

  test:
    desc: Run unit tests
    deps: [deps]
    cmds:
      - go test -mod vendor ./...

  build:
    desc: Build the doclint binary
    deps: [deps]
    cmds:
      - go build -mod vendor -o bin/doclint ./cmd/doclint

  default:
    cmds:
      - task -l
```

- [ ] **Step 5: Create `.gitignore`**

```
/bin/
/dist/
/vendor/
.devbox/
.task/
*.cov
doclint
```

- [ ] **Step 6: Create `LICENSE`** — MIT, copyright `OpenSerbia`. Use the standard MIT text.

- [ ] **Step 7: Create `CHANGELOG.md`** (Keep a Changelog skeleton)

```markdown
# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project scaffold.
```

- [ ] **Step 8: Create `README.md`** (one-paragraph stub; full usage in Task 15)

```markdown
# doclint

A fast, single-binary linter, autofixer and formatter for a Hugo site's
markdown content and data files, with built-in and user-defined custom rules.
Runs as a pre-deploy gate, like `golangci-lint` for Go.

Status: under construction. See `docs/superpowers/plans/` for the build plan.
```

- [ ] **Step 9: Create `cmd/doclint/main.go`** (compiles; real wiring in Task 13)

```go
// Command doclint lints, autofixes and formats Hugo markdown content and data
// files against built-in and user-defined rules.
package main

import "fmt"

// Build-time version metadata (set via -ldflags by GoReleaser).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Replaced by Cobra Execute() in Task 13.
	fmt.Printf("doclint %s (commit %s, built %s)\n", version, commit, date)
}
```

- [ ] **Step 10: Initialize deps and verify build**

Run: `task deps && task build`
Expected: `bin/doclint` builds; `./bin/doclint` prints `doclint dev (commit none, built unknown)`.

- [ ] **Step 11: Verify lint is green**

Run: `task lint`
Expected: PASS (no findings on the stub).

- [ ] **Step 12: Commit**

```bash
git add -A
git commit -m "chore: scaffold doclint Go module (devbox, taskfile, golangci, ci stubs)"
```

---

## Task 2: Document model — Format, Line, fence-aware SplitLines

**Files:**
- Create: `pkg/document/document.go`
- Test: `pkg/document/document_test.go`

- [ ] **Step 1: Write the failing test**

```go
package document

import "testing"

func TestSplitLines_TracksFenceState(t *testing.T) {
	raw := []byte("a\n```\nin fence\n```\nb\n")
	lines := SplitLines(raw)

	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5", len(lines))
	}
	want := []struct {
		text    string
		inFence bool
	}{
		{"a", false},
		{"```", false},      // fence delimiter is not "in fence"
		{"in fence", true},  // interior
		{"```", false},      // closing delimiter
		{"b", false},
	}
	for i, w := range want {
		if lines[i].Text != w.text || lines[i].InFence != w.inFence {
			t.Errorf("line %d = {%q, inFence=%v}, want {%q, %v}",
				i, lines[i].Text, lines[i].InFence, w.text, w.inFence)
		}
	}
	if lines[0].Num != 1 || lines[2].Start != 6 {
		t.Errorf("offsets wrong: line0.Num=%d line2.Start=%d", lines[0].Num, lines[2].Start)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/document/ -run TestSplitLines -v`
Expected: FAIL — `undefined: SplitLines`.

- [ ] **Step 3: Implement `pkg/document/document.go`**

```go
// Package document turns a file's bytes into a Document: raw content,
// fence-aware source lines, and a parsed frontmatter/data map. It is the
// format-agnostic substrate every rule reads from.
package document

import (
	"regexp"
	"sync"
)

// Format identifies how a file is parsed and which rules apply to it.
type Format string

const (
	Markdown Format = "markdown"
	Data     Format = "data"
)

// Line is one source line with byte offsets and fenced-code-block state.
type Line struct {
	Num     int    // 1-based line number
	Text    string // line content without the trailing newline
	Start   int    // byte offset of the line start in Document.Raw
	End     int    // byte offset just past Text (before the newline)
	InFence bool   // true when the line is INSIDE a fenced code block
}

// Document is the parsed view of a single file.
type Document struct {
	Path        string
	Format      Format
	Raw         []byte
	Lines       []Line
	Frontmatter map[string]any // markdown: parsed frontmatter; data: whole file
	Body        []byte         // markdown content after frontmatter (nil for data)
	BodyOffset  int            // byte offset in Raw where Body begins (0 for data)
}

var fenceRe = regexp.MustCompile("^[ \\t]*(```|~~~)")

// SplitLines splits raw into fence-aware Lines. A fence delimiter line toggles
// the in-fence state but is itself reported with InFence=false; only the lines
// strictly between an opening and closing delimiter are InFence=true.
func SplitLines(raw []byte) []Line {
	var lines []Line
	inFence := false
	start := 0
	num := 1
	for i := 0; i <= len(raw); i++ {
		if i < len(raw) && raw[i] != '\n' {
			continue
		}
		text := string(raw[start:i])
		isFence := fenceRe.MatchString(text)
		ln := Line{Num: num, Text: text, Start: start, End: i, InFence: inFence && !isFence}
		lines = append(lines, ln)
		if isFence {
			inFence = !inFence
		}
		start = i + 1
		num++
		if i == len(raw) {
			break
		}
	}
	// A trailing newline produces a final empty line; drop it so line counts
	// match editors (which don't show a phantom line after the last newline).
	if len(lines) > 1 && lines[len(lines)-1].Text == "" && len(raw) > 0 && raw[len(raw)-1] == '\n' {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// sync is imported for future lazy-AST use; keep the seam reserved.
var _ = sync.Once{}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/document/ -run TestSplitLines -v`
Expected: PASS.

- [ ] **Step 5: Remove the unused `sync` seam if golangci flags it**

If `task lint` complains about the unused `sync` import/var, delete the `import "sync"` line and the `var _ = sync.Once{}` line (the lazy AST is a Phase-2 concern). Re-run `task lint`.

- [ ] **Step 6: Commit**

```bash
git add pkg/document/
git commit -m "feat(document): fence-aware line splitter and Document model"
```

---

## Task 3: Rule core types + registry

**Files:**
- Create: `pkg/rule/rule.go`, `pkg/rule/registry.go`
- Test: `pkg/rule/rule_test.go`

- [ ] **Step 1: Write the failing test**

```go
package rule

import "testing"

func TestSeverity_RoundTrip(t *testing.T) {
	for _, s := range []Severity{Info, Warning, Error} {
		parsed, err := ParseSeverity(s.String())
		if err != nil {
			t.Fatalf("ParseSeverity(%q): %v", s.String(), err)
		}
		if parsed != s {
			t.Errorf("round-trip %v -> %q -> %v", s, s.String(), parsed)
		}
	}
	if _, err := ParseSeverity("bogus"); err == nil {
		t.Error("expected error for bogus severity")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/rule/ -run TestSeverity -v`
Expected: FAIL — `undefined: Severity`.

- [ ] **Step 3: Implement `pkg/rule/rule.go`**

```go
// Package rule defines the Rule interface, severity/fix-safety vocabulary, and
// the Finding/TextEdit types every rule emits. Fixes are first-class byte-offset
// edits so lint --fix, fmt, and --diff all consume the same data.
package rule

import (
	"fmt"

	"github.com/openserbia/doclint/pkg/document"
)

// Severity orders findings from advisory to blocking.
type Severity int

const (
	Info Severity = iota
	Warning
	Error
)

func (s Severity) String() string {
	switch s {
	case Info:
		return "info"
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

// ParseSeverity converts a config string into a Severity.
func ParseSeverity(s string) (Severity, error) {
	switch s {
	case "info":
		return Info, nil
	case "warning":
		return Warning, nil
	case "error":
		return Error, nil
	default:
		return Info, fmt.Errorf("invalid severity %q", s)
	}
}

// FixSafety describes whether a fix preserves meaning.
type FixSafety int

const (
	NoFix  FixSafety = iota // no automatic fix
	Safe                    // applied by --fix and fmt
	Unsafe                  // applied only with --unsafe-fixes
)

// TextEdit replaces Raw[Start:End] with NewText. Offsets index Document.Raw.
type TextEdit struct {
	Start   int
	End     int
	NewText string
}

// Meta is a rule's static descriptor.
type Meta struct {
	Name        string
	Description string            // one line, shown by `list`
	Detail      string            // long help, shown by `explain`
	Severity    Severity          // default; config may override
	Formats     []document.Format // which formats this rule applies to
	Safety      FixSafety         // safety of fixes this rule emits
}

// AppliesTo reports whether the rule runs on the given format.
func (m Meta) AppliesTo(f document.Format) bool {
	for _, x := range m.Formats {
		if x == f {
			return true
		}
	}
	return false
}

// Finding is one reported issue.
type Finding struct {
	Rule     string     `json:"rule"`
	Path     string     `json:"path"`
	Line     int        `json:"line"`
	Col      int        `json:"col"`
	Message  string     `json:"message"`
	Severity Severity   `json:"severity"`
	Safety   FixSafety  `json:"-"`
	Fixes    []TextEdit `json:"-"`
}

// Rule inspects a Document and reports findings.
type Rule interface {
	Meta() Meta
	Check(doc *document.Document, report func(Finding))
}
```

- [ ] **Step 4: Run `task deps` then the test**

Run: `task deps && go test ./pkg/rule/ -run TestSeverity -v`
Expected: PASS.

- [ ] **Step 5: Implement `pkg/rule/registry.go`**

```go
package rule

// Registry holds rules by name in registration order.
type Registry struct {
	rules map[string]Rule
	order []string
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{rules: map[string]Rule{}}
}

// Register adds a rule; a duplicate name panics (programmer error).
func (r *Registry) Register(rule Rule) {
	name := rule.Meta().Name
	if _, ok := r.rules[name]; ok {
		panic("duplicate rule registration: " + name)
	}
	r.rules[name] = rule
	r.order = append(r.order, name)
}

// Get returns the rule with name, if present.
func (r *Registry) Get(name string) (Rule, bool) {
	rule, ok := r.rules[name]
	return rule, ok
}

// All returns the rules in registration order.
func (r *Registry) All() []Rule {
	out := make([]Rule, 0, len(r.order))
	for _, n := range r.order {
		out = append(out, r.rules[n])
	}
	return out
}
```

- [ ] **Step 6: Add a registry test to `pkg/rule/rule_test.go`**

```go
type fakeRule struct{ name string }

func (f fakeRule) Meta() Meta { return Meta{Name: f.name, Formats: []document.Format{document.Markdown}} }
func (f fakeRule) Check(_ *document.Document, _ func(Finding)) {}

func TestRegistry_Order(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeRule{"b"})
	r.Register(fakeRule{"a"})
	got := []string{r.All()[0].Meta().Name, r.All()[1].Meta().Name}
	if got[0] != "b" || got[1] != "a" {
		t.Errorf("order = %v, want [b a]", got)
	}
}
```

Add `"github.com/openserbia/doclint/pkg/document"` to the test file imports.

- [ ] **Step 7: Run tests**

Run: `go test ./pkg/rule/ -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add pkg/rule/
git commit -m "feat(rule): Rule interface, severity/fix-safety types, registry"
```

---

## Task 4: Markdown parser (frontmatter + body)

**Files:**
- Create: `pkg/document/markdown.go`, `pkg/document/registry.go`
- Test: `pkg/document/markdown_test.go`

- [ ] **Step 1: Write the failing test**

```go
package document

import (
	"testing"
)

func TestParseMarkdown_FrontmatterAndBody(t *testing.T) {
	raw := []byte("---\ntitle: Hello\ndraft: true\n---\n\n# Body\n")
	doc, err := ParseMarkdown("post.md", raw)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if doc.Format != Markdown {
		t.Errorf("Format = %q, want markdown", doc.Format)
	}
	if doc.Frontmatter["title"] != "Hello" {
		t.Errorf("title = %v, want Hello", doc.Frontmatter["title"])
	}
	if doc.Frontmatter["draft"] != true {
		t.Errorf("draft = %v, want true", doc.Frontmatter["draft"])
	}
	if string(doc.Body) == "" || doc.BodyOffset == 0 {
		t.Errorf("body not extracted: offset=%d body=%q", doc.BodyOffset, doc.Body)
	}
}

func TestParseMarkdown_NoFrontmatter(t *testing.T) {
	raw := []byte("# Just a heading\n")
	doc, err := ParseMarkdown("x.md", raw)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(doc.Frontmatter) != 0 {
		t.Errorf("expected empty frontmatter, got %v", doc.Frontmatter)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/document/ -run TestParseMarkdown -v`
Expected: FAIL — `undefined: ParseMarkdown`.

- [ ] **Step 3: Add the dependency and implement `pkg/document/markdown.go`**

```go
package document

import (
	"bytes"
	"fmt"

	"github.com/adrg/frontmatter"
)

// ParseMarkdown builds a markdown Document: it extracts YAML/TOML/JSON
// frontmatter into a map and records where the body begins.
func ParseMarkdown(path string, raw []byte) (*Document, error) {
	doc := &Document{
		Path:        path,
		Format:      Markdown,
		Raw:         raw,
		Lines:       SplitLines(raw),
		Frontmatter: map[string]any{},
	}

	var matter map[string]any
	rest, err := frontmatter.Parse(bytes.NewReader(raw), &matter)
	if err != nil {
		return nil, fmt.Errorf("parse frontmatter in %s: %w", path, err)
	}
	if matter != nil {
		doc.Frontmatter = matter
	}
	doc.Body = rest
	// BodyOffset = where `rest` starts inside raw. frontmatter.Parse returns the
	// trailing body verbatim, so locate it from the end to stay delimiter-agnostic.
	doc.BodyOffset = len(raw) - len(rest)
	return doc, nil
}
```

- [ ] **Step 4: Implement `pkg/document/registry.go`**

```go
package document

import "fmt"

// ParseFunc builds a Document from a file's bytes.
type ParseFunc func(path string, raw []byte) (*Document, error)

var parsers = map[Format]ParseFunc{
	Markdown: ParseMarkdown,
	Data:     ParseData,
}

// Parse dispatches to the parser registered for format.
func Parse(format Format, path string, raw []byte) (*Document, error) {
	p, ok := parsers[format]
	if !ok {
		return nil, fmt.Errorf("no parser for format %q", format)
	}
	return p(path, raw)
}
```

(`ParseData` is implemented in Task 5; until then this file will not compile, so do Task 5 before running `task lint`. To keep Task 4 self-contained, temporarily comment out the `Data: ParseData` map entry, run the Task-4 test, then restore it in Task 5.)

- [ ] **Step 5: Vendor + run the test** (with the `Data` entry commented out)

Run: `task deps && go test ./pkg/document/ -run TestParseMarkdown -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/document/ go.mod go.sum
git commit -m "feat(document): markdown frontmatter+body parser and parser registry"
```

---

## Task 5: Data-file parser (YAML/TOML/JSON)

**Files:**
- Create: `pkg/document/data.go`
- Test: `pkg/document/data_test.go`

- [ ] **Step 1: Write the failing test**

```go
package document

import "testing"

func TestParseData_AllFormats(t *testing.T) {
	cases := map[string][]byte{
		"x.yaml": []byte("name: a\ncount: 2\n"),
		"x.yml":  []byte("name: a\ncount: 2\n"),
		"x.toml": []byte("name = \"a\"\ncount = 2\n"),
		"x.json": []byte(`{"name":"a","count":2}`),
	}
	for path, raw := range cases {
		doc, err := ParseData(path, raw)
		if err != nil {
			t.Fatalf("ParseData(%s): %v", path, err)
		}
		if doc.Format != Data {
			t.Errorf("%s: format = %q, want data", path, doc.Format)
		}
		if doc.Frontmatter["name"] != "a" {
			t.Errorf("%s: name = %v, want a", path, doc.Frontmatter["name"])
		}
	}
}

func TestParseData_Unsupported(t *testing.T) {
	if _, err := ParseData("x.txt", []byte("hi")); err == nil {
		t.Error("expected error for unsupported data extension")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/document/ -run TestParseData -v`
Expected: FAIL — `undefined: ParseData`.

- [ ] **Step 3: Implement `pkg/document/data.go`**

```go
package document

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// ParseData loads a YAML/TOML/JSON data file into Document.Frontmatter (the
// whole file becomes the parsed map). Data files have no body or AST.
func ParseData(path string, raw []byte) (*Document, error) {
	doc := &Document{
		Path:        path,
		Format:      Data,
		Raw:         raw,
		Lines:       SplitLines(raw),
		Frontmatter: map[string]any{},
	}
	m := map[string]any{}
	switch ext := filepath.Ext(path); ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("parse yaml %s: %w", path, err)
		}
	case ".toml":
		if err := toml.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("parse toml %s: %w", path, err)
		}
	case ".json":
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("parse json %s: %w", path, err)
		}
	default:
		return nil, fmt.Errorf("unsupported data extension %q", ext)
	}
	doc.Frontmatter = m
	return doc, nil
}
```

- [ ] **Step 4: Restore the `Data: ParseData` entry** in `pkg/document/registry.go` (uncomment from Task 4).

- [ ] **Step 5: Vendor + run tests + lint**

Run: `task deps && go test ./pkg/document/ -v && task lint`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/document/ go.mod go.sum
git commit -m "feat(document): YAML/TOML/JSON data-file parser"
```

---

## Task 6: Fix engine — ApplyEdits + UnifiedDiff

**Files:**
- Create: `pkg/engine/fix.go`
- Test: `pkg/engine/fix_test.go`

- [ ] **Step 1: Write the failing test**

```go
package engine

import (
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/rule"
)

func TestApplyEdits_OrdersAndSplices(t *testing.T) {
	src := []byte("hello world")
	// Edits given out of order; applied right-to-left after sorting.
	edits := []rule.TextEdit{
		{Start: 6, End: 11, NewText: "there"},
		{Start: 0, End: 5, NewText: "HI"},
	}
	got, err := ApplyEdits(src, edits)
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	if string(got) != "HI there" {
		t.Errorf("got %q, want %q", got, "HI there")
	}
}

func TestApplyEdits_RejectsOverlap(t *testing.T) {
	src := []byte("abcdef")
	edits := []rule.TextEdit{
		{Start: 0, End: 3, NewText: "x"},
		{Start: 2, End: 5, NewText: "y"},
	}
	if _, err := ApplyEdits(src, edits); err == nil {
		t.Error("expected overlap error")
	}
}

func TestUnifiedDiff(t *testing.T) {
	d := UnifiedDiff("a.md", []byte("one\ntwo\n"), []byte("one\n2\n"))
	if !strings.Contains(d, "-two") || !strings.Contains(d, "+2") {
		t.Errorf("diff missing changes:\n%s", d)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/engine/ -run "TestApplyEdits|TestUnifiedDiff" -v`
Expected: FAIL — `undefined: ApplyEdits`.

- [ ] **Step 3: Implement `pkg/engine/fix.go`**

```go
// Package engine discovers files, runs rules in parallel, applies inline
// suppression, and applies fixes or renders diffs.
package engine

import (
	"fmt"
	"sort"

	"github.com/pmezard/go-difflib/difflib"

	"github.com/openserbia/doclint/pkg/rule"
)

// ApplyEdits returns src with every edit applied. Edits must not overlap; they
// are applied last-to-first so earlier offsets stay valid during splicing.
func ApplyEdits(src []byte, edits []rule.TextEdit) ([]byte, error) {
	if len(edits) == 0 {
		return src, nil
	}
	sorted := make([]rule.TextEdit, len(edits))
	copy(sorted, edits)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start < sorted[j].Start })

	for i := 1; i < len(sorted); i++ {
		if sorted[i].Start < sorted[i-1].End {
			return nil, fmt.Errorf("overlapping edits at offset %d", sorted[i].Start)
		}
	}
	out := make([]byte, len(src))
	copy(out, src)
	for i := len(sorted) - 1; i >= 0; i-- {
		e := sorted[i]
		if e.Start < 0 || e.End > len(out) || e.Start > e.End {
			return nil, fmt.Errorf("edit out of range [%d:%d] (len %d)", e.Start, e.End, len(out))
		}
		out = append(out[:e.Start], append([]byte(e.NewText), out[e.End:]...)...)
	}
	return out, nil
}

// UnifiedDiff renders a unified diff of before vs after for one file.
func UnifiedDiff(path string, before, after []byte) string {
	d := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(before)),
		B:        difflib.SplitLines(string(after)),
		FromFile: "a/" + path,
		ToFile:   "b/" + path,
		Context:  3,
	}
	text, _ := difflib.GetUnifiedDiffString(d)
	return text
}
```

- [ ] **Step 4: Vendor + run tests**

Run: `task deps && go test ./pkg/engine/ -run "TestApplyEdits|TestUnifiedDiff" -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/engine/ go.mod go.sum
git commit -m "feat(engine): non-overlapping TextEdit applier and unified diff"
```

---

## Task 7: `details-blank-line` rule (ports the JS `<details>` checker)

**Files:**
- Create: `pkg/rule/builtin/details.go`, `pkg/rule/builtin/builtin.go`
- Test: `pkg/rule/builtin/details_test.go`

- [ ] **Step 1: Write the failing test**

```go
package builtin

import (
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

func findings(t *testing.T, r rule.Rule, raw []byte) []rule.Finding {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out []rule.Finding
	r.Check(doc, func(f rule.Finding) { out = append(out, f) })
	return out
}

func TestDetails_MissingBlankLine(t *testing.T) {
	raw := []byte("<details><summary>x</summary>\n- item\n")
	got := findings(t, DetailsBlankLine{}, raw)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if got[0].Safety != rule.Safe || len(got[0].Fixes) != 1 {
		t.Errorf("expected one safe fix, got safety=%v fixes=%d", got[0].Safety, len(got[0].Fixes))
	}
}

func TestDetails_OkWithBlankLine(t *testing.T) {
	raw := []byte("<details><summary>x</summary>\n\n- item\n")
	if got := findings(t, DetailsBlankLine{}, raw); len(got) != 0 {
		t.Fatalf("got %d findings, want 0", len(got))
	}
}

func TestDetails_IgnoresFencedCode(t *testing.T) {
	raw := []byte("```html\n<details><summary>x</summary>\ncode\n```\n")
	if got := findings(t, DetailsBlankLine{}, raw); len(got) != 0 {
		t.Fatalf("got %d findings inside fence, want 0", len(got))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/rule/builtin/ -run TestDetails -v`
Expected: FAIL — `undefined: DetailsBlankLine`.

- [ ] **Step 3: Implement `pkg/rule/builtin/details.go`**

```go
// Package builtin holds programmatic (Go) rules.
package builtin

import (
	"strings"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

const closeTag = "</summary>"

// DetailsBlankLine enforces a blank line after a literal </summary>. Goldmark
// (the parser Hugo uses) treats <details><summary>…</summary> as an HTML block
// that runs until the next blank line; without it the inner markdown is
// swallowed as raw HTML and never renders.
type DetailsBlankLine struct{}

func (DetailsBlankLine) Meta() rule.Meta {
	return rule.Meta{
		Name:        "details-blank-line",
		Description: "require a blank line after </summary> so inner markdown renders",
		Detail: "Goldmark parses <details><summary>…</summary> as an HTML block " +
			"that ends at the next blank line. If content or markdown follows " +
			"</summary> on the same line or the very next line, it is captured as " +
			"raw HTML and never rendered. The fix inserts a blank line (and splits " +
			"any content glued onto the </summary> line).",
		Severity: rule.Error,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.Safe,
	}
}

func (d DetailsBlankLine) Check(doc *document.Document, report func(rule.Finding)) {
	lines := doc.Lines
	for i, ln := range lines {
		if ln.InFence || !strings.Contains(ln.Text, closeTag) {
			continue
		}
		cut := strings.LastIndex(ln.Text, closeTag) + len(closeTag)
		trailing := strings.TrimSpace(ln.Text[cut:])

		if trailing != "" {
			// Content glued after </summary> on the same line.
			indent := ln.Text[:len(ln.Text)-len(strings.TrimLeft(ln.Text, " \t"))]
			insertAt := ln.Start + cut
			report(rule.Finding{
				Rule:     d.Meta().Name,
				Path:     doc.Path,
				Line:     ln.Num,
				Col:      cut + 1,
				Message:  "content must not follow </summary> on the same line; put it on its own line after a blank line",
				Severity: rule.Error,
				Safety:   rule.Safe,
				Fixes: []rule.TextEdit{{
					Start:   insertAt,
					End:     ln.End,
					NewText: "\n\n" + indent + trailing,
				}},
			})
			continue
		}

		// </summary> ends the line: require the next line to be blank.
		if i+1 < len(lines) && strings.TrimSpace(lines[i+1].Text) != "" {
			report(rule.Finding{
				Rule:     d.Meta().Name,
				Path:     doc.Path,
				Line:     ln.Num,
				Col:      len(ln.Text) + 1,
				Message:  "missing blank line after </summary>; inner markdown will not render",
				Severity: rule.Error,
				Safety:   rule.Safe,
				Fixes: []rule.TextEdit{{
					Start:   ln.End, // just before the newline char
					End:     ln.End,
					NewText: "\n",
				}},
			})
		}
	}
}
```

- [ ] **Step 4: Implement `pkg/rule/builtin/builtin.go`** (registers built-ins)

```go
package builtin

import "github.com/openserbia/doclint/pkg/rule"

// Register adds every built-in rule to reg.
func Register(reg *rule.Registry) {
	reg.Register(DetailsBlankLine{})
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./pkg/rule/builtin/ -v`
Expected: PASS (all three cases).

- [ ] **Step 6: Add a fix-application golden test** to `details_test.go`

```go
import "github.com/openserbia/doclint/pkg/engine"

func TestDetails_FixProducesBlankLine(t *testing.T) {
	raw := []byte("<details><summary>x</summary>\n- item\n")
	got := findings(t, DetailsBlankLine{}, raw)
	fixed, err := engine.ApplyEdits(raw, got[0].Fixes)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	want := "<details><summary>x</summary>\n\n- item\n"
	if string(fixed) != want {
		t.Errorf("fixed = %q, want %q", fixed, want)
	}
}
```

- [ ] **Step 7: Run + commit**

Run: `go test ./pkg/rule/builtin/ -v`
Expected: PASS.

```bash
git add pkg/rule/builtin/
git commit -m "feat(rule): details-blank-line rule with safe autofix"
```

---

## Task 8: Declarative rule interpreter (ports the frontmatter checker)

**Files:**
- Create: `pkg/rule/declarative.go`
- Test: `pkg/rule/declarative_test.go`

- [ ] **Step 1: Write the failing test**

```go
package rule

import (
	"testing"

	"github.com/openserbia/doclint/pkg/document"
)

func docWith(fm map[string]any, path string) *document.Document {
	return &document.Document{Path: path, Format: document.Markdown, Frontmatter: fm}
}

func run(t *testing.T, spec DeclSpec, doc *document.Document) []Finding {
	t.Helper()
	r, err := NewDeclarativeRule(spec)
	if err != nil {
		t.Fatalf("NewDeclarativeRule: %v", err)
	}
	var out []Finding
	r.Check(doc, func(f Finding) { out = append(out, f) })
	return out
}

func TestDeclarative_RequiredSkipsDraft(t *testing.T) {
	spec := DeclSpec{ID: "desc-req", Type: "required", Field: "description", SkipDrafts: true, Severity: Error}
	if got := run(t, spec, docWith(map[string]any{"draft": true}, "p.md")); len(got) != 0 {
		t.Fatalf("draft should be skipped, got %d findings", len(got))
	}
	if got := run(t, spec, docWith(map[string]any{}, "p.md")); len(got) != 1 {
		t.Fatalf("missing description should flag, got %d", len(got))
	}
}

func TestDeclarative_Length(t *testing.T) {
	spec := DeclSpec{ID: "len", Type: "length", Field: "description", Min: 5, Max: 10, Severity: Warning}
	if got := run(t, spec, docWith(map[string]any{"description": "hi"}, "p.md")); len(got) != 1 {
		t.Fatalf("too-short should flag, got %d", len(got))
	}
	if got := run(t, spec, docWith(map[string]any{"description": "just right"}, "p.md")); len(got) != 0 {
		t.Fatalf("in-range should pass, got %d", len(got))
	}
}

func TestDeclarative_NotEqual(t *testing.T) {
	spec := DeclSpec{ID: "ne", Type: "not_equal", Fields: []string{"description", "lead"}, Severity: Warning}
	doc := docWith(map[string]any{"description": "same", "lead": "same"}, "p.md")
	if got := run(t, spec, doc); len(got) != 1 {
		t.Fatalf("equal fields should flag, got %d", len(got))
	}
}

func TestDeclarative_GlobScope(t *testing.T) {
	spec := DeclSpec{ID: "g", Type: "required", Field: "description", Glob: "content/guides/**/*.md", Severity: Error}
	if got := run(t, spec, docWith(map[string]any{}, "content/blog/x.md")); len(got) != 0 {
		t.Fatalf("out-of-glob file should be skipped, got %d", len(got))
	}
	if got := run(t, spec, docWith(map[string]any{}, "content/guides/a/x.md")); len(got) != 1 {
		t.Fatalf("in-glob file should be checked, got %d", len(got))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/rule/ -run TestDeclarative -v`
Expected: FAIL — `undefined: DeclSpec`.

- [ ] **Step 3: Implement `pkg/rule/declarative.go`**

```go
package rule

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/openserbia/doclint/pkg/document"
)

// DeclSpec is one user-defined rule from the config `custom:` block.
type DeclSpec struct {
	ID         string
	Type       string // required | length | not_equal | match | deny
	Glob       string // optional path scope (doublestar)
	Field      string // for required/length/match/deny
	Fields     []string
	Min, Max   int
	Pattern    string // for match/deny
	SkipDrafts bool
	Severity   Severity
}

type declRule struct {
	spec    DeclSpec
	pattern *regexp.Regexp
}

// NewDeclarativeRule compiles a DeclSpec into a Rule.
func NewDeclarativeRule(spec DeclSpec) (Rule, error) {
	r := &declRule{spec: spec}
	switch spec.Type {
	case "required", "length", "not_equal":
	case "match", "deny":
		p, err := regexp.Compile(spec.Pattern)
		if err != nil {
			return nil, fmt.Errorf("rule %s: bad pattern: %w", spec.ID, err)
		}
		r.pattern = p
	default:
		return nil, fmt.Errorf("rule %s: unknown type %q", spec.ID, spec.Type)
	}
	return r, nil
}

func (r *declRule) Meta() Meta {
	return Meta{
		Name:        r.spec.ID,
		Description: "custom rule (" + r.spec.Type + ")",
		Severity:    r.spec.Severity,
		Formats:     []document.Format{document.Markdown, document.Data},
		Safety:      NoFix,
	}
}

func (r *declRule) Check(doc *document.Document, report func(Finding)) {
	if r.spec.Glob != "" {
		ok, _ := doublestar.Match(r.spec.Glob, doc.Path)
		if !ok {
			return
		}
	}
	if r.spec.SkipDrafts && isDraft(doc.Frontmatter) {
		return
	}
	emit := func(field, msg string) {
		report(Finding{
			Rule:     r.spec.ID,
			Path:     doc.Path,
			Line:     fieldLine(doc, field),
			Col:      1,
			Message:  msg,
			Severity: r.spec.Severity,
			Safety:   NoFix,
		})
	}

	switch r.spec.Type {
	case "required":
		if str(doc.Frontmatter[r.spec.Field]) == "" {
			emit(r.spec.Field, fmt.Sprintf("%q is required and must be non-empty", r.spec.Field))
		}
	case "length":
		v := str(doc.Frontmatter[r.spec.Field])
		if v == "" {
			return // absence is the `required` rule's job
		}
		n := len([]rune(v))
		if n < r.spec.Min {
			emit(r.spec.Field, fmt.Sprintf("%q is %d chars, minimum %d", r.spec.Field, n, r.spec.Min))
		} else if r.spec.Max > 0 && n > r.spec.Max {
			emit(r.spec.Field, fmt.Sprintf("%q is %d chars, maximum %d", r.spec.Field, n, r.spec.Max))
		}
	case "not_equal":
		if len(r.spec.Fields) == 2 {
			a, b := str(doc.Frontmatter[r.spec.Fields[0]]), str(doc.Frontmatter[r.spec.Fields[1]])
			if a != "" && a == b {
				emit(r.spec.Fields[0], fmt.Sprintf("%q and %q must differ", r.spec.Fields[0], r.spec.Fields[1]))
			}
		}
	case "match":
		v := str(doc.Frontmatter[r.spec.Field])
		if v != "" && !r.pattern.MatchString(v) {
			emit(r.spec.Field, fmt.Sprintf("%q must match /%s/", r.spec.Field, r.spec.Pattern))
		}
	case "deny":
		v := str(doc.Frontmatter[r.spec.Field])
		if v != "" && r.pattern.MatchString(v) {
			emit(r.spec.Field, fmt.Sprintf("%q must not match /%s/", r.spec.Field, r.spec.Pattern))
		}
	}
}

func isDraft(fm map[string]any) bool { b, _ := fm["draft"].(bool); return b }

// str renders a frontmatter scalar as a string ("" for nil/non-scalar).
func str(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

// fieldLine finds the source line declaring `field:` (frontmatter), else 1.
func fieldLine(doc *document.Document, field string) int {
	prefix := field + ":"
	for _, ln := range doc.Lines {
		if strings.HasPrefix(strings.TrimSpace(ln.Text), prefix) {
			return ln.Num
		}
	}
	return 1
}
```

- [ ] **Step 4: Vendor + run tests**

Run: `task deps && go test ./pkg/rule/ -run TestDeclarative -v`
Expected: PASS (all four cases).

- [ ] **Step 5: Commit**

```bash
git add pkg/rule/ go.mod go.sum
git commit -m "feat(rule): declarative custom-rule interpreter (required/length/not_equal/match/deny)"
```

---

## Task 9: Config — schema, load, discover

**Files:**
- Create: `pkg/config/config.go`
- Test: `pkg/config/config_test.go`

- [ ] **Step 1: Write the failing test**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndDiscover(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".doclint.yaml")
	body := `
default: standard
disable: [noisy]
settings:
  details-blank-line:
    severity: warning
ignore:
  - "node_modules/**"
custom:
  - id: desc-req
    type: required
    field: description
    skip_drafts: true
    severity: error
`
	if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "content")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	found, err := Discover(sub)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if found != cfgPath {
		t.Errorf("Discover = %q, want %q", found, cfgPath)
	}
	cfg, err := Load(found)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Default != "standard" || len(cfg.Custom) != 1 || cfg.Custom[0].ID != "desc-req" {
		t.Errorf("unexpected config: %+v", cfg)
	}
	if cfg.Settings["details-blank-line"].Severity != "warning" {
		t.Errorf("setting not parsed: %+v", cfg.Settings)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/config/ -v`
Expected: FAIL — `undefined: Discover`.

- [ ] **Step 3: Implement `pkg/config/config.go`**

```go
// Package config loads .doclint.yaml: rule defaults/toggles, per-rule settings,
// ignore globs, and the declarative custom-rule block.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigName is the discovered file name.
const ConfigName = ".doclint.yaml"

// RuleSetting overrides a rule's defaults.
type RuleSetting struct {
	Severity string `yaml:"severity"`
}

// CustomRule is one declarative rule (mirrors rule.DeclSpec; kept decoupled so
// config has no dependency on package rule's internals).
type CustomRule struct {
	ID         string   `yaml:"id"`
	Type       string   `yaml:"type"`
	Glob       string   `yaml:"glob"`
	Field      string   `yaml:"field"`
	Fields     []string `yaml:"fields"`
	Min        int      `yaml:"min"`
	Max        int      `yaml:"max"`
	Pattern    string   `yaml:"pattern"`
	SkipDrafts bool     `yaml:"skip_drafts"`
	Severity   string   `yaml:"severity"`
}

// Config is the parsed .doclint.yaml plus the directory it was loaded from.
type Config struct {
	Default  string                 `yaml:"default"`
	Enable   []string               `yaml:"enable"`
	Disable  []string               `yaml:"disable"`
	Settings map[string]RuleSetting `yaml:"settings"`
	Ignore   []string               `yaml:"ignore"`
	Custom   []CustomRule           `yaml:"custom"`

	Dir string `yaml:"-"` // directory of the config file (relative-path base)
}

// Default returns the built-in config used when no file is found.
func Default() *Config {
	return &Config{Default: "standard", Settings: map[string]RuleSetting{}}
}

// Load reads and parses a config file.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path) //nolint:gosec // path is the discovered config
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg := Default()
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if cfg.Settings == nil {
		cfg.Settings = map[string]RuleSetting{}
	}
	if cfg.Default == "" {
		cfg.Default = "standard"
	}
	cfg.Dir = filepath.Dir(path)
	return cfg, nil
}

// Discover walks up from start looking for ConfigName; returns "" if none.
func Discover(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, ConfigName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil // reached filesystem root
		}
		dir = parent
	}
}
```

- [ ] **Step 4: Vendor + run tests**

Run: `task deps && go test ./pkg/config/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/config/ go.mod go.sum
git commit -m "feat(config): .doclint.yaml schema, loader and upward discovery"
```

---

## Task 10: Inline suppression + unused detection

**Files:**
- Create: `pkg/engine/suppress.go`
- Test: `pkg/engine/suppress_test.go`

- [ ] **Step 1: Write the failing test**

```go
package engine

import (
	"testing"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

func TestSuppress_NextLineAndUnused(t *testing.T) {
	raw := []byte("<!-- doclint-disable-next-line details-blank-line -->\nline2\n<!-- doclint-disable-next-line other-rule -->\nline4\n")
	doc, _ := document.ParseMarkdown("t.md", raw)
	s := NewSuppressor(doc)

	// A finding on line 2 for details-blank-line is suppressed.
	f := rule.Finding{Rule: "details-blank-line", Line: 2}
	if !s.Suppressed(f) {
		t.Error("expected finding on line 2 to be suppressed")
	}
	// Nothing matched the line-4 directive -> it is unused.
	unused := s.Unused()
	if len(unused) != 1 || unused[0].Rule != "other-rule" {
		t.Errorf("unused = %+v, want one for other-rule", unused)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/engine/ -run TestSuppress -v`
Expected: FAIL — `undefined: NewSuppressor`.

- [ ] **Step 3: Implement `pkg/engine/suppress.go`**

```go
package engine

import (
	"regexp"
	"strings"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

var directiveRe = regexp.MustCompile(`<!--\s*doclint-(disable-next-line|disable-line)(\s+[^>]*?)?\s*-->`)

type directive struct {
	Rule       string // "" means all rules
	TargetLine int    // line the directive applies to
	used       bool
}

// Suppressor matches findings against inline doclint-disable directives and
// tracks which directives went unused.
type Suppressor struct {
	directives []*directive
}

// NewSuppressor scans a document for suppression directives.
func NewSuppressor(doc *document.Document) *Suppressor {
	s := &Suppressor{}
	for _, ln := range doc.Lines {
		m := directiveRe.FindStringSubmatch(ln.Text)
		if m == nil {
			continue
		}
		target := ln.Num
		if m[1] == "disable-next-line" {
			target = ln.Num + 1
		}
		rules := strings.Fields(strings.TrimSpace(m[2]))
		if len(rules) == 0 {
			s.directives = append(s.directives, &directive{Rule: "", TargetLine: target})
			continue
		}
		for _, r := range rules {
			s.directives = append(s.directives, &directive{Rule: r, TargetLine: target})
		}
	}
	return s
}

// Suppressed reports whether f is silenced, marking the matching directive used.
func (s *Suppressor) Suppressed(f rule.Finding) bool {
	for _, d := range s.directives {
		if d.TargetLine == f.Line && (d.Rule == "" || d.Rule == f.Rule) {
			d.used = true
			return true
		}
	}
	return false
}

// Unused returns findings describing directives that matched nothing.
func (s *Suppressor) Unused() []rule.Finding {
	var out []rule.Finding
	for _, d := range s.directives {
		if d.used {
			continue
		}
		name := d.Rule
		if name == "" {
			name = "all rules"
		}
		out = append(out, rule.Finding{
			Rule:     "unused-suppression",
			Line:     d.TargetLine,
			Message:  "unused doclint-disable directive for " + name,
			Severity: rule.Warning,
		})
	}
	return out
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/engine/ -run TestSuppress -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/engine/
git commit -m "feat(engine): inline suppression directives with unused detection"
```

---

## Task 11: Formatter (`fmt` whitespace normalizer)

**Files:**
- Create: `pkg/engine/format.go`
- Test: `pkg/engine/format_test.go`

- [ ] **Step 1: Write the failing test**

```go
package engine

import (
	"testing"

	"github.com/openserbia/doclint/pkg/document"
)

func format(t *testing.T, raw string) string {
	t.Helper()
	doc, err := document.ParseMarkdown("t.md", []byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return string(Format(doc))
}

func TestFormat_CollapsesBlankLinesAndFinalNewline(t *testing.T) {
	got := format(t, "a\n\n\n\nb")
	want := "a\n\nb\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormat_PreservesFencedInterior(t *testing.T) {
	in := "```\n\n\n\n```\n"
	if got := format(t, in); got != in {
		t.Errorf("fenced interior changed: got %q want %q", got, in)
	}
}

func TestFormat_Idempotent(t *testing.T) {
	in := "x\n\n\n\ny<details><summary>s</summary>\n- i\n"
	once := format(t, in)
	doc, _ := document.ParseMarkdown("t.md", []byte(once))
	twice := string(Format(doc))
	if once != twice {
		t.Errorf("not idempotent:\n once=%q\ntwice=%q", once, twice)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/engine/ -run TestFormat -v`
Expected: FAIL — `undefined: Format`.

- [ ] **Step 3: Implement `pkg/engine/format.go`**

```go
package engine

import (
	"bytes"
	"strings"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

// Format applies the deterministic, idempotent whitespace pass: it collapses
// 3+ consecutive blank lines (outside fenced code) to one, ensures a single
// trailing newline, and applies the always-safe details-blank-line fix.
// Trailing-whitespace stripping is intentionally omitted (two trailing spaces
// are a markdown hard line break).
func Format(doc *document.Document) []byte {
	// 1. Apply the safe structural fix(es) first, on the raw bytes.
	var fixes []rule.TextEdit
	(builtin.DetailsBlankLine{}).Check(doc, func(f rule.Finding) {
		if f.Safety == rule.Safe {
			fixes = append(fixes, f.Fixes...)
		}
	})
	raw := doc.Raw
	if len(fixes) > 0 {
		if out, err := ApplyEdits(raw, fixes); err == nil {
			raw = out
		}
	}

	// 2. Re-split (offsets changed) and collapse blank runs outside fences.
	lines := document.SplitLines(raw)
	var b bytes.Buffer
	blankRun := 0
	for _, ln := range lines {
		blank := strings.TrimSpace(ln.Text) == ""
		if blank && !ln.InFence {
			blankRun++
			if blankRun >= 2 {
				continue // keep at most one blank line
			}
		} else {
			blankRun = 0
		}
		b.WriteString(ln.Text)
		b.WriteByte('\n')
	}

	// 3. Single trailing newline.
	out := bytes.TrimRight(b.Bytes(), "\n")
	out = append(out, '\n')
	return out
}
```

- [ ] **Step 4: Run tests**

Run: `task deps && go test ./pkg/engine/ -run TestFormat -v`
Expected: PASS (collapse, fence-preservation, idempotence).

- [ ] **Step 5: Commit**

```bash
git add pkg/engine/
git commit -m "feat(engine): idempotent fmt whitespace normalizer (fence-aware)"
```

---

## Task 12: Reporters — human + JSON

**Files:**
- Create: `pkg/report/report.go`, `pkg/report/human.go`, `pkg/report/json.go`
- Test: `pkg/report/report_test.go`

- [ ] **Step 1: Write the failing test**

```go
package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/openserbia/doclint/pkg/rule"
)

var sample = []rule.Finding{
	{Rule: "details-blank-line", Path: "a.md", Line: 3, Col: 1, Message: "missing blank line", Severity: rule.Error},
	{Rule: "seo-len", Path: "a.md", Line: 1, Col: 1, Message: "too short", Severity: rule.Warning},
}

func TestHuman(t *testing.T) {
	var buf bytes.Buffer
	if err := (Human{NoColor: true}).Report(&buf, sample); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "a.md:3:1") || !strings.Contains(out, "[details-blank-line]") {
		t.Errorf("human output missing location/rule:\n%s", out)
	}
	if !strings.Contains(out, "1 error") || !strings.Contains(out, "1 warning") {
		t.Errorf("human output missing summary:\n%s", out)
	}
}

func TestJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := (JSON{}).Report(&buf, sample); err != nil {
		t.Fatal(err)
	}
	var got []rule.Finding
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(got) != 2 || got[0].Rule != "details-blank-line" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/report/ -v`
Expected: FAIL — `undefined: Human`.

- [ ] **Step 3: Implement `pkg/report/report.go`**

```go
// Package report renders findings as human-readable or JSON output.
package report

import (
	"io"

	"github.com/openserbia/doclint/pkg/rule"
)

// Reporter writes findings to w.
type Reporter interface {
	Report(w io.Writer, findings []rule.Finding) error
}

func counts(findings []rule.Finding) (errors, warnings, infos int) {
	for _, f := range findings {
		switch f.Severity {
		case rule.Error:
			errors++
		case rule.Warning:
			warnings++
		case rule.Info:
			infos++
		}
	}
	return
}
```

- [ ] **Step 4: Implement `pkg/report/human.go`**

```go
package report

import (
	"fmt"
	"io"

	"github.com/openserbia/doclint/pkg/rule"
)

// ANSI colors keyed by severity; disabled when NoColor is set.
const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiYellow = "\033[33m"
	ansiBlue   = "\033[34m"
	ansiDim    = "\033[2m"
)

// Human renders colored `path:line:col [rule] severity message` lines.
type Human struct{ NoColor bool }

func (h Human) color(s rule.Severity) string {
	if h.NoColor {
		return ""
	}
	switch s {
	case rule.Error:
		return ansiRed
	case rule.Warning:
		return ansiYellow
	default:
		return ansiBlue
	}
}

func (h Human) reset() string {
	if h.NoColor {
		return ""
	}
	return ansiReset
}

// Report writes each finding then a summary footer.
func (h Human) Report(w io.Writer, findings []rule.Finding) error {
	for _, f := range findings {
		if _, err := fmt.Fprintf(w, "%s:%d:%d %s[%s]%s %s%s%s %s\n",
			f.Path, f.Line, f.Col,
			h.dim(), f.Rule, h.reset(),
			h.color(f.Severity), f.Severity, h.reset(),
			f.Message,
		); err != nil {
			return err
		}
	}
	errs, warns, infos := counts(findings)
	_, err := fmt.Fprintf(w, "\n%d error(s), %d warning(s), %d info\n", errs, warns, infos)
	return err
}

func (h Human) dim() string {
	if h.NoColor {
		return ""
	}
	return ansiDim
}
```

- [ ] **Step 5: Implement `pkg/report/json.go`**

```go
package report

import (
	"encoding/json"
	"io"

	"github.com/openserbia/doclint/pkg/rule"
)

// JSON renders findings as a JSON array (stable machine schema).
type JSON struct{}

// Report marshals findings; severity is rendered via its String form.
func (JSON) Report(w io.Writer, findings []rule.Finding) error {
	type wire struct {
		Rule     string `json:"rule"`
		Path     string `json:"path"`
		Line     int    `json:"line"`
		Col      int    `json:"col"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
	}
	out := make([]wire, 0, len(findings))
	for _, f := range findings {
		out = append(out, wire{f.Rule, f.Path, f.Line, f.Col, f.Message, f.Severity.String()})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
```

Note: the JSON test unmarshals into `[]rule.Finding`; `Severity` is an int there, but the wire form emits a string. Update the test to unmarshal into the same `wire`-shaped anonymous struct, or assert on `buf.String()` containing `"severity": "error"`. Adjust `TestJSON` to:

```go
func TestJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := (JSON{}).Report(&buf, sample); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"severity": "error"`) ||
		!strings.Contains(buf.String(), `"rule": "details-blank-line"`) {
		t.Errorf("json missing fields:\n%s", buf.String())
	}
	_ = json.Valid(buf.Bytes())
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./pkg/report/ -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add pkg/report/
git commit -m "feat(report): human (colored) and JSON reporters"
```

---

## Task 13: Engine — discovery, classify, run, fix/diff

**Files:**
- Create: `pkg/engine/engine.go`
- Test: `pkg/engine/engine_test.go`

- [ ] **Step 1: Write the failing test**

```go
package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/openserbia/doclint/pkg/config"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func TestEngine_RunFindsAndFixes(t *testing.T) {
	dir := t.TempDir()
	md := filepath.Join(dir, "post.md")
	if err := os.WriteFile(md, []byte("<details><summary>x</summary>\n- item\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := rule.NewRegistry()
	builtin.Register(reg)

	cfg := config.Default()
	eng, err := New(cfg, reg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res, err := eng.Run(context.Background(), []string{dir})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Findings) != 1 || res.Findings[0].Rule != "details-blank-line" {
		t.Fatalf("findings = %+v", res.Findings)
	}
	if res.ExitCode() != 1 {
		t.Errorf("exit = %d, want 1 (one error)", res.ExitCode())
	}

	fixed, err := eng.Fix(context.Background(), []string{dir}, false /*unsafe*/, true /*dryRun*/)
	if err != nil {
		t.Fatalf("Fix: %v", err)
	}
	if len(fixed) != 1 {
		t.Fatalf("expected 1 changed file, got %d", len(fixed))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/engine/ -run TestEngine -v`
Expected: FAIL — `undefined: New`.

- [ ] **Step 3: Implement `pkg/engine/engine.go`**

```go
package engine

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar/v4"
	"golang.org/x/sync/errgroup"

	"github.com/openserbia/doclint/pkg/config"
	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

// Engine holds the resolved active rule set and config.
type Engine struct {
	cfg   *config.Config
	rules []rule.Rule
}

// Result is the outcome of a Run.
type Result struct {
	Findings []rule.Finding
}

// ExitCode is 1 when any Error-severity finding is present, else 0.
func (r *Result) ExitCode() int {
	for _, f := range r.Findings {
		if f.Severity == rule.Error {
			return 1
		}
	}
	return 0
}

// New resolves the active rules: built-ins filtered by enable/disable/default,
// plus compiled declarative rules from the config `custom:` block.
func New(cfg *config.Config, reg *rule.Registry) (*Engine, error) {
	e := &Engine{cfg: cfg}
	disabled := toSet(cfg.Disable)
	enabled := toSet(cfg.Enable)

	for _, r := range reg.All() {
		name := r.Meta().Name
		on := cfg.Default != "none"
		if enabled[name] {
			on = true
		}
		if disabled[name] {
			on = false
		}
		if on {
			e.rules = append(e.rules, applySetting(r, cfg))
		}
	}
	for _, c := range cfg.Custom {
		sev := rule.Warning
		if c.Severity != "" {
			s, err := rule.ParseSeverity(c.Severity)
			if err != nil {
				return nil, err
			}
			sev = s
		}
		dr, err := rule.NewDeclarativeRule(rule.DeclSpec{
			ID: c.ID, Type: c.Type, Glob: c.Glob, Field: c.Field, Fields: c.Fields,
			Min: c.Min, Max: c.Max, Pattern: c.Pattern, SkipDrafts: c.SkipDrafts, Severity: sev,
		})
		if err != nil {
			return nil, err
		}
		e.rules = append(e.rules, dr)
	}
	return e, nil
}

// Run lints every discovered file under paths and returns sorted findings.
func (e *Engine) Run(ctx context.Context, paths []string) (*Result, error) {
	files, err := e.discover(paths)
	if err != nil {
		return nil, err
	}
	var (
		mu  sync.Mutex
		all []rule.Finding
	)
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())
	for _, f := range files {
		f := f
		g.Go(func() error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			fs, err := e.lintFile(f.path, f.format)
			if err != nil {
				return err
			}
			mu.Lock()
			all = append(all, fs...)
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	sortFindings(all)
	return &Result{Findings: all}, nil
}

type target struct {
	path   string
	format document.Format
}

func (e *Engine) lintFile(path string, format document.Format) ([]rule.Finding, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path from discovery walk
	if err != nil {
		return nil, err
	}
	doc, err := document.Parse(format, path, raw)
	if err != nil {
		return nil, err
	}
	sup := NewSuppressor(doc)
	var out []rule.Finding
	for _, r := range e.rules {
		if !r.Meta().AppliesTo(format) {
			continue
		}
		r.Check(doc, func(f rule.Finding) {
			if f.Path == "" {
				f.Path = path
			}
			if sup.Suppressed(f) {
				return
			}
			out = append(out, f)
		})
	}
	out = append(out, sup.Unused()...)
	for i := range out {
		if out[i].Path == "" {
			out[i].Path = path
		}
	}
	return out, nil
}

// Fix lints, applies safe (and optionally unsafe) fixes per file, and either
// writes the files (dryRun=false) or returns them without writing. It returns
// the list of changed paths.
func (e *Engine) Fix(ctx context.Context, paths []string, unsafe, dryRun bool) ([]string, error) {
	files, err := e.discover(paths)
	if err != nil {
		return nil, err
	}
	var changed []string
	for _, f := range files {
		raw, err := os.ReadFile(f.path) //nolint:gosec // discovery walk
		if err != nil {
			return nil, err
		}
		doc, err := document.Parse(f.format, f.path, raw)
		if err != nil {
			return nil, err
		}
		var edits []rule.TextEdit
		for _, r := range e.rules {
			if !r.Meta().AppliesTo(f.format) {
				continue
			}
			r.Check(doc, func(fd rule.Finding) {
				if fd.Safety == rule.Safe || (unsafe && fd.Safety == rule.Unsafe) {
					edits = append(edits, fd.Fixes...)
				}
			})
		}
		if len(edits) == 0 {
			continue
		}
		out, err := ApplyEdits(raw, edits)
		if err != nil {
			return nil, err
		}
		changed = append(changed, f.path)
		if !dryRun {
			if err := os.WriteFile(f.path, out, 0o644); err != nil { //nolint:gosec // preserve mode is fine for content files
				return nil, err
			}
		}
	}
	return changed, nil
}

func (e *Engine) discover(paths []string) ([]target, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	var out []target
	seen := map[string]bool{}
	for _, root := range paths {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			format, ok := classify(p)
			if !ok || seen[p] || e.ignored(p) {
				return nil
			}
			seen[p] = true
			out = append(out, target{path: p, format: format})
			return nil
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
	return out, nil
}

func (e *Engine) ignored(p string) bool {
	for _, g := range e.cfg.Ignore {
		if ok, _ := doublestar.Match(g, p); ok {
			return true
		}
		// Also match against the path relative to the config dir, if set.
		if e.cfg.Dir != "" {
			if rel, err := filepath.Rel(e.cfg.Dir, p); err == nil {
				if ok, _ := doublestar.Match(g, filepath.ToSlash(rel)); ok {
					return true
				}
			}
		}
	}
	return false
}

// classify decides a file's Format from its extension/location.
func classify(p string) (document.Format, bool) {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".md", ".markdown":
		return document.Markdown, true
	case ".yaml", ".yml", ".toml", ".json":
		return document.Data, true
	default:
		return "", false
	}
}

func toSet(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// settingRule wraps a rule to override its default severity from config.
type settingRule struct {
	rule.Rule
	severity rule.Severity
}

func (s settingRule) Check(doc *document.Document, report func(rule.Finding)) {
	s.Rule.Check(doc, func(f rule.Finding) {
		f.Severity = s.severity
		report(f)
	})
}

func applySetting(r rule.Rule, cfg *config.Config) rule.Rule {
	set, ok := cfg.Settings[r.Meta().Name]
	if !ok || set.Severity == "" {
		return r
	}
	sev, err := rule.ParseSeverity(set.Severity)
	if err != nil {
		return r
	}
	return settingRule{Rule: r, severity: sev}
}

func sortFindings(f []rule.Finding) {
	sort.Slice(f, func(i, j int) bool {
		if f[i].Path != f[j].Path {
			return f[i].Path < f[j].Path
		}
		if f[i].Line != f[j].Line {
			return f[i].Line < f[j].Line
		}
		return f[i].Col < f[j].Col
	})
}
```

- [ ] **Step 4: Vendor + run tests**

Run: `task deps && go test ./pkg/engine/ -v`
Expected: PASS (all engine + fix + format + suppress tests).

- [ ] **Step 5: Run the full suite + lint**

Run: `task test && task lint`
Expected: PASS. (Fix any `gocritic`/`revive` findings the org config raises — e.g. wrap errors, name receivers consistently.)

- [ ] **Step 6: Commit**

```bash
git add pkg/engine/ go.mod go.sum
git commit -m "feat(engine): file discovery, parallel run, severity settings, fix/diff"
```

---

## Task 14: CLI wiring (Cobra commands)

**Files:**
- Create: `internal/cli/root.go`, `internal/cli/lint.go`, `internal/cli/fmtcmd.go`, `internal/cli/explain.go`, `internal/cli/list.go`
- Modify: `cmd/doclint/main.go`
- Test: `internal/cli/cli_test.go`

- [ ] **Step 1: Add the Cobra dependency**

Run: `go get github.com/spf13/cobra@latest && task deps`
Expected: `go.mod` lists cobra.

- [ ] **Step 2: Implement `internal/cli/root.go`**

```go
// Package cli wires the doclint Cobra command tree over the engine.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/openserbia/doclint/pkg/config"
	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

// Options holds resolved global flags.
type Options struct {
	ConfigPath string
	Format     string
	NoColor    bool
	Quiet      bool
}

// NewRootCmd builds the command tree. version/commit/date come from main.
func NewRootCmd(version, commit, date string) *cobra.Command {
	opts := &Options{}
	root := &cobra.Command{
		Use:           "doclint",
		Short:         "Lint, autofix and format Hugo markdown content and data files",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version + " (commit " + commit + ", built " + date + ")",
	}
	root.PersistentFlags().StringVar(&opts.ConfigPath, "config", "", "path to .doclint.yaml (default: discovered)")
	root.PersistentFlags().StringVar(&opts.Format, "format", "human", "output format: human|json")
	root.PersistentFlags().BoolVar(&opts.NoColor, "no-color", false, "disable colored output")
	root.PersistentFlags().BoolVar(&opts.Quiet, "quiet", false, "suppress non-finding output")

	root.AddCommand(newLintCmd(opts))
	root.AddCommand(newFmtCmd(opts))
	root.AddCommand(newExplainCmd())
	root.AddCommand(newListCmd(opts))
	return root
}

// loadConfig discovers/loads config and builds the registry of built-ins.
func loadConfig(opts *Options) (*config.Config, *rule.Registry, error) {
	reg := rule.NewRegistry()
	builtin.Register(reg)

	path := opts.ConfigPath
	if path == "" {
		found, err := config.Discover(".")
		if err != nil {
			return nil, nil, err
		}
		path = found
	}
	if path == "" {
		return config.Default(), reg, nil
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, nil, err
	}
	return cfg, reg, nil
}
```

- [ ] **Step 3: Implement `internal/cli/lint.go`**

```go
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openserbia/doclint/pkg/engine"
	"github.com/openserbia/doclint/pkg/report"
)

func newLintCmd(opts *Options) *cobra.Command {
	var (
		fix         bool
		unsafeFixes bool
		diff        bool
		maxWarn     int
	)
	cmd := &cobra.Command{
		Use:   "lint [paths...]",
		Short: "Report findings; with --fix apply safe fixes (--diff to preview)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, reg, err := loadConfig(opts)
			if err != nil {
				return err
			}
			eng, err := engine.New(cfg, reg)
			if err != nil {
				return err
			}
			ctx := context.Background()

			if fix || diff {
				changed, err := eng.Fix(ctx, args, unsafeFixes, diff)
				if err != nil {
					return err
				}
				if diff {
					for _, p := range changed {
						before, _ := os.ReadFile(p) //nolint:gosec,errcheck // best-effort preview
						_ = before
					}
				}
				if !opts.Quiet {
					fmt.Fprintf(cmd.OutOrStdout(), "%d file(s) %s\n", len(changed), verb(diff))
				}
				return nil
			}

			res, err := eng.Run(ctx, args)
			if err != nil {
				return err
			}
			rep := pickReporter(opts)
			if err := rep.Report(cmd.OutOrStdout(), res.Findings); err != nil {
				return err
			}
			if res.ExitCode() != 0 || tooManyWarnings(res, maxWarn) {
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "apply safe fixes in place")
	cmd.Flags().BoolVar(&unsafeFixes, "unsafe-fixes", false, "also apply unsafe fixes (implies --fix)")
	cmd.Flags().BoolVar(&diff, "diff", false, "print fixes as a diff without writing")
	cmd.Flags().IntVar(&maxWarn, "max-warnings", -1, "fail if warnings exceed N (-1 = never)")
	return cmd
}

func verb(diff bool) string {
	if diff {
		return "would change"
	}
	return "fixed"
}

func tooManyWarnings(res interface{ ExitCode() int }, maxWarn int) bool {
	_ = res
	return false // wired in Step 4 helper; kept simple for MVP
}

func pickReporter(opts *Options) report.Reporter {
	if opts.Format == "json" {
		return report.JSON{}
	}
	return report.Human{NoColor: opts.NoColor}
}
```

Note: the `--diff` preview above is intentionally minimal for MVP (it reports the count of files that would change). A full unified-diff print uses `engine.UnifiedDiff`; wire it in by having `engine.Fix` return per-file before/after bytes. For MVP, keep the count form and add the `--max-warnings` handling as below.

- [ ] **Step 4: Replace the `tooManyWarnings` stub** with a real check by giving the lint command the findings. Simplify: compute warnings from `res.Findings` directly. Replace the stub and its call:

```go
// in RunE, after res is obtained:
if maxWarn >= 0 {
	warns := 0
	for _, f := range res.Findings {
		if f.Severity.String() == "warning" {
			warns++
		}
	}
	if warns > maxWarn {
		os.Exit(1)
	}
}
if res.ExitCode() != 0 {
	os.Exit(1)
}
return nil
```

Delete the `tooManyWarnings` function and its earlier call.

- [ ] **Step 5: Implement `internal/cli/fmtcmd.go`**

```go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/engine"
)

func newFmtCmd(opts *Options) *cobra.Command {
	var (
		check bool
		diff  bool
	)
	cmd := &cobra.Command{
		Use:   "fmt [paths...]",
		Short: "Normalize markdown spacing (idempotent); --check/--diff for dry-run",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, reg, err := loadConfig(opts)
			if err != nil {
				return err
			}
			eng, err := engine.New(cfg, reg)
			if err != nil {
				return err
			}
			files, err := eng.MarkdownFiles(args)
			if err != nil {
				return err
			}
			changed := 0
			for _, p := range files {
				raw, err := os.ReadFile(p) //nolint:gosec // discovered path
				if err != nil {
					return err
				}
				doc, err := document.ParseMarkdown(p, raw)
				if err != nil {
					return err
				}
				out := engine.Format(doc)
				if string(out) == string(raw) {
					continue
				}
				changed++
				switch {
				case diff:
					fmt.Fprint(cmd.OutOrStdout(), engine.UnifiedDiff(p, raw, out))
				case check:
					fmt.Fprintf(cmd.OutOrStdout(), "would reformat %s\n", p)
				default:
					if err := os.WriteFile(p, out, 0o644); err != nil { //nolint:gosec // content file
						return err
					}
				}
			}
			if (check || diff) && changed > 0 {
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "exit non-zero if any file would change")
	cmd.Flags().BoolVar(&diff, "diff", false, "print formatting changes as a diff")
	return cmd
}
```

- [ ] **Step 6: Add `MarkdownFiles` to the engine** (`pkg/engine/engine.go`)

```go
// MarkdownFiles returns the discovered markdown file paths under paths.
func (e *Engine) MarkdownFiles(paths []string) ([]string, error) {
	files, err := e.discover(paths)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, f := range files {
		if f.format == document.Markdown {
			out = append(out, f.path)
		}
	}
	return out, nil
}
```

- [ ] **Step 7: Implement `internal/cli/list.go` and `internal/cli/explain.go`**

```go
// list.go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newListCmd(opts *Options) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List built-in and custom rules",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, reg, err := loadConfig(opts)
			if err != nil {
				return err
			}
			for _, r := range reg.All() {
				m := r.Meta()
				fmt.Fprintf(cmd.OutOrStdout(), "%-24s %-8s %s\n", m.Name, m.Severity, m.Description)
			}
			for _, c := range cfg.Custom {
				fmt.Fprintf(cmd.OutOrStdout(), "%-24s %-8s custom (%s)\n", c.ID, c.Severity, c.Type)
			}
			return nil
		},
	}
}
```

```go
// explain.go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openserbia/doclint/pkg/rule"
	"github.com/openserbia/doclint/pkg/rule/builtin"
)

func newExplainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "explain <rule>",
		Short: "Show a rule's rationale and examples",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := rule.NewRegistry()
			builtin.Register(reg)
			r, ok := reg.Get(args[0])
			if !ok {
				return fmt.Errorf("unknown rule %q", args[0])
			}
			m := r.Meta()
			fmt.Fprintf(cmd.OutOrStdout(), "%s [%s]\n\n%s\n\n%s\n", m.Name, m.Severity, m.Description, m.Detail)
			return nil
		},
	}
}
```

- [ ] **Step 8: Rewrite `cmd/doclint/main.go`**

```go
// Command doclint lints, autofixes and formats Hugo markdown content and data
// files against built-in and user-defined rules.
package main

import (
	"fmt"
	"os"

	"github.com/openserbia/doclint/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := cli.NewRootCmd(version, commit, date).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "doclint:", err)
		os.Exit(2)
	}
}
```

- [ ] **Step 9: Write an integration test `internal/cli/cli_test.go`**

```go
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLintCmd_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "p.md"),
		[]byte("<details><summary>x</summary>\n- i\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := NewRootCmd("test", "t", "t")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"lint", "--format", "json", dir})

	// lint calls os.Exit(1) on error findings; run in a subprocess-free way by
	// asserting on output before exit is unreachable here. Instead, use `list`
	// to validate wiring without the exit path:
	root.SetArgs([]string{"list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("details-blank-line")) {
		t.Errorf("list output missing built-in rule:\n%s", out.String())
	}
}
```

Note on testing the exit path: `lint` calls `os.Exit(1)` on findings, which is hostile to in-process tests. The engine itself is already covered in Task 13; the CLI test above validates command wiring via `list`. (A Phase-2 refactor can have `lint` return an exit code to `main` instead of calling `os.Exit`, making it unit-testable — tracked as a follow-up.)

- [ ] **Step 10: Vendor, build, run, lint**

Run: `task deps && task build && go test ./internal/... && task lint`
Expected: build succeeds; tests pass; lint clean.

- [ ] **Step 11: Manual smoke test**

```bash
mkdir -p /tmp/doclint-smoke && printf '<details><summary>x</summary>\n- item\n' > /tmp/doclint-smoke/p.md
./bin/doclint lint /tmp/doclint-smoke ; echo "exit=$?"
./bin/doclint lint --fix /tmp/doclint-smoke && cat /tmp/doclint-smoke/p.md
```
Expected: first run reports the finding and exits 1; after `--fix`, the file has a blank line after `</summary>`.

- [ ] **Step 12: Commit**

```bash
git add internal/ cmd/ pkg/engine/ go.mod go.sum
git commit -m "feat(cli): cobra commands lint/fmt/list/explain over the engine"
```

---

## Task 15: GoReleaser, CI, release workflow, dependabot

**Files:**
- Create: `.goreleaser.yaml`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `.github/dependabot.yml`

- [ ] **Step 1: Create `.goreleaser.yaml`**

```yaml
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - id: doclint
    main: ./cmd/doclint
    binary: doclint
    env: [CGO_ENABLED=0]
    goos: [linux, darwin]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

archives:
  - formats: [tar.gz]
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: checksums.txt

changelog:
  use: github
  sort: asc
  groups:
    - title: Features
      regexp: '^.*?feat(\(.+\))??!?:.+$'
      order: 0
    - title: Bug fixes
      regexp: '^.*?fix(\(.+\))??!?:.+$'
      order: 1
    - title: Others
      order: 999
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'
```

- [ ] **Step 2: Create `.github/workflows/ci.yml`**

```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:

jobs:
  build:
    runs-on: [self-hosted, ax41]
    timeout-minutes: 10
    steps:
      - uses: actions/checkout@v7
      - name: Lint + test
        run: devbox run -- task lint && devbox run -- task test
```

Note: if a self-hosted runner is not desired for this repo, change `runs-on` to `ubuntu-latest` and replace the devbox step with `actions/setup-go@v5` (go-version-file: go.mod) + `task lint && task test`. Pick one and keep it consistent with the org's other tool repos.

- [ ] **Step 3: Create `.github/workflows/release.yml`**

```yaml
name: Release
on:
  push:
    tags: ['v*']

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 4: Create `.github/dependabot.yml`**

```yaml
version: 2
updates:
  - package-ecosystem: gomod
    directory: "/"
    schedule: { interval: weekly }
  - package-ecosystem: github-actions
    directory: "/"
    schedule: { interval: weekly }
```

- [ ] **Step 5: Validate GoReleaser config locally**

Run: `goreleaser check`
Expected: `config is valid`.

- [ ] **Step 6: Commit**

```bash
git add .goreleaser.yaml .github/
git commit -m "ci: goreleaser release pipeline, CI lint/test, dependabot"
```

---

## Task 16: Dogfood config, README usage, CHANGELOG

**Files:**
- Create: `.doclint.yaml`
- Modify: `README.md`, `CHANGELOG.md`

- [ ] **Step 1: Create a minimal `.doclint.yaml`** (doclint lints its own `docs/`)

```yaml
default: standard
ignore:
  - "vendor/**"
  - "dist/**"
custom:
  - id: example-frontmatter-description-required
    type: required
    glob: "content/**/*.md"
    field: description
    skip_drafts: true
    severity: error
```

- [ ] **Step 2: Expand `README.md`** with install, usage, config, and a custom-rule example. Keep all examples GENERIC (a Hugo site, `content/posts/**`, `example.com`). Cover: `lint`, `lint --fix`, `lint --diff`, `fmt`, `fmt --check`, `list`, `explain`, the `.doclint.yaml` schema, and the safe/unsafe fix model.

- [ ] **Step 3: Update `CHANGELOG.md` `[Unreleased]`** to list the MVP feature set (linting, declarative + Go rules, autofix safe/unsafe, fmt, JSON output, inline suppression, GoReleaser).

- [ ] **Step 4: Final full check**

Run: `task lint && task test && ./bin/doclint lint docs ; echo "exit=$?"`
Expected: lint/test pass; doclint runs cleanly over its own `docs/`.

- [ ] **Step 5: Commit**

```bash
git add .doclint.yaml README.md CHANGELOG.md
git commit -m "docs: usage README, dogfood config, changelog for MVP"
```

---

## Task 17: Tag v0.1.0 and push (release the MVP)

- [ ] **Step 1: Push `main`** (use the embedded-token HTTPS URL; the SSH remote hangs)

```bash
TOKEN=$(gh auth token)
git push "https://x-access-token:${TOKEN}@github.com/openserbia/doclint.git" main:main 2>&1 | sed "s|${TOKEN}|***|g"
```

- [ ] **Step 2: Verify remote head**

Run: `gh api repos/openserbia/doclint/commits/main --jq .sha`
Expected: matches local `git rev-parse HEAD`.

- [ ] **Step 3: Tag and push the tag** (triggers the release workflow)

```bash
git tag v0.1.0
TOKEN=$(gh auth token)
git push "https://x-access-token:${TOKEN}@github.com/openserbia/doclint.git" v0.1.0 2>&1 | sed "s|${TOKEN}|***|g"
```

- [ ] **Step 4: Confirm the release ran**

Run: `gh run list --repo openserbia/doclint --workflow Release`
Expected: a Release run for `v0.1.0`; once green, `gh release view v0.1.0 --repo openserbia/doclint` shows the binaries.

---

## Downstream integration (separate, in the PRIVATE consumer repo — not committed here)

This is applied **in the Hugo site repo**, not in `doclint`, so no internal names land in this public tool. For the maintainer:

1. Add a `.doclint.yaml` to the site root porting the existing ad-hoc checks:
   - `required` `description` (skip drafts),
   - `length` `description` 120–160 scoped to the guides glob (warning),
   - `not_equal` `[description, lead]` (warning),
   - enable `details-blank-line` (error).
2. Install doclint: `go install github.com/openserbia/doclint/cmd/doclint@v0.1.0` (needs `GOPRIVATE` while the repo is private) or download the released binary.
3. Replace the `lint:frontmatter` + `lint:details` scripts in the site's task runner with `doclint lint` (and drop the two old script files once parity is confirmed by running both on the current content).
4. Keep the generic markdown style linter for now (non-goal to absorb in MVP).

---

## Self-Review

**Spec coverage** (spec §→task):
- §3 markdown + data linting → Tasks 4, 5, 13 (classify). ✓
- §5 architecture / packages → Tasks 2–14 follow the package map. ✓
- §5.2 Rule/Finding/TextEdit/safety → Task 3. ✓
- §6.1 declarative rules (required/length/not_equal/match/deny, glob, skip_drafts) → Task 8. ✓
- §6.2 details-blank-line w/ safe fix → Task 7. ✓
- §7 CLI (lint/--fix/--unsafe-fixes/--diff, fmt/--check/--diff, explain, list, global flags) → Task 14. ✓
- §8 fmt idempotent + fence-aware → Task 11. ✓
- §9 config + inline suppression + warn-on-unused → Tasks 9, 10. ✓
- §10 human+JSON output, exit codes, advisory warnings, --max-warnings → Tasks 12, 14. ✓
- §11 GoReleaser + go install → Tasks 15, 17. ✓
- §12 CHANGELOG/conventional commits → Tasks 1, 15, 16. ✓
- §13 golden + idempotence + engine tests → Tasks 7, 8, 11, 13. ✓
- §15 repo conventions (cmd/+pkg/, devbox, taskfile, golangci, vendored) → Task 1. ✓

**Known MVP simplifications (intentional, documented):**
- Lazy Goldmark AST deferred (no MVP rule needs it) — Task 2 Step 5; Phase 2.
- `lint --diff` prints a change count, not a full per-file unified diff (full diff is available in `fmt --diff`); a follow-up threads before/after bytes through `engine.Fix`. — Task 14 Step 3 note.
- `lint` uses `os.Exit`, so the CLI test exercises `list`; a Phase-2 refactor returns an exit code for unit-testable lint. — Task 14 Step 9 note.

**Type consistency:** `Severity`, `FixSafety`, `TextEdit`, `Meta`, `Finding`, `Rule` (Task 3) are used unchanged in Tasks 7, 8, 10–14. `DeclSpec` fields (Task 8) match `config.CustomRule` fields mapped in `engine.New` (Task 13). `engine.ApplyEdits`/`UnifiedDiff`/`Format`/`MarkdownFiles`/`New`/`Run`/`Fix` signatures are consistent across Tasks 6, 11, 13, 14. ✓
