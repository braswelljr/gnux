// Package dirx wires the dirx command (GNU dir) to the lsx engine.
package dirx

import (
	"github.com/braswelljr/gnux/cmd/lsx"
	"github.com/braswelljr/gnux/core/ls"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "dirx"

// Spec describes the dirx tool: lsx with dir's defaults.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "dir",
		Flags: lsx.Register,
		Run:   lsx.RunAs(Name, ls.FormatColumns),
		// As for lsx, a bad command line is "serious trouble".
		UsageExit: 2,
	}
}

// Meta returns the documentation metadata for dirx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:       Name,
		SeeAlso:    []string{"lsx", "vdirx"},
		ExitStatus: lsx.Meta().ExitStatus,
		Use:        Name + " [OPTION]... [FILE]...",
		Short:      "List directory contents in columns, whether or not the output is a terminal.",
		Long: "dirx is lsx -C -b: it lists each FILE, or the current directory, exactly as lsx " +
			"does and takes every lsx option, but arranges the names in columns even when the " +
			"output is piped or redirected (80 columns wide then, or -w COLS), and writes " +
			"unprintable characters in names as backslash escapes (\\n, \\t, \\ooo) so that " +
			"every name fits on one line. An option that picks a format (-l, -1, -x, -m) or a " +
			"quoting style (-Q) replaces those defaults. Diagnostics and exit statuses are " +
			"lsx's, prefixed with dirx.",
		Permissions: "Listing a directory needs read permission on it; examining its entries " +
			"(-l, -s, -i, -F, --color, sorting by time or size) also needs search permission. " +
			"Nothing is ever modified.",
		UseCases: []string{
			"Getting a compact column listing in a log or a file, where lsx prints one name per line.",
			"Listing a directory the way Windows users expect a dir command to work.",
			"Seeing names with newlines or tabs in them as escapes rather than broken lines.",
			"Keeping a script's listing the same whether or not it runs in a terminal.",
			"Comparing directory contents side by side in a narrow terminal with -w.",
		},
		Examples: []docmeta.Example{
			{Description: "List the current directory in columns:", Bash: "dirx", PowerShell: "dirx"},
			{Description: "Keep the columns when writing to a file:", Bash: "dirx > listing.txt", PowerShell: "dirx > listing.txt"},
			{Description: "Include hidden entries:", Bash: "dirx -a", PowerShell: "dirx -a"},
			{Description: "List entries across rather than down:", Bash: "dirx -x src", PowerShell: "dirx -x src"},
			{Description: "Fit the columns in 40 characters:", Bash: "dirx -w 40 /etc", PowerShell: "dirx -w 40 C:\\Windows"},
			{Description: "Mark directories and executables:", Bash: "dirx -F", PowerShell: "dirx -F"},
			{Description: "Sort by modification time, newest first:", Bash: "dirx -t", PowerShell: "dirx -t"},
			{Description: "Show sizes in blocks next to each name:", Bash: "dirx -sh", PowerShell: "dirx -sh"},
			{Description: "Switch to the long format, as vdirx does:", Bash: "dirx -l", PowerShell: "dirx -l"},
			{Description: "List a directory in the portable gnux shell:", Bash: "gnux shell -c 'dirx -A dist'", PowerShell: "gnux shell -c 'dirx -A dist'"},
		},
	}
}
