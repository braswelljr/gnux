package shred

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/human"
	"github.com/braswelljr/gnux/core/internal/visual"
)

// passView is -v under gnux's terminal view (BR-13): each pass is one
// line, rewritten in place as the pass goes.
type passView struct {
	line    *visual.Line
	theme   *visual.Theme
	unicode bool
}

// viewInterval is how often the view redraws a pass.
const viewInterval = 200 * time.Millisecond

// passBar is the width of a pass's bar.
const passBar = 20

// prepareView sets up the view for a Run. Every diagnostic then goes
// through a writer that first ends the line being drawn, so an error or
// the next report starts on a line of its own.
func (s *Shred) prepareView() {
	s.view = nil
	if !s.Verbose || !s.Pretty {
		return
	}
	getenv := s.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	v := &passView{unicode: visual.UTF8(getenv)}
	v.theme = visual.Use(s.Color, s.Theme, getenv, s.diag.Warnf)
	v.line = &visual.Line{W: s.streams.Err, Max: core.TerminalWidth(s.streams.Err, getenv), Unicode: v.unicode}
	s.view = v
	streams := s.streams
	streams.Err = lineEnder{v.line, s.streams.Err}
	s.diag = core.NewDiag(Name, streams)
}

// lineEnder ends the line being drawn before anything else is written.
type lineEnder struct {
	line *visual.Line
	w    io.Writer
}

func (e lineEnder) Write(p []byte) (int, error) {
	e.line.Done()
	return e.w.Write(p)
}

// show draws pass k of n over the line: GNU's words, then a bar, the
// percentage and the amount written of the size, or the amount alone when
// the size is not known.
func (v *passView) show(q string, k, n int, name string, offset, size int64) {
	t := v.theme
	opts := human.Autoscale | human.SI | human.Base1024 | human.B
	off := t.Paint(visual.Size, human.Readable(uint64(offset), human.Floor|opts, 1, 1))
	head := t.Paint(visual.Punctuation, Name+": ") + t.Paint(visual.Path, q) +
		fmt.Sprintf(": pass %d/%d (%s) ", k, n, name)
	if size < 0 {
		v.line.Show(head + off)
		return
	}
	fraction := 1.0
	if size > 0 {
		fraction = float64(offset) / float64(size)
	}
	total := t.Paint(visual.Size, human.Readable(uint64(size), human.Ceiling|opts, 1, 1))
	if offset == size {
		// Done: the amount is the size, as GNU writes it then.
		off = total
	}
	v.line.Show(head + t.Progress(fraction, passBar, v.unicode) + fmt.Sprintf(" %3d%%  ", int(fraction*100)) +
		off + t.Paint(visual.Punctuation, "/") + total)
}
