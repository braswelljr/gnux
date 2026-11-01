package tool

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
)

// When is the tri-state of --pretty: never, auto (on a terminal) or always.
type When int

// The values of When.
const (
	Never When = iota
	Auto
	Always
)

// ParseWhen reads a WHEN word as GNU's --color takes it.
func ParseWhen(option, arg string) (When, error) {
	switch arg {
	case "always", "yes", "force":
		return Always, nil
	case "auto", "tty", "if-tty":
		return Auto, nil
	case "never", "no", "none":
		return Never, nil
	}
	return Never, &core.UsageError{Msg: fmt.Sprintf("invalid argument '%s' for '%s'\n"+
		"Valid arguments are:\n  - 'always', 'yes', 'force'\n  - 'never', 'no', 'none'\n  - 'auto', 'tty', 'if-tty'", arg, option), Code: 1}
}

// On reports whether w is on for output that is, or is not, a terminal.
func (w When) On(terminal bool) bool {
	return w == Always || (w == Auto && terminal)
}

// RegisterView installs the two options every tool with gnux's terminal
// view takes: --pretty[=WHEN] and --theme=NAME. Neither is a GNU option,
// so both are names of gnux's own (OwnNames).
func RegisterView(fs *pflag.FlagSet) {
	fs.String("pretty", "", "the gnux terminal view `WHEN`: 'always' (the default when WHEN is omitted), "+
		"'auto' (the default: on a terminal only), or 'never'; GNUX_PRETTY sets the default")
	fs.Lookup("pretty").NoOptDefVal = "always"
	fs.String("theme", "", "color the view with the built-in theme `NAME`: "+strings.Join(core.ThemeNames(), ", ")+
		"; GNUX_THEME sets the default")
	OwnNames(fs, "pretty", "theme")
}

// View is how a tool shows its output, once the options and the
// environment have had their say.
type View struct {
	// Pretty is whether the gnux view is on at all.
	Pretty bool
	// Color is whether it is colored: off under NO_COLOR, and through a
	// pipe unless asked for with --pretty=always or GNUX_PRETTY=always.
	Color bool
	// Theme is the --theme name, "" for GNUX_THEME's.
	Theme string
}

// ResolvePretty settles --pretty: the option when given, else
// GNUX_PRETTY, else auto, and the When it was set to. An invalid
// GNUX_PRETTY is warned about on warn and taken as auto.
func ResolvePretty(value string, given bool, name string, getenv func(string) string, warn io.Writer) (When, error) {
	if given {
		return ParseWhen("--pretty", value)
	}
	env := getenv("GNUX_PRETTY")
	if env == "" {
		return Auto, nil
	}
	w, err := ParseWhen("GNUX_PRETTY", env)
	if err != nil {
		_, _ = fmt.Fprintf(warn, "%s: ignoring invalid value of environment variable GNUX_PRETTY: '%s'\n", name, env)
		return Auto, nil
	}
	return w, nil
}

// ViewOf resolves the view of a tool whose flags were registered with
// RegisterView, for output that is or is not a terminal.
func ViewOf(fs *pflag.FlagSet, terminal bool, name string, getenv func(string) string, warn io.Writer) (View, error) {
	f := fs.Lookup("pretty")
	when, err := ResolvePretty(f.Value.String(), f.Changed, name, getenv, warn)
	if err != nil {
		return View{}, err
	}
	theme, _ := fs.GetString("theme")
	if theme != "" && !validTheme(theme) {
		return View{}, &core.UsageError{Msg: fmt.Sprintf("invalid argument '%s' for '--theme'\nValid arguments are: %s",
			theme, strings.Join(core.ThemeNames(), ", ")), Code: 1}
	}
	v := View{Pretty: when.On(terminal), Theme: theme}
	v.Color = v.Pretty && getenv("NO_COLOR") == "" && (terminal || when == Always) && getenv("TERM") != "dumb"
	return v, nil
}

func validTheme(name string) bool {
	for _, t := range core.ThemeNames() {
		if strings.EqualFold(t, name) {
			return true
		}
	}
	return false
}
