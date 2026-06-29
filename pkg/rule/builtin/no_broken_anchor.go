package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/openserbia/doclint/pkg/document"
	"github.com/openserbia/doclint/pkg/rule"
)

// anchorLinkRe matches an in-page markdown link whose destination is a pure
// fragment: [text](#fragment), optionally with a "title". Cross-page links with a
// path are handled by crossPageAnchorLinkRe instead; external URLs and shortcode
// links never start with "(#", so they are not matched here.
var anchorLinkRe = regexp.MustCompile(`\]\(#([^)\s"]+)(?:\s+"[^"]*")?\)`)

// crossPageAnchorLinkRe matches a markdown link to a site-absolute internal page
// plus a fragment: [text](/section/page/#fragment), optionally with a "title".
// Group 1 is the page path (starts with '/', no '#'), group 2 the fragment. A raw
// link like this is NOT a Hugo {{< relref >}}, so Hugo neither resolves nor
// build-fails it: a wrong path or anchor renders fine and the jump silently does
// nothing — the same failure mode the in-page check guards against, one page over.
var crossPageAnchorLinkRe = regexp.MustCompile(`\]\((/[^)\s#"]*)#([^)\s"]+)(?:\s+"[^"]*")?\)`)

// relrefLinkRe matches a Hugo {{< relref "…" >}} (or {{% relref %}}) reference
// that carries a fragment, in either observed shape: the anchor inside the
// argument — {{< relref "/page#frag" >}} — or appended right after the close —
// {{< relref "/page" >}}#frag. Group 1 is the quoted relref argument (path,
// optionally with its own #fragment); group 2 is an appended #fragment. It is
// matched context-free, so it is found in a markdown link [t]({{< relref … >}}),
// an HTML attribute href="{{< relref … >}}", or bare text alike. Hugo resolves and
// build-fails a bad relref *path*, but not the fragment, so a wrong anchor renders
// fine and the jump silently does nothing — the same failure mode this rule
// catches for raw links.
var relrefLinkRe = regexp.MustCompile(
	`\{\{[<%]\s*relref\s+["'` + "`" + `]([^"'` + "`" + `]+)["'` + "`" + `][^}]*\}\}(#[^)\s"]+)?`,
)

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
			"the heading whose auto-generated id matches that fragment. A raw " +
			"#fragment link renders fine even when no such heading exists: the jump " +
			"silently does nothing. This rule computes every heading's id the way " +
			"Hugo's default (GitHub-style) anchorize does — lowercase, drop " +
			"punctuation, spaces to hyphens, Unicode-aware so Cyrillic headings work " +
			"— honoring an explicit '{#custom-id}' and the -1/-2 suffixes Hugo adds to " +
			"duplicate headings, then flags any in-page link whose fragment is not " +
			"among them. It also resolves cross-page links to the target markdown file " +
			"under the nearest 'content' (or content/<lang>) directory — page.md, " +
			"page/index.md or page/_index.md — and flags a fragment matching no heading " +
			"there. Both raw site-absolute links — [text](/section/page/#frag) — and Hugo " +
			"{{< relref \"/section/page\" >}} shortcodes (with the #frag inside the " +
			"argument or appended after the close) are checked: a relref build-fails on a " +
			"bad page path but NOT on a bad fragment, so the anchor still fails silently. " +
			"Anything it cannot resolve to a local file — a missing page, an external " +
			"URL, a relative or named-parameter relref, or a page whose URL is remapped " +
			"via slug/url front matter — is left unchecked, so the rule never guesses. No " +
			"autofix: the intended target cannot be guessed. Links inside fenced code " +
			"blocks are not checked.",
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
	// roots are the directories a site-absolute URL could be served from (a
	// per-language content/<lang> dir and the bare content dir); cross-page links
	// are only resolvable when at least one is known. pages memoizes each target
	// page's anchors for the duration of this Check so a page linked twice is read once.
	roots, hasRoot := contentRoots(doc.Path)
	pages := pageAnchorCache{}

	emit := func(ln document.Line, hashByte int, msg string) {
		report(rule.Finding{
			Rule:     "no-broken-anchor",
			Path:     doc.Path,
			Line:     ln.Num,
			Col:      utf8.RuneCountInString(ln.Text[:hashByte]) + 1,
			Message:  msg,
			Severity: rule.Warning,
			Safety:   rule.NoFix,
		})
	}

	for _, ln := range doc.Lines {
		if ln.InFence || ln.Start < doc.BodyOffset {
			continue
		}
		for _, m := range anchorLinkRe.FindAllStringSubmatchIndex(ln.Text, -1) {
			frag := ln.Text[m[2]:m[3]]
			if anchors[frag] {
				continue
			}
			emit(ln, m[2]-1, fmt.Sprintf("link target #%s matches no heading in this page", frag))
		}
		if !hasRoot {
			continue
		}
		for _, m := range crossPageAnchorLinkRe.FindAllStringSubmatchIndex(ln.Text, -1) {
			urlPath, frag := ln.Text[m[2]:m[3]], ln.Text[m[4]:m[5]]
			target, found := pages.anchors(roots, urlPath)
			if !found || target[frag] {
				continue // unresolved target (skip) or a real heading there
			}
			emit(ln, m[4]-1, fmt.Sprintf("link target #%s matches no heading on page %s", frag, urlPath))
		}
		for _, m := range relrefLinkRe.FindAllStringSubmatchIndex(ln.Text, -1) {
			path, frag, hashByte := relrefPathFragment(ln.Text, m)
			if frag == "" || !strings.HasPrefix(path, "/") {
				continue // no fragment to check, or a relative path we cannot resolve
			}
			target, found := pages.anchors(roots, path)
			if !found || target[frag] {
				continue
			}
			emit(ln, hashByte, fmt.Sprintf("link target #%s matches no heading on page %s", frag, path))
		}
	}
}

// relrefPathFragment splits a relrefLinkRe match into the target page path, the
// fragment, and the byte offset of the fragment's '#'. The fragment is taken from
// inside the relref argument when present ([t]({{< relref "/page#frag" >}})),
// otherwise from the #frag appended after the close ([t]({{< relref "/page" >}}#frag)).
// path/frag are empty when there is nothing to check.
func relrefPathFragment(text string, m []int) (path, frag string, hashByte int) {
	arg := text[m[2]:m[3]]
	if i := strings.IndexByte(arg, '#'); i >= 0 {
		return arg[:i], arg[i+1:], m[2] + i
	}
	if m[4] >= 0 { // appended #fragment captured in group 2 (leading '#' at m[4])
		return arg, text[m[4]+1 : m[5]], m[4]
	}
	return arg, "", 0
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

// pageAnchorCache memoizes a target page's anchor set by URL path within a single
// Check. A nil entry records "could not resolve to a local page" so a repeated
// (broken) link to a missing page is not re-resolved.
type pageAnchorCache map[string]map[string]bool

// anchors returns the anchor set of the page served at urlPath (resolved under
// the first of roots that backs it with a file), and whether such a file was
// found. A not-found result makes the cross-page check skip the link rather than
// report it, so the rule never flags a target it cannot see.
func (c pageAnchorCache) anchors(roots []string, urlPath string) (map[string]bool, bool) {
	if a, ok := c[urlPath]; ok {
		return a, a != nil
	}
	file, ok := resolveContentFile(roots, urlPath)
	if !ok {
		c[urlPath] = nil
		return nil, false
	}
	raw, err := os.ReadFile(file) //nolint:gosec // file resolved under the content root
	if err != nil {
		c[urlPath] = nil
		return nil, false
	}
	doc, err := document.ParseMarkdown(file, raw)
	if err != nil {
		c[urlPath] = nil
		return nil, false
	}
	a := collectHeadingAnchors(doc)
	c[urlPath] = a
	return a, true
}

// contentRoots returns the directories a site-absolute URL could be served from,
// nearest-first. The first candidate is the "content" ancestor extended by the
// path segment that follows it in docPath: Hugo's per-language content directory
// is typically content/<lang>, and a link from a page under content/ru resolves
// against content/ru. The second is the bare "content" ancestor, for the flat
// single-language layout. Resolution tries each in order and takes the first that
// backs the URL with a real file, so both layouts work without reading Hugo's
// config. Returns false when docPath is not under a content tree.
func contentRoots(docPath string) ([]string, bool) {
	if docPath == "" {
		return nil, false
	}
	parts := strings.Split(filepath.Clean(docPath), string(filepath.Separator))
	idx := -1
	for i, p := range parts {
		if p == "content" {
			idx = i // nearest to the file: keep the last match
		}
	}
	if idx < 0 {
		return nil, false
	}
	base := strings.Join(parts[:idx+1], string(filepath.Separator))
	if base == "" { // absolute path whose first segment was "content"
		base = string(filepath.Separator)
	}
	// A directory segment follows "content" when there are at least two more parts
	// (the segment plus the filename). Prefer that content/<seg> root, then bare content.
	if idx+2 < len(parts) {
		return []string{filepath.Join(base, parts[idx+1]), base}, true
	}
	return []string{base}, true
}

// resolveContentFile maps a site-absolute URL path to the markdown file Hugo would
// serve it from, trying each root in order and, within a root, the leaf page then
// the leaf/branch bundle index. It only returns a path inside its root that exists
// and is a regular file.
func resolveContentFile(roots []string, urlPath string) (string, bool) {
	clean := strings.Trim(urlPath, "/")
	for _, root := range roots {
		base := filepath.Join(root, filepath.FromSlash(clean))
		candidates := []string{base + ".md", filepath.Join(base, "index.md"), filepath.Join(base, "_index.md")}
		if clean == "" { // the site root URL "/"
			candidates = []string{filepath.Join(root, "_index.md"), filepath.Join(root, "index.md")}
		}
		for _, c := range candidates {
			if !within(root, c) {
				continue
			}
			if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
				return c, true
			}
		}
	}
	return "", false
}

// within reports whether p resolves to a location inside root, guarding against a
// URL path that climbs out with "..".
func within(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
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
