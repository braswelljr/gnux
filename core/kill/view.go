package kill

import (
	"os"
	"strconv"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/visual"
)

// signalKey is the color of a signal's name in the view's table: the two
// no program can catch or ignore in the alert color, those whose default
// is to end the process in the "bad" color, those that stop it in the
// warning color, the rest plain.
func signalKey(name string) string {
	switch name {
	case "KILL", "STOP":
		return visual.BarAlert
	case "TSTP", "TTIN", "TTOU":
		return visual.BarWarn
	case "CHLD", "CONT", "URG", "WINCH":
		return ""
	}
	return visual.Bad
}

// viewTable is gnux's terminal view (BR-13) of -l and -L without operands:
// every signal as a row of a framed table, number, name and description.
func (k *Kill) viewTable() error {
	getenv := k.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	t := visual.Use(k.Color, k.Theme, getenv, k.diag.Warnf)
	tb := visual.Table{
		Header:   []string{"No.", "Name", "Description"},
		Right:    []bool{true, false, false},
		MaxWidth: core.TerminalWidth(k.streams.Out, getenv),
	}
	for _, s := range signals {
		tb.Rows = append(tb.Rows, []string{
			strconv.Itoa(s.Number),
			t.Paint(signalKey(s.Name), s.Name),
			s.Description,
		})
	}
	if _, err := k.streams.Out.Write([]byte(t.Render(tb, visual.UTF8(getenv)))); err != nil {
		k.diag.Errorf("write error: %s", core.ErrText(err))
	}
	return k.diag.Result()
}
