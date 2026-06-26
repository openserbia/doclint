package cli

import (
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Status glyphs, shared with the lint reporter (pkg/report) so every command
// speaks the same visual language.
const (
	glyphOK   = "✓"
	glyphWarn = "⚠"
	glyphInfo = "ℹ"
)

// ui prints status output in doclint's common human style: a headline is
// " <glyph> <message>" and a detail line is indented "   <item>", with the
// headline summary placed last (matching the lint footer). Color is applied with
// lipgloss and disabled under --no-color, NO_COLOR, or a non-terminal writer.
// Write errors are latched so call sites stay terse; check Err() at the end.
type ui struct {
	w     io.Writer
	err   error
	okS   lipgloss.Style
	warnS lipgloss.Style
	infoS lipgloss.Style
}

func newUI(w io.Writer, noColor bool) *ui {
	r := lipgloss.NewRenderer(w)
	if noColor || os.Getenv("NO_COLOR") != "" {
		r.SetColorProfile(termenv.Ascii) // Ascii drops the foreground colors below
	}
	return &ui{
		w:     w,
		okS:   r.NewStyle().Foreground(lipgloss.Color("10")),
		warnS: r.NewStyle().Foreground(lipgloss.Color("11")),
		infoS: r.NewStyle().Foreground(lipgloss.Color("12")),
	}
}

func (u *ui) write(s string) {
	if u.err != nil {
		return
	}
	_, u.err = io.WriteString(u.w, s)
}

func (u *ui) headline(style lipgloss.Style, glyph, msg string) {
	u.write(" " + style.Render(glyph) + " " + msg + "\n")
}

func (u *ui) ok(msg string)   { u.headline(u.okS, glyphOK, msg) }
func (u *ui) warn(msg string) { u.headline(u.warnS, glyphWarn, msg) }
func (u *ui) info(msg string) { u.headline(u.infoS, glyphInfo, msg) }

// item prints an indented detail line (e.g. a changed file path), kept literal
// so editors still linkify it.
func (u *ui) item(text string) { u.write("   " + text + "\n") }

// Err returns the first write error, if any.
func (u *ui) Err() error { return u.err }

// plural appends "s" to word unless n == 1 (matches the lint reporter).
func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
