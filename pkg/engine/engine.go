package engine

import (
	"context"
	"errors"
	"fmt"
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

// fileMode is the permission bits used when writing fixed content back to disk.
const fileMode = 0o600

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
		if ruleEnabled(r.Meta().Name, cfg.Default, enabled, disabled) {
			wrapped, err := applySetting(r, cfg)
			if err != nil {
				return nil, err
			}
			e.rules = append(e.rules, wrapped)
		}
	}
	if err := e.addCustomRules(cfg.Custom); err != nil {
		return nil, err
	}
	return e, nil
}

func ruleEnabled(name, preset string, enabled, disabled map[string]bool) bool {
	on := preset != "none"
	if enabled[name] {
		on = true
	}
	if disabled[name] {
		on = false
	}
	return on
}

func (e *Engine) addCustomRules(customs []config.CustomRule) error {
	for i := range customs {
		c := &customs[i]
		sev := rule.Warning
		if c.Severity != "" {
			s, err := rule.ParseSeverity(c.Severity)
			if err != nil {
				return err
			}
			sev = s
		}
		dr, err := rule.NewDeclarativeRule(rule.DeclSpec{
			ID: c.ID, Type: c.Type, Glob: c.Glob, Field: c.Field, Fields: c.Fields,
			Min: c.Min, Max: c.Max, Pattern: c.Pattern, SkipDrafts: c.SkipDrafts, Severity: sev,
		})
		if err != nil {
			return err
		}
		e.rules = append(e.rules, dr)
	}
	return nil
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
		g.Go(func() error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			findings, err := e.lintFile(f.path, f.format)
			if err != nil {
				return err
			}
			mu.Lock()
			all = append(all, findings...)
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
	_ = ctx
	files, err := e.discover(paths)
	if err != nil {
		return nil, err
	}
	var changed []string
	for _, f := range files {
		out, modified, err := e.fixFile(f, unsafe)
		if err != nil {
			return nil, err
		}
		if !modified {
			continue
		}
		changed = append(changed, f.path)
		if !dryRun {
			if err := os.WriteFile(f.path, out, fileMode); err != nil { //nolint:gosec // content file
				return nil, err
			}
		}
	}
	return changed, nil
}

func (e *Engine) fixFile(f target, unsafe bool) (out []byte, modified bool, err error) {
	raw, err := os.ReadFile(f.path) //nolint:gosec // discovery walk
	if err != nil {
		return nil, false, err
	}
	doc, err := document.Parse(f.format, f.path, raw)
	if err != nil {
		return nil, false, err
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
		return nil, false, nil
	}
	out, err = ApplyEdits(raw, edits)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func (e *Engine) discover(paths []string) ([]target, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	var out []target
	seen := map[string]bool{}
	for _, root := range paths {
		targets, err := walkRoot(root, seen, e)
		if err != nil {
			return nil, err
		}
		out = append(out, targets...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
	return out, nil
}

func walkRoot(root string, seen map[string]bool, e *Engine) ([]target, error) {
	var out []target
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
	return out, nil
}

func (e *Engine) ignored(p string) bool {
	for _, g := range e.cfg.Ignore {
		if ok, _ := doublestar.Match(g, p); ok {
			return true
		}
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

func (s settingRule) Meta() rule.Meta {
	m := s.Rule.Meta()
	m.Severity = s.severity
	return m
}

func (s settingRule) Check(doc *document.Document, report func(rule.Finding)) {
	s.Rule.Check(doc, func(f rule.Finding) {
		f.Severity = s.severity
		report(f)
	})
}

func applySetting(r rule.Rule, cfg *config.Config) (rule.Rule, error) {
	set, ok := cfg.Settings[r.Meta().Name]
	if !ok || set.Severity == "" {
		return r, nil
	}
	sev, err := rule.ParseSeverity(set.Severity)
	if err != nil {
		return nil, fmt.Errorf("rule %s: invalid severity setting: %w", r.Meta().Name, err)
	}
	return settingRule{Rule: r, severity: sev}, nil
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
