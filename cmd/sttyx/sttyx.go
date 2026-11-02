// Package sttyx wires the sttyx command (GNU stty) to its engine.
package sttyx

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/stty"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the sttyx tool. Its settings look like options (-echo,
// -icanon), so sttyx reads its command line itself, as GNU stty does
// (parseArgs); Register declares -a, -g and -F for the help and the
// reference only.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:    Meta(),
		GNU:     "stty",
		Flags:   Register,
		Run:     run,
		RawArgs: true,
	}
}

// Register declares the sttyx options, for the documentation: parseArgs
// reads them.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("all", "a", false, "print all current settings in human-readable form")
	fs.BoolP("save", "g", false, "print all current settings in a stty-readable form")
	fs.StringP("file", "F", "", "open and use the specified `DEVICE` instead of stdin")
	tool.RegisterView(fs)
}

// fatal is a mistake GNU reports without the --help hint.
type fatal struct{ msg string }

func (e *fatal) Error() string { return e.msg }

// parseArgs splits a stty command line into its options and its settings,
// as GNU stty does: -a, -g and -F are options, as are --all, --save and
// --file and their abbreviations, and anything else, -echo included, is a
// setting. "drain" and "-drain" do not count as settings to change.
func parseArgs(args []string) (opts stty.Options, settings []string, err error) {
	files := 0
	setFile := func(f string) error {
		files++
		if files > 1 {
			return &fatal{"only one device may be specified"}
		}
		opts.File = f
		return nil
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			settings = append(settings, args[i+1:]...)
			return opts, settings, nil
		case strings.HasPrefix(a, "--") && len(a) > 2:
			name, value, hasValue := strings.Cut(a[2:], "=")
			switch {
			case abbreviates(name, "all") && !hasValue:
				opts.All = true
			case abbreviates(name, "save") && !hasValue:
				opts.Save = true
			case abbreviates(name, "file"):
				if !hasValue {
					if i+1 >= len(args) {
						return opts, nil, core.Usagef("option '--file' requires an argument")
					}
					i++
					value = args[i]
				}
				if err := setFile(value); err != nil {
					return opts, nil, err
				}
			default:
				settings = append(settings, a)
			}
		case strings.HasPrefix(a, "-") && len(a) > 1:
			// Letters are options until one that is not: then the whole
			// argument is a setting, the letters before it still counting.
			isSetting := false
		letters:
			for j := 1; j < len(a); j++ {
				switch a[j] {
				case 'a':
					opts.All = true
				case 'g':
					opts.Save = true
				case 'F':
					value := a[j+1:]
					if value == "" {
						if i+1 >= len(args) {
							return opts, nil, core.Usagef("option requires an argument -- 'F'")
						}
						i++
						value = args[i]
					}
					if err := setFile(value); err != nil {
						return opts, nil, err
					}
					break letters
				default:
					isSetting = true
					break letters
				}
			}
			if isSetting {
				settings = append(settings, a)
			}
		default:
			settings = append(settings, a)
		}
	}
	return opts, settings, nil
}

// abbreviates reports whether name is long, or an abbreviation of it.
func abbreviates(name, long string) bool { return strings.HasPrefix(long, name) }

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	view, args, err := viewArgs(fs, args, s)
	if err != nil {
		return err
	}
	opts, settings, err := parseArgs(args)
	var f *fatal
	if errors.As(err, &f) {
		d := core.NewDiag(stty.Name, s)
		d.Errorf("%s", f.msg)
		return d.Result()
	}
	if err != nil {
		return err
	}
	opts.Pretty, opts.Color, opts.Theme = view.Pretty, view.Color, view.Theme
	return stty.New(opts, s).Run(settings)
}

// viewArgs takes gnux's view options, --pretty[=WHEN] and --theme=NAME, off
// the command line before "--". They are matched only in full, so no GNU
// setting or abbreviation changes meaning, and resolved as every tool's.
func viewArgs(fs *pflag.FlagSet, args []string, s core.Streams) (tool.View, []string, error) {
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		name, value, hasValue := strings.Cut(strings.TrimPrefix(a, "--"), "=")
		switch {
		case a == "--":
			rest = append(rest, args[i:]...)
			i = len(args)
			continue
		case !strings.HasPrefix(a, "--") || (name != "pretty" && name != "theme"):
			rest = append(rest, a)
			continue
		case name == "pretty" && !hasValue:
			value = "always"
		case name == "theme" && !hasValue:
			if i+1 >= len(args) {
				return tool.View{}, nil, core.Usagef("option '--theme' requires an argument")
			}
			i++
			value = args[i]
		}
		if err := fs.Set(name, value); err != nil {
			return tool.View{}, nil, core.Usagef("%s", err)
		}
	}
	view, err := tool.ViewOf(fs, core.IsTerminal(s.Out), stty.Name, os.Getenv, s.Err)
	return view, rest, err
}

// Meta returns the documentation metadata for sttyx.
func Meta() docmeta.Meta {
	n := stty.Name
	return docmeta.Meta{
		Name:    n,
		SeeAlso: []string{"ttyx", "dircolorsx"},
		Use:     n + " [-F DEVICE | --file=DEVICE] [SETTING]...\n" + n + " [-F DEVICE | --file=DEVICE] [-a|--all]\n" + n + " [-F DEVICE | --file=DEVICE] [-g|--save]",
		Short:   "Print or change terminal line settings.",
		Long: n + " works on the terminal on standard input, or the DEVICE -F names. With no " +
			"SETTING it prints the speed and the settings that differ from 'sane' ones; -a " +
			"prints them all, and -g prints them in a form " + n + " takes back as a setting, to " +
			"restore them later. A SETTING turns a mode on (echo) or, with a leading -, off " +
			"(-echo); sets a control character (intr ^C, erase ^?, eof undef); sets min and " +
			"time for non-canonical reads, the speed (9600, or ispeed and ospeed), the window " +
			"size (rows, cols) or the line discipline (Linux); prints the size or the speed " +
			"(size, speed); or applies a combination: sane resets everything to sensible " +
			"values, raw and cooked switch line editing off and on, cbreak and -icanon read a " +
			"key at a time, and evenp, oddp, nl, ek, pass8, litout, tabs, lcase, crt and dec " +
			"are GNU's other shortcuts. Changes wait for pending output to drain unless -drain " +
			"is given. The modes are those the system has, so the list differs between Linux, " +
			"macOS and FreeBSD, as with GNU stty built there. After setting, " + n + " reads the " +
			"settings back and fails if the terminal did not take them all. Windows has no " +
			"termios, so there every SETTING is checked and then fails with 'Operation not " +
			"supported'.\n\n" +
			"gnux's terminal view (BR-13): on a terminal, -a is a panel titled with the device: the speed and size, the control characters in an aligned grid, and the control, input, output and local flags each on lines of their own, set flags colored and unset ones dimmed. -g, the plain listing and settings are GNU's. --pretty[=WHEN] and --theme=NAME, options of gnux's own matched only in full, choose the view (GNUX_PRETTY, GNUX_THEME); piped or redirected output is always GNU's.",
		Permissions: "Needs read access to the terminal; changing it needs no more, but a DEVICE " +
			"given with -F must be one you may open.",
		UseCases: []string{
			"Reading a password in a script without showing it.",
			"Restoring a terminal a crashed program left in raw mode (stty sane).",
			"Saving the settings before a change and restoring them afterwards.",
			"Setting the speed and framing of a serial line.",
			"Finding the size of the terminal in a script.",
		},
		Examples: []docmeta.Example{
			{Description: "Read a password without echoing it:", Bash: n + " -echo; read -r pass; " + n + " echo; echo"},
			{Description: "Fix a terminal that a program left in a bad state:", Bash: n + " sane"},
			{Description: "Save the settings, change them, and restore them:", Bash: "saved=$(" + n + " -g)\n" + n + " raw -echo\n# ...\n" + n + " \"$saved\""},
			{Description: "Print the terminal's rows and columns:", Bash: n + " size", Output: "24 80"},
			{Description: "Set a serial line to 115200 baud, 8 data bits, no parity:", Bash: n + " -F /dev/ttyUSB0 115200 cs8 -parenb -cstopb"},
			{Description: "See every setting:", Bash: n + " -a"},
			{Description: "Make Ctrl-H erase, for an old terminal:", Bash: n + " erase ^H"},
		},
	}
}
