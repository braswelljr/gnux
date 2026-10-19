// Package touchx wires the touchx command (GNU touch) to its engine.
package touchx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/touch"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the touchx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "touch",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for touchx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    touch.Name,
		SeeAlso: []string{"lsx", "mkdirx", "catx"},
		Use:     touch.Name + " [OPTION]... FILE...",
		Short:   "Update the access and modification times of each FILE to the current time.",
		Long: "touchx sets the access and modification times of each FILE to the current " +
			"time, creating an empty file for any FILE that does not exist unless -c is given. " +
			"An existing file's contents are never changed. -a and -m (or --time) restrict the " +
			"update to one of the two timestamps and leave the other exactly as it was. The " +
			"time to apply can come from somewhere other than the clock: -d takes a date " +
			"string (2024-01-02, '2024-01-02 15:04:05', an RFC 3339 timestamp, @SECONDS since " +
			"the epoch, or now), -t takes a [[CC]YY]MMDDhhmm[.ss] stamp, and -r copies the " +
			"times of another file. -h acts on a symbolic link itself rather than the file it " +
			"points to.",
		Permissions: "Setting a file's times to the current time needs write permission on " +
			"it; setting them to any other time (-d, -t, -r) needs ownership of the file. " +
			"Creating a missing file needs write and search permission on its directory. " +
			"Changing the times of a symbolic link itself (-h) is only possible on Linux; " +
			"elsewhere touchx reports the link rather than silently touching its target.",
		UseCases: []string{
			"Creating an empty placeholder or marker file.",
			"Forcing a rebuild by making a source file newer than its outputs.",
			"Stamping files with a fixed time for reproducible archives.",
			"Copying timestamps from one file to another with -r.",
			"Refreshing the timestamps of existing files only, without creating strays, with -c.",
			"Recording when a step ran by touching a marker file that later steps compare against.",
			"Backdating a file with -d or -t to test retention or cleanup rules.",
			"Changing only the access time (-a) or only the modification time (-m).",
		},
		Examples: []docmeta.Example{
			{
				Description: "Create a file, or bump its timestamps if it exists:",
				Bash:        "touchx notes.txt",
				PowerShell:  "touchx notes.txt",
			},
			{
				Description: "Create several empty files at once:",
				Bash:        "touchx a.txt b.txt c.txt",
				PowerShell:  "touchx a.txt b.txt c.txt",
			},
			{
				Description: "Update timestamps without creating anything:",
				Bash:        "touchx -c build/*.o",
				PowerShell:  "touchx -c (Get-ChildItem build\\*.o)",
			},
			{
				Description: "Set only the modification time to a specific date:",
				Bash:        "touchx -m -d '2024-01-02 15:04:05' report.pdf",
				PowerShell:  "touchx -m -d '2024-01-02 15:04:05' report.pdf",
			},
			{
				Description: "Set only the access time to now:",
				Bash:        "touchx -a app.log",
				PowerShell:  "touchx -a app.log",
			},
			{
				Description: "Choose the timestamp by name with --time (access/atime/use or modify/mtime):",
				Bash:        "touchx --time=mtime app.log",
				PowerShell:  "touchx --time=mtime app.log",
			},
			{
				Description: "Use a [[CC]YY]MMDDhhmm[.ss] stamp:",
				Bash:        "touchx -t 202401021504.05 report.pdf",
				PowerShell:  "touchx -t 202401021504.05 report.pdf",
			},
			{
				Description: "Backdate only the access time with a stamp:",
				Bash:        "touchx -a -t 202001010000 app.log",
				PowerShell:  "touchx -a -t 202001010000 app.log",
			},
			{
				Description: "Use an RFC 3339 timestamp, or seconds since the epoch:",
				Bash:        "touchx -d 2024-06-01T12:00:00Z release.txt && touchx -d @0 epoch.txt",
				PowerShell:  "touchx -d 2024-06-01T12:00:00Z release.txt; touchx -d '@0' epoch.txt",
			},
			{
				Description: "Give one file the timestamps of another:",
				Bash:        "touchx -r original.jpg edited.jpg",
				PowerShell:  "touchx -r original.jpg edited.jpg",
			},
			{
				Description: "Copy just the modification time, and only to files that already exist:",
				Bash:        "touchx -c -m -r original.jpg edited.jpg report.pdf",
				PowerShell:  "touchx -c -m -r original.jpg edited.jpg report.pdf",
			},
			{
				Description: "Stamp all sources with one fixed date for a reproducible archive:",
				Bash:        "touchx -d 2024-01-01 src/*.go",
				PowerShell:  "touchx -d 2024-01-01 (Get-ChildItem src\\*.go)",
			},
			{
				Description: "Change the times of a symbolic link itself, not its target (Linux only):",
				Bash:        "touchx -h -d 2024-01-01 link.txt",
				PowerShell:  "touchx -h -d 2024-01-01 link.txt",
			},
			{
				Description: "Create a marker file inside a new directory:",
				Bash:        "mkdirx -p cache && touchx cache/.keep",
				PowerShell:  "mkdirx -p cache; touchx cache\\.keep",
			},
			{
				Description: "Stamp files and list them with the gnux shell:",
				Bash:        "gnux shell -c 'touchx -d 2024-01-01 a.txt b.txt && lsx -l a.txt b.txt'",
				PowerShell:  "gnux shell -c 'touchx -d 2024-01-01 a.txt b.txt && lsx -l a.txt b.txt'",
			},
		},
	}
}

// Register installs the touchx flags on fs. GNU touch has no long names for
// -a, -m and -t; pflag requires one, so descriptive names stand in.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("access-time", "a", false, "change only the access time")
	fs.BoolP("no-create", "c", false, "do not create any files")
	fs.StringP("date", "d", "", "parse `STRING` and use it instead of current time")
	fs.BoolP("no-dereference", "h", false,
		"affect each symbolic link instead of any referenced file")
	fs.BoolP("modification-time", "m", false, "change only the modification time")
	fs.StringP("reference", "r", "", "use this `FILE`'s times instead of current time")
	fs.StringP("stamp", "t", "", "use [[CC]YY]MMDDhhmm[.ss] `STAMP` instead of current time")
	fs.String("time", "", "change the specified time: `WORD` is access, atime, or use "+
		"(like -a), or modify or mtime (like -m)")

	// Long names GNU does not have: accepted, never an abbreviation.
	tool.OwnNames(fs, "access-time", "modification-time", "stamp")
}

// buildOptions folds the parsed flags into touch.Options.
func buildOptions(fs *pflag.FlagSet) touch.Options {
	var o touch.Options
	o.Access, _ = fs.GetBool("access-time")
	o.Modify, _ = fs.GetBool("modification-time")
	o.NoCreate, _ = fs.GetBool("no-create")
	o.Date, _ = fs.GetString("date")
	o.Stamp, _ = fs.GetString("stamp")
	o.Reference, _ = fs.GetString("reference")
	o.NoDereference, _ = fs.GetBool("no-dereference")
	o.Time, _ = fs.GetString("time")
	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return touch.New(buildOptions(fs), s).Run(args)
}
