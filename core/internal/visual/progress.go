package visual

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Line is one terminal line rewritten in place, for progress: each Show
// returns to the start of the line, writes s (cut to Max columns, so it
// never wraps and leaves a line \r cannot reach) and pads over what the
// last one left. Done ends the line.
type Line struct {
	W       io.Writer
	Max     int // the terminal's width; 0 is no limit
	Unicode bool
	last    int
}

// Show draws s over the line.
func (l *Line) Show(s string) {
	if l.Max > 1 && Width(s) > l.Max-1 {
		// One column short of the edge: a terminal that wraps at the
		// last column would otherwise move the cursor down.
		s = Truncate(s, l.Max-1, l.Unicode)
	}
	w := Width(s)
	pad := ""
	if w < l.last {
		pad = strings.Repeat(" ", l.last-w)
	}
	_, _ = io.WriteString(l.W, "\r"+s+pad)
	l.last = w
}

// Shown reports whether the line holds anything.
func (l *Line) Shown() bool { return l.last > 0 }

// Done ends the line, when anything was shown, so what follows starts on
// a line of its own.
func (l *Line) Done() {
	if l.last > 0 {
		_, _ = io.WriteString(l.W, "\n")
		l.last = 0
	}
}

// Clock renders a duration as a clock: m:ss, or h:mm:ss from an hour on
// ("0:06", "12:40", "1:02:03").
func Clock(d time.Duration) string {
	s := int64(d.Round(time.Second) / time.Second)
	s = max(s, 0)
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}
