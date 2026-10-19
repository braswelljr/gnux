// Package vdirx wires the vdirx command (GNU vdir) to the lsx engine.
package vdirx

import (
	"github.com/braswelljr/gnux/cmd/lsx"
	"github.com/braswelljr/gnux/core/ls"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "vdirx"

// Spec describes the vdirx tool: lsx with vdir's defaults.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "vdir",
		Flags: lsx.Register,
		Run:   lsx.RunAs(Name, ls.FormatLong),
		// As for lsx, a bad command line is "serious trouble".
		UsageExit: 2,
	}
}

// Meta returns the documentation metadata for vdirx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:       Name,
		SeeAlso:    []string{"lsx", "dirx", "statx"},
		ExitStatus: lsx.Meta().ExitStatus,
		Use:        Name + " [OPTION]... [FILE]...",
		Short:      "List directory contents in the long format.",
		Long: "vdirx is lsx -l -b: it lists each FILE, or the current directory, exactly as " +
			"lsx does and takes every lsx option, but in the long format (mode, link count, " +
			"owner, group, size, timestamp and name) by default, and writes unprintable " +
			"characters in names as backslash escapes (\\n, \\t, \\ooo) so that every entry " +
			"fits on one line. An option that picks a format (-1, -C, -x, -m) or a quoting " +
			"style (-Q) replaces those defaults. Diagnostics and exit statuses are lsx's, " +
			"prefixed with vdirx.",
		Permissions: "Listing a directory needs read permission on it, and the long format " +
			"needs search permission to examine each entry. Nothing is ever modified.",
		UseCases: []string{
			"Seeing permissions, owners, sizes and times without remembering -l.",
			"Checking what a build produced, with sizes in human units (-h).",
			"Finding the most recently changed files with -t.",
			"Recording a directory's state in a log, one entry per line even for odd names.",
			"Giving Windows users a detailed directory listing under a familiar name.",
		},
		Examples: []docmeta.Example{
			{Description: "List the current directory in detail:", Bash: "vdirx", PowerShell: "vdirx"},
			{Description: "Show sizes in human units:", Bash: "vdirx -h dist", PowerShell: "vdirx -h dist"},
			{Description: "Include hidden entries:", Bash: "vdirx -a", PowerShell: "vdirx -a"},
			{Description: "Newest files first:", Bash: "vdirx -t", PowerShell: "vdirx -t"},
			{Description: "Largest files first:", Bash: "vdirx -S", PowerShell: "vdirx -S"},
			{Description: "Show full timestamps:", Bash: "vdirx --full-time", PowerShell: "vdirx --full-time"},
			{Description: "Show numeric user and group IDs:", Bash: "vdirx -n", PowerShell: "vdirx -n"},
			{Description: "Describe a directory itself, not its contents:", Bash: "vdirx -d src", PowerShell: "vdirx -d src"},
			{Description: "Go back to names only:", Bash: "vdirx -1", PowerShell: "vdirx -1"},
			{Description: "List a directory in the portable gnux shell:", Bash: "gnux shell -c 'vdirx -h dist'", PowerShell: "gnux shell -c 'vdirx -h dist'"},
		},
	}
}
