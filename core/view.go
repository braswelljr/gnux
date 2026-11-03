package core

import (
	"bytes"
	"io"
	"sync"

	"github.com/braswelljr/gnux/core/internal/visual"
)

// ThemeNames lists the built-in themes of gnux's terminal view, which a
// tool's --theme option and the GNUX_THEME variable take.
func ThemeNames() []string {
	return visual.ThemeNames()
}

// ThemeSurface returns the background and foreground (#rrggbb) of the
// editor theme a built-in theme follows, its family's variant chosen from
// getenv (GNUX_BACKGROUND, COLORFGBG). The tools never paint a background;
// it is for a host that draws the whole terminal, such as the playground.
// ok is false for default, mono and an unknown name.
func ThemeSurface(name string, getenv func(string) string) (background, foreground string, ok bool) {
	return visual.Surface(name, getenv)
}

// ReportWriter returns w with each line written to it colored as gnux's
// terminal view shows a -v report (visual.Theme.Report), in the built-in
// theme named by theme ("" takes GNUX_THEME's). An unknown GNUX_THEME is
// reported through warn. It is safe for concurrent use, and passes whole
// lines on as they end; Flush writes a final unterminated one.
func ReportWriter(w io.Writer, theme string, getenv func(string) string, warn func(format string, a ...any)) *LineWriter {
	t := visual.Use(true, theme, getenv, warn)
	return &LineWriter{w: w, paint: t.Report}
}

// LineWriter passes each line through a function before writing it.
type LineWriter struct {
	mu      sync.Mutex
	w       io.Writer
	paint   func(string) string
	partial []byte
}

// Write paints and writes every complete line in p, holding the rest.
func (l *LineWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.partial = append(l.partial, p...)
	for {
		i := bytes.IndexByte(l.partial, '\n')
		if i < 0 {
			return len(p), nil
		}
		line := l.paint(string(l.partial[:i])) + "\n"
		l.partial = l.partial[i+1:]
		if _, err := io.WriteString(l.w, line); err != nil {
			return len(p), err
		}
	}
}

// Flush writes what is left of an unterminated last line.
func (l *LineWriter) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.partial) == 0 {
		return nil
	}
	_, err := io.WriteString(l.w, l.paint(string(l.partial)))
	l.partial = nil
	return err
}

// WriteThemes writes gnux's built-in themes as gnux themes shows them: each
// on a sample row in its colors when color is on, else the names alone.
func WriteThemes(w io.Writer, getenv func(string) string, color bool) error {
	return visual.Preview(w, getenv, color, visual.UTF8(getenv))
}

// ThemeShowcase is a few lines of output in the colors of the built-in
// theme name stands for (a family's variant from getenv): what a theme
// picker shows beside its list. unicode draws bars and arrows in Unicode.
func ThemeShowcase(name string, getenv func(string) string, unicode bool) []string {
	return visual.Showcase(name, getenv, unicode)
}

// ResolveTheme gives the theme a built-in name stands for: a family's dark
// or light variant (GNUX_BACKGROUND, COLORFGBG, the system's appearance),
// and any other name as it is, lower-cased.
func ResolveTheme(name string, getenv func(string) string) string {
	return visual.Resolve(name, getenv)
}

// UnicodeTerminal reports whether the locale in getenv (LC_ALL, LC_CTYPE,
// LANG) is UTF-8, so that a view may draw box lines, bars and arrows.
func UnicodeTerminal(getenv func(string) string) bool {
	return visual.UTF8(getenv)
}

// ThemeCode is the SGR code ("1;36") the built-in theme name stands for
// gives key, a two-letter theme key as GNUX_THEME's overrides write them
// ("pa", "ok", "xx"), with GNUX_THEME's overrides applied; "" when the key
// is off. An unknown name gives the default theme's code.
func ThemeCode(name, key string, getenv func(string) string) string {
	t, _ := visual.LoadTheme(name, getenv)
	return t.Code(key)
}
