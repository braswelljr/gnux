package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/internal/config"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/themepick"
	"github.com/braswelljr/gnux/internal/tool"
)

// themeName prefixes gnux theme's diagnostics.
const themeName = "gnux theme"

// ThemeSpec describes gnux theme, which picks, saves and lists the built-in
// themes of the terminal view. Like the shell, it is a gnux command and not
// a tool.
func ThemeSpec() tool.Spec {
	return tool.Spec{
		Meta: docmeta.Meta{
			Name:    "theme",
			SeeAlso: []string{"lsx", "dfx", "dux", "grepx"},
			Use:     "theme [list | show | set NAME | reset] [OPTION]...",
			Short:   "Pick, save and list the built-in themes of the gnux terminal view.",
			Long: "gnux theme chooses the colors of gnux's terminal view from its built-in themes. " +
				"Run alone on a terminal, it opens a picker: the themes on the left, the one under " +
				"the cursor drawn on the right on a long listing, Git's marks, a check's verdicts, " +
				"usage bars and an environment line. Up and Down (or Ctrl-P and Ctrl-N), Page " +
				"Up and Page Down, Home and End move; typing filters the list by name, Backspace " +
				"and Ctrl-U edit the filter; Enter saves the theme and Esc or Ctrl-C leaves without " +
				"saving. The picker draws on the terminal's alternate screen, so what was on screen " +
				"comes back when it ends.\n\n" +
				"Tab and Shift-Tab (or Right and Left) switch between the groups: All, Adaptive (a family's " +
				"own name, which follows the background, with default and mono), Dark and Light. " +
				"gnux theme list draws every theme on a sample row (the names alone through a pipe); " +
				"gnux themes is the same. gnux theme show prints the theme in use and where it comes " +
				"from (the name alone through a pipe). gnux theme set NAME saves NAME without the " +
				"picker, for scripts, and gnux theme reset removes the saved theme; choosing default, in the " +
				"picker or with set, does the same.\n\n" +
				"The saved theme is a file holding the theme's name: $XDG_CONFIG_HOME/gnux/theme, else " +
				"~/.config/gnux/theme (%AppData%\\gnux\\theme on Windows), or theme in GNUX_CONFIG_DIR " +
				"when that is set. Only a built-in name is accepted, the file is replaced whole and " +
				"only its owner may read or write it. A tool reads it only when its output is colored " +
				"and neither --theme nor GNUX_THEME names a theme, so --theme wins over GNUX_THEME, " +
				"which wins over the saved theme, which wins over the default; piped output never " +
				"reads it. A file gnux would not have written (another name, more than 64 bytes, " +
				"not a regular file) is reported and ignored.\n\n" +
				"Most themes come in a dark and a light variant, named FAMILY-dark and FAMILY-light " +
				"after the editor themes they follow (vscode, github, one, solarized, gruvbox, " +
				"catppuccin, tokyo-night, ayu, rose-pine, everforest, night-owl, high-contrast, " +
				"material, tomorrow, selenized, flexoki, xcode), Apple's system colors (ios), or " +
				"gnux's own: colorful, every hue at full strength, and monochrome, shades of gray. " +
				"The family's own name picks the variant that suits the background: GNUX_BACKGROUND " +
				"(dark or light) when set, else the terminal's COLORFGBG, else the system's appearance " +
				"(dark mode on macOS and Windows, a dark GTK theme), else dark. Some themes have one, " +
				"dark, variant (dracula, nord, monokai, kanagawa, palenight, ayu-mirage, zenburn, " +
				"synthwave, horizon, moonlight); default uses the 16 basic colors, so it follows the " +
				"terminal's own scheme, and mono uses no color at all, only bold, dim and underline.",
			ExitStatus: []docmeta.Status{
				{Code: "0", Meaning: "the theme was shown, saved or reset, or the picker was left without saving"},
				{Code: "1", Meaning: "the theme could not be saved or removed, or a command-line mistake"},
			},
			UseCases: []string{
				"Choosing a theme by trying each on sample output, then keeping it for every tool.",
				"Setting the theme from a dotfiles script: gnux theme set github.",
				"Checking which theme is in use and why: gnux theme show.",
				"Listing the theme names in a script or a completion: gnux theme list | catx.",
			},
			Examples: []docmeta.Example{
				{
					Description: "Browse the themes with a live preview and keep the one you like:",
					Bash:        "gnux theme",
					PowerShell:  "gnux theme",
				},
				{
					Description: "Your terminal is light: keep the solarized family, which then takes its light variant:",
					Bash:        "gnux theme set solarized",
					PowerShell:  "gnux theme set solarized",
				},
				{
					Description: "Try a theme on a real listing before keeping it, then see what is in use:",
					Bash:        "lsx -l --theme=tokyo-night\ngnux theme show",
					PowerShell:  "lsx -l --theme=tokyo-night\ngnux theme show",
				},
				{
					Description: "Go back to the default colors:",
					Bash:        "gnux theme reset",
					PowerShell:  "gnux theme reset",
				},
			},
		},
		Flags: tool.RegisterView,
		Run:   runTheme,
	}
}

// ThemesSpec is gnux themes, an alias of gnux theme kept for scripts that
// call it; gnux themes runs gnux theme list.
//
// Deprecated: use ThemeSpec.
func ThemesSpec() tool.Spec { return ThemeSpec() }

func runTheme(fs *pflag.FlagSet, args []string, s core.Streams) error {
	sub := ""
	if len(args) > 0 {
		sub, args = args[0], args[1:]
	}
	switch sub {
	case "":
		if in, out, ok := terminals(s); ok {
			return pickTheme(in, out, s, os.Getenv)
		}
		return showTheme(s, os.Getenv)
	case "list":
		if len(args) > 0 {
			return core.Usagef("extra operand '%s'", args[0])
		}
		return listThemes(fs, s)
	case "show":
		if len(args) > 0 {
			return core.Usagef("extra operand '%s'", args[0])
		}
		return showTheme(s, os.Getenv)
	case "set":
		switch {
		case len(args) == 0:
			return core.Usagef("missing theme name after 'set'")
		case len(args) > 1:
			return core.Usagef("extra operand '%s'", args[1])
		case !config.Valid(args[0]):
			return &core.UsageError{Msg: fmt.Sprintf("invalid theme '%s'\nValid themes are: %s",
				args[0], strings.Join(core.ThemeNames(), ", ")), Code: 1}
		}
		return saveTheme(strings.ToLower(args[0]), s, os.Getenv)
	case "reset":
		if len(args) > 0 {
			return core.Usagef("extra operand '%s'", args[0])
		}
		return resetTheme(s, os.Getenv)
	}
	return core.Usagef("unknown command '%s' (list, show, set NAME, reset)", sub)
}

// terminals returns the streams as files when both are terminals, which
// the picker needs: it reads single key presses and draws a whole screen.
func terminals(s core.Streams) (in, out *os.File, ok bool) {
	in, inFile := s.In.(*os.File)
	out, outFile := s.Out.(*os.File)
	if !inFile || !outFile || !core.IsTerminal(in) || !core.IsTerminal(out) {
		return nil, nil, false
	}
	return in, out, true
}

// listThemes draws every theme on a sample row, as gnux themes always has.
func listThemes(fs *pflag.FlagSet, s core.Streams) error {
	view, err := tool.ViewOf(fs, core.IsTerminal(s.Out), themeName, os.Getenv, s.Err)
	if err != nil {
		return err
	}
	getenv := os.Getenv
	if view.Theme != "" {
		getenv = func(k string) string {
			if k == "GNUX_THEME" {
				return view.Theme
			}
			return os.Getenv(k)
		}
	}
	return core.WriteThemes(s.Out, getenv, view.Color)
}

// activeTheme is the theme the tools paint with, and where it comes from.
func activeTheme(getenv func(string) string, warn io.Writer) (name, from string) {
	if n := tool.ThemeOfEnv(getenv("GNUX_THEME")); n != "" {
		return strings.ToLower(n), "GNUX_THEME"
	}
	saved, path, err := config.Theme(getenv)
	if err != nil {
		_, _ = fmt.Fprintf(warn, "%s: ignoring the saved theme in '%s': %v\n", themeName, path, err)
	}
	if saved != "" {
		return saved, path
	}
	return "default", "the built-in default"
}

func showTheme(s core.Streams, getenv func(string) string) error {
	name, from := activeTheme(getenv, s.Err)
	if !core.IsTerminal(s.Out) {
		_, err := fmt.Fprintln(s.Out, name)
		return err
	}
	line := name
	if r := core.ResolveTheme(name, getenv); r != name {
		line += " (" + r + " on this background)"
	}
	_, err := fmt.Fprintf(s.Out, "Theme  %s\nFrom   %s\n", line, from)
	return err
}

func saveTheme(name string, s core.Streams, getenv func(string) string) error {
	// default is what nothing saved means: choosing it removes the setting,
	// so the tools follow a later change of default too.
	if name == "default" {
		return resetTheme(s, getenv)
	}
	path, err := config.SaveTheme(getenv, name)
	if err != nil {
		core.NewDiag(themeName, s).Errorf("cannot save the theme in '%s': %s", path, core.ErrText(errors.Unwrap(err)))
		return core.ErrFailed
	}
	_, _ = fmt.Fprintf(s.Out, "Theme set to %s (saved in %s).\n", name, path)
	if env := tool.ThemeOfEnv(getenv("GNUX_THEME")); env != "" && !strings.EqualFold(env, name) {
		_, _ = fmt.Fprintf(s.Err, "%s: GNUX_THEME=%s is set, and wins over the saved theme until it is unset\n", themeName, env)
	}
	return nil
}

func resetTheme(s core.Streams, getenv func(string) string) error {
	path, removed, err := config.ResetTheme(getenv)
	if err != nil {
		if path == "" {
			core.NewDiag(themeName, s).Errorf("%v", err)
		} else {
			core.NewDiag(themeName, s).Errorf("cannot remove '%s': %s", path, core.ErrText(err))
		}
		return core.ErrFailed
	}
	if removed {
		_, _ = fmt.Fprintln(s.Out, "Theme reset: the tools use the default colors again.")
	} else {
		_, _ = fmt.Fprintln(s.Out, "No theme was saved: the tools use the default colors.")
	}
	return nil
}

func pickTheme(in, out *os.File, s core.Streams, getenv func(string) string) error {
	active, _ := activeTheme(getenv, s.Err)
	saved, _, _ := config.Theme(getenv)
	unicode := core.UnicodeTerminal(getenv)
	m := themepick.New(core.ThemeNames(), active, saved,
		func(name string) []string { return core.ThemeShowcase(name, getenv, unicode) },
		func(name string) string { return core.ResolveTheme(name, getenv) },
		unicode)
	chosen, ok, err := themepick.Run(in, out, m)
	if err != nil {
		core.NewDiag(themeName, s).Errorf("%v", err)
		return core.ErrFailed
	}
	if !ok {
		_, _ = fmt.Fprintln(s.Out, "Theme unchanged.")
		return nil
	}
	return saveTheme(chosen, s, getenv)
}
