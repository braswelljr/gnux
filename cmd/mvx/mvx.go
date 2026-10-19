// Package mvx wires the mvx command (GNU mv) to its move engine.
package mvx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/mv"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the mvx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "mv",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for mvx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    mv.Name,
		SeeAlso: []string{"cpx", "rmx", "lnx"},
		Use:     mv.Name + " [OPTION]... SOURCE... DIRECTORY",
		Short:   "Move (rename) files and directories.",
		Long: "mvx renames SOURCE to DEST, or moves several SOURCEs into DIRECTORY (also " +
			"selectable with -t; -T forces the two-operand form). Within one filesystem a move " +
			"is an atomic rename. Across filesystems mvx copies the source with all of its " +
			"attributes, symbolic links and hard links intact and removes the original only " +
			"after the copy has fully succeeded; --no-copy refuses such a move instead ('Invalid " +
			"cross-device link'), so a move is always a rename. An existing destination is replaced unless -n " +
			"(never), -u (only when the source is newer) or -i (ask first) says otherwise; a " +
			"directory is never moved into itself, and a directory only replaces an empty " +
			"directory. GNU mv lets the last of -f, -i and -n win; mvx cannot see option order, " +
			"so it prefers the most cautious instead: -n over -i over -f.",
		Permissions: "Renaming needs write and search permission on both the source and " +
			"destination directories, not on the file itself; in a sticky directory you must " +
			"also own the entry. On a terminal, replacing a write-protected destination asks " +
			"for confirmation unless -f is given. A cross-filesystem move additionally needs " +
			"read permission on everything being moved, and preserves ownership only when " +
			"privileged.",
		UseCases: []string{
			"Renaming a file or directory in place.",
			"Sorting files into a directory, including with find/xargs via -t.",
			"Relocating a tree to another disk without losing modes, times or links.",
			"Protecting existing files in scripts with -n, or reviewing overwrites with -i.",
			"Promoting a staged file over the live one only when it is newer (-u).",
			"Publishing a file atomically, never by a slow copy, with --no-copy.",
			"Atomically swapping a freshly written file into place after it is complete.",
			"Replacing a directory's name with -T instead of moving into it.",
			"Logging every move in a cleanup script with -v.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Rename a file:",
				Bash:        "mvx draft.md final.md",
				PowerShell:  "mvx draft.md final.md",
			},
			{
				Description: "Move several files into a directory, reporting each one:",
				Bash:        "mvx -v a.log b.log archive/",
				PowerShell:  "mvx -v a.log b.log archive",
			},
			{
				Description: "Move without ever overwriting an existing file:",
				Bash:        "mvx -n photo.jpg ~/Pictures/",
				PowerShell:  "mvx -n photo.jpg $HOME\\Pictures",
			},
			{
				Description: "Name the destination directory first (handy with xargs):",
				Bash:        "mvx -t build/out lib.a lib.so",
				PowerShell:  "mvx -t build\\out lib.a lib.so",
			},
			{
				Description: "Ask before replacing an existing file:",
				Bash:        "mvx -i new-config.yml config.yml",
				PowerShell:  "mvx -i new-config.yml config.yml",
			},
			{
				Description: "Replace a write-protected destination without being asked:",
				Bash:        "mvx -f settings.tmp settings.json",
				PowerShell:  "mvx -f settings.tmp settings.json",
			},
			{
				Description: "Replace the destination only when the source is newer:",
				Bash:        "mvx -u staging/index.html public/index.html",
				PowerShell:  "mvx -u staging\\index.html public\\index.html",
			},
			{
				Description: "Skip files that already exist at the destination, using --update:",
				Bash:        "mvx --update=none -v incoming/*.csv data/",
				PowerShell:  "mvx --update=none -v (Get-ChildItem incoming\\*.csv).FullName data",
			},
			{
				Description: "Rename a directory over an existing empty one instead of moving into it:",
				Bash:        "mvx -T build-new build",
				PowerShell:  "mvx -T build-new build",
			},
			{
				Description: "Move a directory given with a trailing slash:",
				Bash:        "mvx --strip-trailing-slashes logs/ old-logs",
				PowerShell:  "mvx --strip-trailing-slashes logs\\ old-logs",
			},
			{
				Description: "Write to a temporary file, then move it into place only on success:",
				Bash:        "echox \"generated\" > report.tmp && mvx report.tmp report.txt",
				PowerShell:  "echox \"generated\" > report.tmp; if ($LASTEXITCODE -eq 0) { mvx report.tmp report.txt }",
			},
			{
				Description: "Move every log file into an archive directory found with find:",
				Bash:        "find . -maxdepth 1 -name '*.log' -print0 | xargs -0 mvx -t archive/",
				PowerShell:  "mvx -t archive (Get-ChildItem -Filter *.log -Name)",
			},
			{
				Description: "Report an error when the move fails:",
				Bash:        "mvx missing.txt dest/ || echox \"move failed with status $?\"",
				PowerShell:  "mvx missing.txt dest; if ($LASTEXITCODE -ne 0) { echox \"move failed with status $LASTEXITCODE\" }",
			},
			{
				Description: "Archive text files into a dated directory with the gnux shell:",
				Bash:        "gnux shell -c 'mkdirx -p archive/2024 && mvx -v *.txt archive/2024/'",
				PowerShell:  "gnux shell -c 'mkdirx -p archive/2024 && mvx -v *.txt archive/2024/'",
			},
			{
				Description: "Publish a finished build atomically, or fail if it is on another disk:",
				Bash:        "mvx --no-copy build/site.new www/site",
				PowerShell:  "mvx --no-copy build\\site.new www\\site",
			},
		},
	}
}

// Register installs the mvx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("force", "f", false, "do not prompt before overwriting")
	fs.BoolP("interactive", "i", false, "prompt before overwrite")
	fs.BoolP("no-clobber", "n", false, "do not overwrite an existing file")
	fs.Bool("strip-trailing-slashes", false, "remove any trailing slashes from each SOURCE argument")
	fs.Bool("no-copy", false, "do not copy if renaming fails")
	fs.StringP("target-directory", "t", "", "move all SOURCE arguments into DIRECTORY")
	fs.BoolP("no-target-directory", "T", false, "treat DEST as a normal file")
	fs.StringP("update", "u", "", "control which existing files are updated: all, none, or older (the default for -u)")
	fs.Lookup("update").NoOptDefVal = "older"
	fs.BoolP("verbose", "v", false, "explain what is being done")
}

// buildOptions folds the parsed flags into mv.Options. GNU mv lets the last
// of -f, -i and -n win, which pflag cannot reproduce because it does not
// expose option order; the most cautious one wins instead (-n > -i > -f).
func buildOptions(fs *pflag.FlagSet) (mv.Options, error) {
	flag := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}

	o := mv.Options{
		Force:                flag("force"),
		Interactive:          flag("interactive"),
		NoClobber:            flag("no-clobber"),
		Verbose:              flag("verbose"),
		NoTargetDirectory:    flag("no-target-directory"),
		StripTrailingSlashes: flag("strip-trailing-slashes"),
		NoCopy:               flag("no-copy"),
	}
	o.TargetDirectory, _ = fs.GetString("target-directory")

	if fs.Changed("update") {
		when, _ := fs.GetString("update")
		switch when {
		case "older":
			o.Update = true
		case "none":
			o.NoClobber = true
		case "all":
		default:
			return mv.Options{}, core.Usagef("invalid argument '%s' for '--update'\nValid arguments are:\n  - 'all'\n  - 'none'\n  - 'older'", when)
		}
	}

	switch {
	case o.NoClobber:
		o.Interactive, o.Force = false, false
	case o.Interactive:
		o.Force = false
	}
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return mv.New(opts, s).Run(args)
}
