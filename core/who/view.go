package who

import (
	"os"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/utmp"
	"github.com/braswelljr/gnux/core/internal/visual"
)

// gnux's terminal view (BR-13): the records as a framed table with a
// header, the columns that hold nothing left out, and the caller's own
// logins in the theme's "yours" color. -q is GNU's.

// whoView is the table being filled.
type whoView struct {
	theme *visual.Theme
	self  string
	rows  [][]string
	right []bool
	head  []string
}

func (w *Who) getenv() func(string) string {
	if w.Getenv != nil {
		return w.Getenv
	}
	return os.Getenv
}

func (w *Who) prepareView() {
	w.view = nil
	if !w.Pretty || w.Count {
		return
	}
	getenv := w.getenv()
	v := &whoView{theme: visual.Use(w.Color, w.Theme, getenv, w.diag.Warnf)}
	if w.Self != nil {
		v.self = w.Self()
	} else {
		v.self = utmp.Self(getenv)
	}
	v.head, v.right = w.columns("NAME", "", "LINE", "TIME", "IDLE", "PID", "COMMENT", "EXIT"),
		w.columnsRight()
	w.view = v
}

// columns picks, from every field of a line, those the options show, as
// line does.
func (w *Who) columns(user, state, line, when, idle, pid, comment, exit string) []string {
	c := []string{user}
	if w.includeMesg {
		c = append(c, state)
	}
	c = append(c, line, when)
	if w.includeIdle && !w.Short {
		c = append(c, idle)
	}
	if !w.Short {
		c = append(c, pid)
	}
	c = append(c, comment)
	if w.includeExit {
		c = append(c, exit)
	}
	return c
}

// columnsRight marks the PID column, which GNU right-aligns.
func (w *Who) columnsRight() []bool {
	c := w.columns("", "", "", "", "", "pid", "", "")
	right := make([]bool, len(c))
	for i, s := range c {
		right[i] = s == "pid"
	}
	return right
}

// addRow takes one line's fields into the table.
func (w *Who) addRow(user string, state byte, line, when, idle, pid, comment, exit string) {
	v := w.view
	t := v.theme
	key := visual.Others
	if user != "" && user == v.self {
		key = visual.You
	}
	s := string(state)
	if state == ' ' {
		s = ""
	}
	v.rows = append(v.rows, w.columns(t.Paint(key, user), s, line, t.Paint(visual.Date, when),
		strings.TrimSpace(idle), pid, t.Paint(visual.Punctuation, comment), exit))
}

// drawView writes the table, without the columns that are empty in every
// row (the name, line and time always stay).
func (w *Who) drawView() {
	v := w.view
	if v == nil || len(v.rows) == 0 {
		return
	}
	keep := make([]bool, len(v.head))
	for i, h := range v.head {
		keep[i] = h == "NAME" || h == "LINE" || h == "TIME"
		for _, r := range v.rows {
			keep[i] = keep[i] || visual.Width(r[i]) > 0
		}
	}
	pick := func(row []string) []string {
		var out []string
		for i, c := range row {
			if keep[i] {
				out = append(out, c)
			}
		}
		return out
	}
	tb := visual.Table{Header: pick(v.head), MaxWidth: core.TerminalWidth(w.streams.Out, w.getenv())}
	for i, r := range v.right {
		if keep[i] {
			tb.Right = append(tb.Right, r)
		}
	}
	for _, r := range v.rows {
		tb.Rows = append(tb.Rows, pick(r))
	}
	_, _ = w.out.WriteString(v.theme.Render(tb, visual.UTF8(w.getenv())))
}
