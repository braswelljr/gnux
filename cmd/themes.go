package cmd

import (
	"os"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// ThemesSpec describes gnux themes, which shows the built-in themes of the
// terminal view. Like the shell, it is a gnux command and not a tool.
func ThemesSpec() tool.Spec {
	return tool.Spec{
		Meta: docmeta.Meta{
			Name:    "themes",
			SeeAlso: []string{"lsx", "dfx", "dux", "grepx"},
			Use:     "themes [OPTION]...",
			Short:   "Show the built-in themes of the gnux terminal view.",
			Long: "gnux themes shows every built-in theme of the terminal view, which the " +
				"tools' --theme option and the GNUX_THEME variable take, each drawn on a " +
				"sample row in its own colors: a long listing's columns, a check's OK and " +
				"FAILED, and a usage bar. A panel at the top names the theme in use and the " +
				"background gnux has read.\n\n" +
				"Most themes come in a dark and a light variant, named FAMILY-dark and " +
				"FAMILY-light after the editor themes they follow (vscode, github, one, " +
				"solarized, gruvbox, catppuccin, tokyo-night, ayu, rose-pine, everforest, " +
				"night-owl, high-contrast). The family's own name picks the variant that " +
				"suits the background: GNUX_BACKGROUND (dark or light) when set, else the " +
				"terminal's COLORFGBG, else the system's appearance (dark mode on macOS and " +
				"Windows, a dark GTK theme), else dark. A few themes have one, dark, variant " +
				"(dracula, nord, monokai, kanagawa, palenight, ayu-mirage); default uses the " +
				"16 basic colors, so it follows the terminal's own scheme, and mono uses no " +
				"color at all, only bold, dim and underline.\n\n" +
				"When standard output is not a terminal, or under NO_COLOR, gnux themes " +
				"prints the names alone, one a line; --pretty=always draws them anyway.",
			UseCases: []string{
				"Choosing a theme that suits your terminal before setting GNUX_THEME.",
				"Checking which variant a family picks on this machine.",
				"Listing the theme names in a script or a completion.",
			},
			Examples: []docmeta.Example{
				{
					Description: "Your listings are hard to read on a light terminal: see every theme, then keep one that follows the background:",
					Bash:        "gnux themes\necho 'export GNUX_THEME=github' >> ~/.bashrc",
					PowerShell:  "gnux themes\nAdd-Content $PROFILE '$env:GNUX_THEME = \"github\"'",
				},
				{
					Description: "Your terminal is light but gnux picks the dark variants: say so once:",
					Bash:        "echo 'export GNUX_BACKGROUND=light' >> ~/.bashrc",
					PowerShell:  "Add-Content $PROFILE '$env:GNUX_BACKGROUND = \"light\"'",
				},
				{
					Description: "Try a theme on a real listing before keeping it:",
					Bash:        "lsx -l --theme=tokyo-night",
					PowerShell:  "lsx -l --theme=tokyo-night",
				},
			},
		},
		Flags: tool.RegisterView,
		Run:   runThemes,
	}
}

func runThemes(fs *pflag.FlagSet, args []string, s core.Streams) error {
	if len(args) > 0 {
		return core.Usagef("extra operand '%s'", args[0])
	}
	view, err := tool.ViewOf(fs, core.IsTerminal(s.Out), "themes", os.Getenv, s.Err)
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
