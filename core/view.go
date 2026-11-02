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
