package builtin

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

// anchorLinkRe matches an in-page markdown link whose destination is a pure
// fragment: [text](#fragment), optionally with a "title". Cross-page links
// (page#frag), external URLs and shortcode links never start with "(#", so they
// are not matched — Hugo already build-fails a bad relref, whereas a raw in-page
// #link fails silently, which is what this rule catches.
var anchorLinkRe = regexp.MustCompile(`\]\(#([^)\s"]+)(?:\s+"[^"]*")?\)`)

// headingIDAttrRe captures an explicit heading id attribute at the end of a
// heading line: "## Heading {#custom-id}" (optionally with extra classes).
var headingIDAttrRe = regexp.MustCompile(`\{#([^\s}]+)[^}]*\}\s*$`)

// trailingAttrRe matches a trailing {…} attribute block, stripped from a
// heading's text before it is slugified.
var trailingAttrRe = regexp.MustCompile(`\s*\{[^}]*\}\s*$`)

// Shortcodes also generate anchors a markdown-only scan misses. These cover the
// two observed sources: an explicit anchor="…" parameter (e.g. the alert
// shortcode), and the per-step ids the steps shortcode emits — {{< step >}} →
// #step-N by ordinal, or an explicit id="…".
var (
	shortcodeAnchorRe = regexp.MustCompile(`\banchor=["']([^"']+)["']`)
	stepsOpenRe       = regexp.MustCompile(`\{\{[<%]\s*steps\b`)
	stepOpenRe        = regexp.MustCompile(`\{\{[<%]\s*step\b`)
	stepIDRe          = regexp.MustCompile(`\bid=["']([^"']+)["']`)
)

// NoBrokenAnchor flags an in-page anchor link [text](#fragment) whose fragment
// matches no heading anchor in the same document.
type NoBrokenAnchor struct{}

func (NoBrokenAnchor) Meta() rule.Meta {
	return rule.Meta{
		Name:        "no-broken-anchor",
		Title:       "Valid in-page anchor links",
		Description: "in-page anchor links must point at a heading in the same page",
		Detail: "A markdown link to a pure fragment — [text](#some-heading) — jumps to " +
			"the heading whose auto-generated id matches that fragment. Unlike a Hugo " +
			"{{< relref >}} (which fails the build when its target is missing), a raw " +
			"#fragment link renders fine even when no such heading exists: the jump " +
			"silently does nothing. This rule computes every heading's id the way " +
			"Hugo's default (GitHub-style) anchorize does — lowercase, drop " +
			"punctuation, spaces to hyphens, Unicode-aware so Cyrillic headings work " +
			"— honoring an explicit '{#custom-id}' and the -1/-2 suffixes Hugo adds to " +
			"duplicate headings, then flags any in-page link whose fragment is not " +
			"among them. No autofix: the intended target cannot be guessed. Cross-page " +
			"'page#frag' links and links inside fenced code blocks are not checked.",
		Severity: rule.Warning,
		Formats:  []document.Format{document.Markdown},
		Safety:   rule.NoFix,
		Example: rule.Example{
			Bad: `## Overview

See the [overview](#overvew).`,
			Good: `## Overview

See the [overview](#overview).`,
		},
	}
}

func (r NoBrokenAnchor) Check(doc *document.Document, report func(rule.Finding)) {
	anchors := collectHeadingAnchors(doc)
	for _, ln := range doc.Lines {
		if ln.InFence || ln.Start < doc.BodyOffset {
			continue
		}
		for _, m := range anchorLinkRe.FindAllStringSubmatchIndex(ln.Text, -1) {
			frag := ln.Text[m[2]:m[3]]
			if anchors[frag] {
				continue
			}
			hashByte := m[2] - 1 // the '#'
			report(rule.Finding{
				Rule:     "no-broken-anchor",
				Path:     doc.Path,
				Line:     ln.Num,
				Col:      utf8.RuneCountInString(ln.Text[:hashByte]) + 1,
				Message:  fmt.Sprintf("link target #%s matches no heading in this page", frag),
				Severity: rule.Warning,
				Safety:   rule.NoFix,
			})
		}
	}
}

// collectHeadingAnchors returns the set of anchor ids Hugo would generate for the
// document's headings, including explicit {#id}s and the duplicate -N suffixes.
func collectHeadingAnchors(doc *document.Document) map[string]bool {
	anchors := map[string]bool{}
	seen := map[string]int{}
	add := func(a string) {
		if a == "" {
			return
		}
		final := a
		if n := seen[a]; n > 0 {
			final = fmt.Sprintf("%s-%d", a, n)
		}
		seen[a]++
		anchors[final] = true
	}
	lines := doc.Lines
	for i, ln := range lines {
		if ln.InFence || ln.Start < doc.BodyOffset {
			continue
		}
		text, ok := headingText(doc, lines, i)
		if !ok {
			continue
		}
		if m := headingIDAttrRe.FindStringSubmatch(text); m != nil {
			add(m[1])
			continue
		}
		add(githubAnchor(trailingAttrRe.ReplaceAllString(text, "")))
	}
	collectShortcodeAnchors(lines, doc.BodyOffset, add)
	return anchors
}

// collectShortcodeAnchors adds anchors that Hugo shortcodes generate at render
// time and a markdown-only scan would otherwise miss: an explicit anchor="…"
// parameter (e.g. the alert shortcode), and the step-N / id="…" ids the steps
// shortcode emits for each {{< step >}} (the ordinal resets at each {{< steps >}}).
// Over-collecting is safe — it only makes the broken-anchor check more lenient,
// never produces a false positive.
func collectShortcodeAnchors(lines []document.Line, bodyOffset int, add func(string)) {
	ordinal := 0
	for _, ln := range lines {
		if ln.InFence || ln.Start < bodyOffset {
			continue
		}
		for _, m := range shortcodeAnchorRe.FindAllStringSubmatch(ln.Text, -1) {
			add(m[1])
		}
		if stepsOpenRe.MatchString(ln.Text) {
			ordinal = 0
		}
		for _, idx := range stepOpenRe.FindAllStringIndex(ln.Text, -1) {
			ordinal++
			if id := stepIDRe.FindStringSubmatch(ln.Text[idx[0]:]); id != nil {
				add(id[1])
			} else {
				add(fmt.Sprintf("step-%d", ordinal))
			}
		}
	}
}

// headingText returns the visible text of the ATX or setext heading at line i.
func headingText(doc *document.Document, lines []document.Line, i int) (string, bool) {
	ln := lines[i]
	if isATXHeading(ln.Text) {
		w := leadingWhitespace(ln.Text)
		s := strings.TrimLeft(ln.Text[w:], "#") // opening hashes
		s = strings.TrimSpace(s)
		s = strings.TrimRight(s, "#") // optional closing hashes
		return strings.TrimSpace(s), true
	}
	if i+1 < len(lines) && isSetextHeading(doc, lines, i) {
		return strings.TrimSpace(ln.Text), true
	}
	return "", false
}

// githubAnchor slugifies heading text the way Hugo's default (github) anchorize
// does: lowercase; keep Unicode letters, numbers and underscore; spaces and
// hyphens become hyphens; everything else (punctuation, symbols, emoji) dropped.
func githubAnchor(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsLetter(r), unicode.IsNumber(r), r == '_':
			b.WriteRune(r)
		case r == ' ', r == '-':
			b.WriteByte('-')
		}
	}
	return b.String()
}
