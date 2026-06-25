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

const (
	typeRequired = "required"
	typeLength   = "length"
	typeNotEqual = "not_equal"
	typeMatch    = "match"
	typeDeny     = "deny"

	twoFields = 2
)

type declRule struct {
	spec    DeclSpec
	pattern *regexp.Regexp
}

// NewDeclarativeRule compiles a DeclSpec into a Rule.
func NewDeclarativeRule(spec DeclSpec) (Rule, error) {
	r := &declRule{spec: spec}
	switch spec.Type {
	case typeRequired, typeLength, typeNotEqual:
	case typeMatch, typeDeny:
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
	switch r.spec.Type {
	case typeRequired:
		r.checkRequired(doc, report)
	case typeLength:
		r.checkLength(doc, report)
	case typeNotEqual:
		r.checkNotEqual(doc, report)
	case typeMatch:
		r.checkMatch(doc, report)
	case typeDeny:
		r.checkDeny(doc, report)
	}
}

func (r *declRule) emit(doc *document.Document, field, msg string, report func(Finding)) {
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

func (r *declRule) checkRequired(doc *document.Document, report func(Finding)) {
	if str(doc.Frontmatter[r.spec.Field]) == "" {
		r.emit(doc, r.spec.Field, fmt.Sprintf("%q is required and must be non-empty", r.spec.Field), report)
	}
}

func (r *declRule) checkLength(doc *document.Document, report func(Finding)) {
	v := str(doc.Frontmatter[r.spec.Field])
	if v == "" {
		return // absence is the `required` rule's job
	}
	n := len([]rune(v))
	if n < r.spec.Min {
		r.emit(doc, r.spec.Field, fmt.Sprintf("%q is %d chars, minimum %d", r.spec.Field, n, r.spec.Min), report)
	} else if r.spec.Max > 0 && n > r.spec.Max {
		r.emit(doc, r.spec.Field, fmt.Sprintf("%q is %d chars, maximum %d", r.spec.Field, n, r.spec.Max), report)
	}
}

func (r *declRule) checkNotEqual(doc *document.Document, report func(Finding)) {
	if len(r.spec.Fields) == twoFields {
		a, b := str(doc.Frontmatter[r.spec.Fields[0]]), str(doc.Frontmatter[r.spec.Fields[1]])
		if a != "" && a == b {
			r.emit(doc, r.spec.Fields[0], fmt.Sprintf("%q and %q must differ", r.spec.Fields[0], r.spec.Fields[1]), report)
		}
	}
}

func (r *declRule) checkMatch(doc *document.Document, report func(Finding)) {
	v := str(doc.Frontmatter[r.spec.Field])
	if v != "" && !r.pattern.MatchString(v) {
		r.emit(doc, r.spec.Field, fmt.Sprintf("%q must match /%s/", r.spec.Field, r.spec.Pattern), report)
	}
}

func (r *declRule) checkDeny(doc *document.Document, report func(Finding)) {
	v := str(doc.Frontmatter[r.spec.Field])
	if v != "" && r.pattern.MatchString(v) {
		r.emit(doc, r.spec.Field, fmt.Sprintf("%q must not match /%s/", r.spec.Field, r.spec.Pattern), report)
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
