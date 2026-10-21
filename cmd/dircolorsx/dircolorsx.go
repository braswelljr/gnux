// Package dircolorsx wires the dircolorsx command (GNU dircolors) to its
// engine.
package dircolorsx

import (
	"strconv"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/dircolors"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the dircolorsx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "dircolors",
		Flags: Register,
		Run:   run,
	}
}

// shell is a shell-syntax switch. The switches of a flag set share one
// selection, which the switch given last sets, so that the last of -b and -c
// wins, as in GNU dircolors.
type shell struct {
	syntax dircolors.Shell
	sel    *dircolors.Shell
}

func (s *shell) Set(v string) error {
	on, err := strconv.ParseBool(v)
	if err == nil && on {
		*s.sel = s.syntax
	}
	return err
}
func (s *shell) String() string   { return strconv.FormatBool(s.sel != nil && *s.sel == s.syntax) }
func (s *shell) Type() string     { return "bool" }
func (s *shell) IsBoolFlag() bool { return true }

// Register installs the dircolorsx flags on fs. --sh and --csh have the
// second spellings --bourne-shell and --c-shell, as in GNU.
func Register(fs *pflag.FlagSet) {
	sel := new(dircolors.Shell)
	for _, f := range []struct {
		name, short string
		syntax      dircolors.Shell
		usage       string
		hidden      bool
	}{
		{"sh", "b", dircolors.Bourne, "output Bourne shell code to set LS_COLORS (also --bourne-shell)", false},
		{"bourne-shell", "", dircolors.Bourne, "output Bourne shell code to set LS_COLORS", true},
		{"csh", "c", dircolors.C, "output C shell code to set LS_COLORS (also --c-shell)", false},
		{"c-shell", "", dircolors.C, "output C shell code to set LS_COLORS", true},
	} {
		flag := fs.VarPF(&shell{syntax: f.syntax, sel: sel}, f.name, f.short, f.usage)
		flag.NoOptDefVal = "true"
		flag.Hidden = f.hidden
	}
	fs.BoolP("print-database", "p", false, "output defaults")
	fs.Bool("print-ls-colors", false, "output fully escaped colors for display")
}

// buildOptions folds the parsed flags into dircolors.Options.
func buildOptions(fs *pflag.FlagSet) dircolors.Options {
	var o dircolors.Options
	o.Shell = *fs.Lookup("sh").Value.(*shell).sel
	o.PrintDatabase, _ = fs.GetBool("print-database")
	o.PrintLsColors, _ = fs.GetBool("print-ls-colors")
	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return dircolors.New(buildOptions(fs), s).Run(args)
}

// Meta returns the documentation metadata for dircolorsx.
func Meta() docmeta.Meta {
	n := dircolors.Name
	return docmeta.Meta{
		Name:    n,
		SeeAlso: []string{"lsx", "dirx", "vdirx"},
		Use:     n + " [OPTION]... [FILE]",
		Short:   "Print commands that set LS_COLORS, the colors of lsx --color.",
		Long: n + " reads a color database, FILE or GNU's built-in one, and prints shell " +
			"commands that set the LS_COLORS environment variable from it, for lsx --color " +
			"and GNU ls to use. -b prints Bourne shell syntax (LS_COLORS='...'; export " +
			"LS_COLORS), -c C shell syntax (setenv LS_COLORS '...'); without either, the " +
			"syntax follows $SHELL: C shell for csh and tcsh, Bourne shell for anything else, " +
			"and an error when SHELL is not set. -p prints the built-in database, a starting " +
			"point for a FILE of your own, and --print-ls-colors prints each entry in its own " +
			"color. A database line is a keyword and a value: a file type (DIR, LINK, EXEC, ...), " +
			"a file name extension (.tar) or pattern (*~), and TERM or COLORTERM lines whose " +
			"glob patterns say which terminals the lines after them apply to. The built-in " +
			"database is GNU coreutils' own. PowerShell has no syntax of its own: use -b and " +
			"take the value between the quotes, as the example shows.",
		Permissions: "Needs read permission on FILE. Prints commands; changes nothing itself.",
		UseCases: []string{
			"Setting up colors for lsx --color in a shell profile.",
			"Customizing the colors: print the database with -p, edit it, and load the result.",
			"Previewing a color scheme with --print-ls-colors.",
		},
		Examples: []docmeta.Example{
			{Description: "Load the default colors in a Bourne-style shell profile:", Bash: "eval \"$(" + n + " -b)\""},
			{Description: "Use your own database, kept in your home directory:", Bash: n + " -p > ~/.dircolors\neval \"$(" + n + " -b ~/.dircolors)\""},
			{Description: "Set the colors in PowerShell:", PowerShell: "$env:LS_COLORS = (" + n + " -b | Select-Object -First 1) -replace \"^LS_COLORS='(.*)';$\", '$1'"},
			{Description: "See every color the database defines, in that color:", Bash: n + " --print-ls-colors", PowerShell: n + " --print-ls-colors"},
			{Description: "Print C shell commands for a tcsh profile:", Bash: n + " -c >> ~/.tcshrc"},
		},
	}
}
