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
