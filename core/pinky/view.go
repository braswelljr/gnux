package pinky

import (
	"io"
	"os"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fsys"
	"github.com/braswelljr/gnux/core/internal/utmp"
	"github.com/braswelljr/gnux/core/internal/visual"
)

// gnux's terminal view (BR-13): the short format as a framed table with
// a header, the caller's own logins in the "yours" color, and -l as a
// panel for each user.

type pinkyView struct {
	theme *visual.Theme
	self  string
	table visual.Table
}

func (p *Pinky) getenv() func(string) string {
	if p.Getenv != nil {
		return p.Getenv
	}
	return os.Getenv
}

func (p *Pinky) prepareView() {
	p.view = nil
	if !p.Pretty {
		return
	}
	getenv := p.getenv()
	v := &pinkyView{theme: visual.Use(p.Color, p.Theme, getenv, p.diag.Warnf)}
	if p.Self != nil {
		v.self = p.Self()
	} else {
		v.self = utmp.Self(getenv)
	}
	head := []string{"Login"}
	if !p.NoFullName {
		head = append(head, "Name")
	}
	head = append(head, "TTY")
	if !p.NoIdle {
		head = append(head, "Idle")
	}
	head = append(head, "When")
	if !p.NoWhere {
		head = append(head, "Where")
	}
	v.table = visual.Table{Header: head, MaxWidth: core.TerminalWidth(p.streams.Out, getenv)}
	p.view = v
}

// addRow takes one login into the table. The TTY keeps GNU's mark of a
// terminal that refuses messages ('*') or cannot be examined ('?').
func (p *Pinky) addRow(r row) {
	v := p.view
	t := v.theme
	key := visual.Others
	if r.user == v.self {
		key = visual.You
	}
	cells := []string{t.Paint(key, r.user)}
	if !p.NoFullName {
		// The table fits the terminal, so the name needs no cut.
		name := r.fullName
		if r.noName {
			name = "???"
		}
		cells = append(cells, name)
	}
	cells = append(cells, strings.TrimLeft(string(r.mesg)+r.line, " "))
	if !p.NoIdle {
		cells = append(cells, strings.TrimSpace(r.idle))
	}
	cells = append(cells, t.Paint(visual.Date, r.when))
	if !p.NoWhere {
		cells = append(cells, t.Paint(visual.Punctuation, r.where))
	}
	v.table.Rows = append(v.table.Rows, cells)
}

func (p *Pinky) drawView() {
	if p.view == nil || len(p.view.table.Rows) == 0 {
		return
	}
	tb := dropEmpty(p.view.table, "Login", "TTY", "When")
	if p.NoHeading {
		tb.Header = nil
	}
	_, _ = p.out.WriteString(p.view.theme.Render(tb, visual.UTF8(p.getenv())))
}

// longPanel is -l under the view: what is known about one user, in a box
// titled with the login name.
func (p *Pinky) longPanel(name string) {
	getenv := p.getenv()
	t := visual.Use(p.Color, p.Theme, getenv, p.diag.Warnf)
	label := func(s string) string { return t.Paint(visual.Punctuation, s) }
	var lines []string
	a, err := p.LookupUser(name)
	if err != nil || a == nil {
		lines = append(lines, label("In real life  ")+"???")
	} else {
		lines = append(lines, label("In real life  ")+fullName(a))
		if !p.NoHomeAndShell {
			lines = append(lines, label("Directory     ")+t.Paint(visual.TypeDir, a.Home))
			if a.Shell != "" {
				lines = append(lines, label("Shell         ")+a.Shell)
			}
		}
		if !p.NoProject {
			lines = append(lines, fileLines(a.Home+"/.project", label("Project       "))...)
		}
		if !p.NoPlan {
			lines = append(lines, fileLines(a.Home+"/.plan", label("Plan          "))...)
		}
	}
	_, _ = p.out.WriteString(t.Box(name, lines, visual.UTF8(getenv)))
}

// fileLines is a user's ~/.project or ~/.plan as lines of a panel, the
// first after the label and the rest under it; nothing when it cannot be
// read.
func fileLines(path, label string) []string {
	f, err := fsys.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	data, _ := io.ReadAll(io.LimitReader(f, 64*1024))
	text := strings.TrimRight(strings.ReplaceAll(string(data), "\t", "    "), "\n")
	if text == "" {
		return nil
	}
	var lines []string
	indent := strings.Repeat(" ", visual.Width(label))
	for i, l := range strings.Split(text, "\n") {
		if i == 0 {
			lines = append(lines, label+l)
		} else {
			lines = append(lines, indent+l)
		}
	}
	return lines
}

// dropEmpty leaves out the columns empty in every row, but those named
// keep.
func dropEmpty(tb visual.Table, keep ...string) visual.Table {
	var cols []int
	for i, h := range tb.Header {
		used := false
		for _, k := range keep {
			used = used || h == k
		}
		for _, r := range tb.Rows {
			used = used || visual.Width(r[i]) > 0
		}
		if used {
			cols = append(cols, i)
		}
	}
	out := visual.Table{MaxWidth: tb.MaxWidth}
	pick := func(row []string) []string {
		picked := make([]string, 0, len(cols))
		for _, i := range cols {
			picked = append(picked, row[i])
		}
		return picked
	}
	out.Header = pick(tb.Header)
	for _, r := range tb.Rows {
		out.Rows = append(out.Rows, pick(r))
	}
	return out
}
