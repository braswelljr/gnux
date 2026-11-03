package shell

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core/fsys"
	"github.com/braswelljr/gnux/core/internal/visual"
)

// gnux's terminal view of the shell (BR-13): with Pretty, an interactive
// session's prompt names the working directory and the last command's
// status when it failed, and with Color the help builtin's syntax column
// is colored. A script, piped or redirected, sees none of it.

// theme is the view's colors, loaded once; nil when uncolored.
func (sh *Shell) theme() *visual.Theme {
	if !sh.Pretty || !sh.Color {
		return nil
	}
	if sh.colors == nil {
		sh.colors = visual.Use(true, sh.Theme, sh.getenv(), func(format string, a ...any) {
			sh.errorf(sh.streams, format, a...)
		})
	}
	return sh.colors
}

func (sh *Shell) getenv() func(string) string {
	if sh.Getenv != nil {
		return sh.Getenv
	}
	return os.Getenv
}

// prompt is what an interactive session writes before a line; continuing
// says the line continues an unfinished command.
func (sh *Shell) prompt(continuing bool) string {
	if !sh.Pretty {
		if continuing {
			return ContinuePrompt
		}
		return Prompt
	}
	t := sh.theme()
	arrow := "❯"
	if !visual.UTF8(sh.getenv()) {
		arrow = ">"
	}
	if continuing {
		return t.Paint(visual.Punctuation, strings.Repeat(" ", len("gnux "))+arrow) + " "
	}
	var b strings.Builder
	b.WriteString(t.Paint(visual.Punctuation, "gnux "))
	b.WriteString(t.Paint(visual.Heading, sh.where()))
	b.WriteByte(' ')
	key := visual.Good
	if sh.status != 0 {
		key = visual.Bad
		b.WriteString(t.Paint(key, "["+strconv.Itoa(sh.status)+"]"))
		b.WriteByte(' ')
	}
	b.WriteString(t.Paint(key, arrow))
	b.WriteByte(' ')
	return b.String()
}

// where is the working directory as the prompt shows it: under the home
// directory as ~/..., else whole.
func (sh *Shell) where() string {
	wd, err := fsys.Getwd()
	if err != nil {
		return "?"
	}
	if sh.HomeDir != nil {
		if home, err := sh.HomeDir(); err == nil && home != "" {
			// Either may be reached through a link (/var is /private/var
			// on macOS), so they are compared with their links resolved.
			wd, home = resolved(wd), resolved(home)
			if wd == home {
				return "~"
			}
			if rel, ok := strings.CutPrefix(wd, home+string(filepath.Separator)); ok {
				return "~" + string(filepath.Separator) + rel
			}
		}
	}
	return wd
}

// helpColumn is where the help text's descriptions start.
const helpColumn = 28

// resolved is path with its links resolved, or path when that fails.
func resolved(path string) string {
	if real, err := fsys.EvalSymlinks(path); err == nil {
		return real
	}
	return path
}

// paintHelp colors the syntax column of the help text: what is written
// before the description on each line.
func paintHelp(t *visual.Theme, text string) string {
	if t == nil {
		return text
	}
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		switch {
		case i == 0:
			lines[i] = t.Paint(visual.Heading, l)
		case strings.HasPrefix(l, "  "):
			// The descriptions start in one column; a line without one is
			// all syntax.
			if len(l) > helpColumn && l[helpColumn-2:helpColumn] == "  " && l[helpColumn] != ' ' {
				syntax := strings.TrimRight(l[2:helpColumn], " ")
				lines[i] = "  " + t.Paint(visual.Size, syntax) + l[2+len(syntax):]
			} else {
				lines[i] = "  " + t.Paint(visual.Size, l[2:])
			}
		}
	}
	return strings.Join(lines, "\n")
}
