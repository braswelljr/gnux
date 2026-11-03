package dd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/visual"
)

// progress is status=progress: GNU's line of statistics rewritten once a
// second, or under the view a bar rewritten five times a second.
type progress struct {
	start, next time.Time
	total       int64 // input bytes the copy will read; -1 when unknown
	gnuLen      int   // the length of GNU's last line, to pad over
	line        *visual.Line
	theme       *visual.Theme
	unicode     bool
}

// The intervals between two progress lines.
const (
	gnuInterval    = time.Second
	prettyInterval = 200 * time.Millisecond
)

func (d *Dd) newProgress(start time.Time, total int64) *progress {
	p := &progress{start: start, next: start.Add(gnuInterval), total: total}
	if d.Pretty {
		getenv := d.Getenv
		if getenv == nil {
			getenv = os.Getenv
		}
		p.unicode = visual.UTF8(getenv)
		p.theme = visual.Use(d.Color, d.Theme, getenv, d.diag.Warnf)
		p.line = &visual.Line{W: d.streams.Err, Max: core.TerminalWidth(d.streams.Err, getenv), Unicode: p.unicode}
		p.next = start.Add(prettyInterval)
	}
	return p
}

// total is how many input bytes the copy will read, when that is known:
// what is left of a regular file, or count=, whichever is less.
func (d *Dd) total(in io.Reader, ibs int64) int64 {
	total := int64(-1)
	if d.HasCount {
		total = d.Count
		if !d.CountBytes {
			total *= ibs
		}
	}
	if f, ok := in.(*os.File); ok {
		if info, err := f.Stat(); err == nil && info.Mode().IsRegular() {
			if pos, err := f.Seek(0, io.SeekCurrent); err == nil {
				left := max(info.Size()-pos, 0)
				if total < 0 || left < total {
					total = left
				}
			}
		}
	}
	return total
}

// tick draws the progress line when its time has come. GNU checks before
// each block it reads, and so does ddx.
func (c *copier) tick() {
	p := c.progress
	if p == nil {
		return
	}
	now := c.d.Now()
	if now.Before(p.next) {
		return
	}
	if p.line != nil {
		p.next = now.Add(prettyInterval)
		c.drawPretty(now)
		return
	}
	// GNU moves its next time on by one second, not from now.
	p.next = p.next.Add(gnuInterval)
	c.drawGNU(now)
}

func (c *copier) drawGNU(now time.Time) {
	p := c.progress
	s := xfer(c.st.bytes, now.Sub(p.start), true)
	pad := ""
	if len(s) < p.gnuLen {
		pad = spaces(p.gnuLen - len(s))
	}
	_, _ = io.WriteString(c.d.streams.Err, "\r"+s+pad)
	p.gnuLen = len(s)
}

func spaces(n int) string { return strings.Repeat(" ", n) }

// barWidth is the width of the view's bar.
const barWidth = 24

// drawPretty draws the view's line: with a known size, a bar, the
// percentage, what was copied, the rate and the time left; without one,
// what was copied, the rate and the time so far.
func (c *copier) drawPretty(now time.Time) {
	p, t := c.progress, c.progress.theme
	elapsed := now.Sub(p.start)
	copied := t.Paint(visual.Size, human(float64(c.st.bytes), 1000, siUnits))
	speed := t.Paint(visual.Size, rate(c.st.bytes, elapsed)+"/s")
	sep := t.Paint(visual.Punctuation, "  ")
	if p.total < 0 {
		p.line.Show(copied + t.Paint(visual.Punctuation, " copied") + sep + speed + sep + t.Paint(visual.Date, visual.Clock(elapsed)))
		return
	}
	fraction := 1.0
	if p.total > 0 {
		fraction = min(float64(c.inBytes)/float64(p.total), 1)
	}
	percent := fmt.Sprintf("%3d%%", int(fraction*100))
	left := "-:--"
	switch {
	case fraction >= 1:
		left = "0:00"
	case fraction > 0:
		left = visual.Clock(time.Duration(float64(elapsed) * (1 - fraction) / fraction))
	}
	of := t.Paint(visual.Punctuation, " of ")
	p.line.Show(t.Progress(fraction, barWidth, p.unicode) + " " + percent + sep + copied + of +
		t.Paint(visual.Size, human(float64(p.total), 1000, siUnits)) + sep + speed + sep +
		t.Paint(visual.Date, left) + t.Paint(visual.Punctuation, " left"))
}

// endProgress finishes the progress line before the final statistics:
// the view draws its last state; both end the line, as GNU does, when one
// was drawn.
func (c *copier) endProgress() {
	p := c.progress
	if p == nil {
		return
	}
	if p.line != nil {
		if p.line.Shown() {
			c.drawPretty(c.d.Now())
		}
		p.line.Done()
		return
	}
	if p.gnuLen > 0 {
		_, _ = io.WriteString(c.d.streams.Err, "\n")
	}
}
