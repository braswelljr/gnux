package comm

import (
	"os"
	"path/filepath"

	"github.com/braswelljr/gnux/core/internal/visual"
)

// gnux's terminal view (BR-13): a header row naming each column's file, and
// every line in the color of the column it is in, so the three columns stay
// apart however long the lines are. The text, tabs included, is GNU's, and
// the output still streams.

// columnKeys are the colors of the three columns.
var columnKeys = [3]string{visual.FromFirst, visual.FromSecond, visual.FromBoth}

// tabStop is the width of a tab on a terminal.
const tabStop = 8

// prepareView sets the theme and writes the header row.
func (c *Comm) prepareView(names []string) {
	c.theme = nil
	if !c.Pretty || c.ZeroTerminated {
		return
	}
	getenv := c.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	c.theme = visual.Use(c.Color, c.Theme, getenv, c.diag.Warnf)
	// The columns are one tab apart, so with GNU's tab a title wider than
	// a tab stop would sit over the next column: it is cut to fit, its base
	// name first.
	unicode := visual.UTF8(getenv)
	title := func(name string) string {
		if name == "-" {
			name = "standard input"
		}
		if c.sep != "\t" {
			return name
		}
		if visual.Width(name) > tabStop-1 {
			name = filepath.Base(name)
		}
		return visual.Truncate(name, tabStop-1, unicode)
	}
	titles := [3]string{title(names[0]), title(names[1]), "both"}
	shown := [3]bool{!c.Suppress1, !c.Suppress2, !c.Suppress3}
	first := true
	for i, t := range titles {
		if !shown[i] {
			continue
		}
		if !first {
			_, _ = c.out.WriteString(c.sep)
		}
		first = false
		_, _ = c.out.WriteString(c.theme.Paint(visual.Header, t))
	}
	if !first {
		_ = c.out.WriteByte(c.eol)
	}
	c.pretty = true
}

// paintLine writes line in column col's color.
func (c *Comm) paintLine(col int, line []byte) {
	if code := c.theme.Code(columnKeys[col]); code != "" {
		_, _ = c.out.WriteString(visual.Paint(code, string(line)))
		_ = c.out.WriteByte(c.eol)
		return
	}
	c.writeLine(line)
}
