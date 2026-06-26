package report

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/openserbia/doclint/pkg/rule"
)

// Severity glyphs and the layout budget for the message column.
const (
	glyphError   = "✖"
	glyphWarning = "⚠"
	glyphInfo    = "ℹ"
	glyphOK      = "✓"

	msgMaxRunes = 70
	ellipsis    = "…"
)

// Human renders findings grouped by file in an eslint-"stylish" layout: a bold
// file-path header (the clickable anchor) followed by indented, column-aligned
// finding rows, then a one-line problem summary. Styling is applied with
// lipgloss; NoColor (or the NO_COLOR env var) yields plain, ANSI-free text.
type Human struct{ NoColor bool }

// humanStyles bundles the lipgloss styles for one render pass. When color is
// off every field is an attribute-less style, so Render is an identity — this
// also strips bold/underline/faint, which the termenv Ascii profile alone does
// not (it only degrades color).
type humanStyles struct {
	header lipgloss.Style
	errG   lipgloss.Style
	warnG  lipgloss.Style
	infoG  lipgloss.Style
	okG    lipgloss.Style
	loc    lipgloss.Style
	msg    lipgloss.Style
	ruleN  lipgloss.Style
}

func (h Human) styles(w io.Writer) humanStyles {
	r := lipgloss.NewRenderer(w)
	noColor := h.NoColor || os.Getenv("NO_COLOR") != ""
	if noColor {
		r.SetColorProfile(termenv.Ascii)
	}
	plain := r.NewStyle()
	st := humanStyles{
		header: plain, errG: plain, warnG: plain, infoG: plain,
		okG: plain, loc: plain, msg: plain, ruleN: plain,
	}
	if noColor {
		return st
	}
	st.header = r.NewStyle().Bold(true).Underline(true)
	st.errG = r.NewStyle().Foreground(lipgloss.Color("9"))
	st.warnG = r.NewStyle().Foreground(lipgloss.Color("11"))
	st.infoG = r.NewStyle().Foreground(lipgloss.Color("12"))
	st.okG = r.NewStyle().Foreground(lipgloss.Color("10"))
	st.loc = r.NewStyle().Faint(true)
	st.ruleN = r.NewStyle().Faint(true)
	return st
}

func (st humanStyles) glyph(s rule.Severity) string {
	switch s {
	case rule.Error:
		return st.errG.Render(glyphError)
	case rule.Warning:
		return st.warnG.Render(glyphWarning)
	default:
		return st.infoG.Render(glyphInfo)
	}
}

// fixMark is the one-cell autofix indicator: a green "*" for a safe fix, a yellow
// "~" for an unsafe-only fix, a space when the finding has no fix. Fixed width so
// the rows stay aligned.
func (st humanStyles) fixMark(s rule.FixSafety) string {
	switch s {
	case rule.Safe:
		return st.okG.Render("*")
	case rule.Unsafe:
		return st.warnG.Render("~")
	default:
		return " "
	}
}

// lineWriter streams strings to an io.Writer, latching the first error so call
// sites stay terse (every Write to the underlying writer is still checked).
type lineWriter struct {
	w   io.Writer
	err error
}

func (lw *lineWriter) write(s string) {
	if lw.err != nil {
		return
	}
	_, lw.err = io.WriteString(lw.w, s)
}

// Report writes findings grouped by file, then a summary footer. Findings are
// expected pre-sorted by path,line,col (the engine guarantees this).
func (h Human) Report(w io.Writer, findings []rule.Finding) error {
	st := h.styles(w)
	lw := &lineWriter{w: w}

	if len(findings) == 0 {
		lw.write(" " + st.okG.Render(glyphOK) + " no problems\n")
		return lw.err
	}

	files := 0
	for i := 0; i < len(findings); {
		j := i
		for j < len(findings) && findings[j].Path == findings[i].Path {
			j++
		}
		if files > 0 {
			lw.write("\n")
		}
		renderGroup(lw, st, findings[i:j])
		files++
		i = j
	}
	lw.write("\n")
	renderFooter(lw, st, findings, files)
	return lw.err
}

// renderGroup prints one file's header and its column-aligned finding rows.
// Within the group, higher severities come first; line order is preserved
// within a severity (a stable sort over already line-sorted input).
func renderGroup(lw *lineWriter, st humanStyles, group []rule.Finding) {
	lw.write(" " + st.header.Render(group[0].Path) + "\n")

	ordered := make([]rule.Finding, len(group))
	copy(ordered, group)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Severity > ordered[j].Severity
	})

	locs := make([]string, len(ordered))
	msgs := make([]string, len(ordered))
	locW, msgW := 0, 0
	for k, f := range ordered {
		// The line:col token is kept literal and contiguous for IDE linkifying.
		locs[k] = fmt.Sprintf("%d:%d", f.Line, f.Col)
		msgs[k] = truncateRunes(f.Message, msgMaxRunes)
		if n := len([]rune(locs[k])); n > locW {
			locW = n
		}
		if n := len([]rune(msgs[k])); n > msgW {
			msgW = n
		}
	}

	for k, f := range ordered {
		lw.write("  ")
		lw.write(st.glyph(f.Severity))
		lw.write(" ")
		lw.write(st.fixMark(f.Safety))
		lw.write(" ")
		lw.write(st.loc.Render(locs[k]))
		lw.write(strings.Repeat(" ", locW-len([]rune(locs[k]))))
		lw.write("  ")
		lw.write(st.msg.Render(msgs[k]))
		lw.write(strings.Repeat(" ", msgW-len([]rune(msgs[k]))))
		lw.write("  ")
		lw.write(st.ruleN.Render(f.Rule))
		lw.write("\n")
	}
}

// renderFooter prints the trailing summary line; zero-count severities are
// omitted and the leading glyph reflects the worst severity present.
func renderFooter(lw *lineWriter, st humanStyles, findings []rule.Finding, files int) {
	errs, warns, infos := counts(findings)
	total := len(findings)

	var glyph string
	switch {
	case errs > 0:
		glyph = st.errG.Render(glyphError)
	case warns > 0:
		glyph = st.warnG.Render(glyphWarning)
	default:
		glyph = st.infoG.Render(glyphInfo)
	}

	parts := []string{fmt.Sprintf("%d %s", total, plural(total, "problem"))}
	if errs > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", errs, plural(errs, "error")))
	}
	if warns > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", warns, plural(warns, "warning")))
	}
	if infos > 0 {
		parts = append(parts, fmt.Sprintf("%d info", infos))
	}
	parts = append(parts, fmt.Sprintf("across %d %s", files, plural(files, "file")))

	lw.write(" " + glyph + " " + strings.Join(parts, " · ") + "\n")

	safe, unsafe := fixCounts(findings)
	var fixParts []string
	if safe > 0 {
		fixParts = append(fixParts, st.okG.Render("*")+fmt.Sprintf(" %d fixable with --fix", safe))
	}
	if unsafe > 0 {
		fixParts = append(fixParts, st.warnG.Render("~")+fmt.Sprintf(" %d with --unsafe-fixes", unsafe))
	}
	if len(fixParts) > 0 {
		lw.write("   " + strings.Join(fixParts, "  ") + "\n")
	}

	docs := distinctDocs(findings)
	if len(docs) > 0 {
		lw.write("\n " + st.ruleN.Render("learn how to fix:") + "\n")
		w := 0
		for _, d := range docs {
			if len(d.rule) > w {
				w = len(d.rule)
			}
		}
		for _, d := range docs {
			lw.write("   " + d.rule + strings.Repeat(" ", w-len(d.rule)) + "  " + st.loc.Render(d.url) + "\n")
		}
	}
}

type docRef struct{ rule, url string }

// distinctDocs returns the rule→doc-URL pairs of the built-in rules that fired,
// one per rule, sorted by rule name — so the footer lists each problem type once.
func distinctDocs(findings []rule.Finding) []docRef {
	seen := map[string]string{}
	for _, f := range findings {
		if f.DocURL != "" {
			seen[f.Rule] = f.DocURL
		}
	}
	out := make([]docRef, 0, len(seen))
	for r, u := range seen {
		out = append(out, docRef{rule: r, url: u})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].rule < out[j].rule })
	return out
}

// truncateRunes shortens s to at most maxRunes runes (rune-aware so multibyte
// content stays valid), appending an ellipsis when it trims.
func truncateRunes(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	if maxRunes <= 1 {
		return string(r[:maxRunes])
	}
	return string(r[:maxRunes-1]) + ellipsis
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
