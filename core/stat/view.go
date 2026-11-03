package stat

import (
	"bytes"
	"os"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/visual"
)

// The pretty view's lines: a label, then a format expanded as the default
// format's directives are, so every value is the one GNU's layout shows.
var (
	prettyFile = []string{
		"Type\t%F",
		"Size\t%s bytes, %b blocks of %B, IO block %o",
		"Mode\t%04a %A",
		"Owner\t%U (%u)",
		"Group\t%G (%g)",
		"Links\t%h",
		"Inode\t%i on device %Hd,%Ld",
		"Accessed\t%x",
		"Modified\t%y",
		"Changed\t%z",
		"Born\t%w",
	}
	prettyDevice = "Device\t%Hr,%Lr"
	prettyLink   = "Points to\t%N"
	prettyFS     = []string{
		"Type\t%T",
		"ID\t%i",
		"Name length\t%l",
		"Block size\t%s, fundamental %S",
		"Blocks\t%b total, %f free, %a available",
		"Inodes\t%c total, %d free",
	}
)

// pretty reports whether the view applies: it replaces the default format
// only, never one the user wrote or -t.
func (st *Stat) pretty() bool {
	return st.Pretty && !st.FormatSet && !st.PrintfSet && !st.Terse
}

// expand writes format through the directive expander into a string.
func (st *Stat) expand(format string, each func(directive)) (string, error) {
	saved := st.out
	var b bytes.Buffer
	st.out = core.NewWriter(&b)
	err := st.print(format, false, "", each)
	_ = st.out.Flush()
	st.out = saved
	return b.String(), err
}

// panel writes one file's box: the name as its title, each label in the
// punctuation color and its value after it, the mode painted letter by
// letter, and for a file system a bar of the blocks in use.
func (st *Stat) panel(title string, titleKey string, lines []string, each func(directive)) error {
	theme := st.viewTheme()
	width := 0
	for _, l := range lines {
		label, _, _ := strings.Cut(l, "\t")
		width = max(width, len(label))
	}
	var rows []string
	for _, l := range lines {
		label, format, _ := strings.Cut(l, "\t")
		value, err := st.expand(format, each)
		if err != nil {
			return err
		}
		if label == "Mode" {
			octal, mode, _ := strings.Cut(value, " ")
			value = theme.Paint(visual.Octal, octal) + " " + visual.PaintMode(theme, mode)
		}
		rows = append(rows, theme.Paint(visual.Punctuation, label+strings.Repeat(" ", width-len(label)))+"  "+value)
		if label == "Blocks" && st.FileSystem {
			rows = append(rows, st.usage(theme, width, each))
		}
	}
	unicode := visual.UTF8(st.Getenv)
	if titleKey != visual.Header {
		title = theme.Paint(titleKey, title)
	}
	box := theme.Box(title, rows, unicode)
	_, err := st.out.WriteString(box)
	return err
}

// usage is a file system's line of blocks in use: a bar and a percentage.
func (st *Stat) usage(theme *visual.Theme, width int, each func(directive)) string {
	num := func(f string) float64 {
		s, _ := st.expand(f, each)
		n, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
		return n
	}
	total, avail, free := num("%b"), num("%a"), num("%f")
	used := total - free
	fraction := 0.0
	if used+avail > 0 {
		fraction = used / (used + avail)
	}
	key := visual.BarFull
	switch {
	case fraction >= visual.Alert:
		key = visual.BarAlert
	case fraction >= visual.Warn:
		key = visual.BarWarn
	}
	return theme.Paint(visual.Punctuation, "Used"+strings.Repeat(" ", width-len("Used"))) + "  " +
		theme.Bar(fraction, 20, visual.UTF8(st.Getenv)) + " " +
		theme.Paint(key, strconv.Itoa(int(fraction*100+0.5))+"%")
}

// viewTheme is the theme the panels are painted with, nil when uncolored.
func (st *Stat) viewTheme() *visual.Theme {
	if !st.themed {
		st.themed = true
		st.theme = visual.Use(st.Color, st.Theme, st.Getenv, st.diag.Warnf)
	}
	return st.theme
}

// filePanel is the panel of one file.
func (st *Stat) filePanel(f *file, each func(directive)) error {
	lines := prettyFile
	key := visual.Header
	switch m := f.info.Mode(); {
	case m&os.ModeDevice != 0:
		lines = append(append([]string{}, lines[:1]...), append([]string{prettyDevice}, lines[1:]...)...)
	case m&os.ModeSymlink != 0:
		lines = append(append([]string{}, lines[:1]...), append([]string{prettyLink}, lines[1:]...)...)
		key = visual.TypeLink
	case m.IsDir():
		key = visual.TypeDir
	}
	return st.panel(f.name, key, lines, each)
}
