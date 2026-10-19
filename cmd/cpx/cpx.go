// Package cpx wires the cpx command (GNU cp) to its copy engine.
package cpx

import (
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/cp"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// defaultPreserve is the attribute list selected by -p and a bare --preserve.
const defaultPreserve = "mode,ownership,timestamps"

// Spec describes the cpx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "cp",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for cpx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    cp.Name,
		SeeAlso: []string{"mvx", "lnx", "rmx", "mkdirx"},
		Use:     cp.Name + " [OPTION]... SOURCE... DIRECTORY",
		Short:   "Copy files and directories.",
		Long: "cpx copies SOURCE to DEST, or several SOURCEs into DIRECTORY (also selectable " +
			"with -t; -T forces the two-operand form). Directories are copied only with -r, and " +
			"a directory is never copied into itself. A new file receives the source's " +
			"permission bits filtered by the umask; -p or --preserve[=LIST] carries over mode, " +
			"ownership, timestamps and hard links, and -a is shorthand for -dR --preserve=all. " +
			"Symbolic links are followed unless copying recursively; -P/-d keep them as links, " +
			"-L follows them everywhere and -H only on the command line. -l and -s create hard " +
			"or symbolic links instead of copying data. Existing destinations are overwritten " +
			"unless -n (never), -u (only when the source is newer) or -i (ask first) says " +
			"otherwise; -f removes a destination that cannot be opened and tries again, and " +
			"--remove-destination removes every existing destination before writing it, so a " +
			"destination symbolic link is replaced rather than written through and a file " +
			"that other names share is not changed under them. GNU cp " +
			"lets the last of conflicting options win; cpx cannot see option order, so it " +
			"prefers the most cautious instead: -n overrides -i, and an explicit -P/-d overrides " +
			"-H, which overrides -L.",
		Permissions: "Copying needs read permission on each source and write and search " +
			"permission on the destination directory. Without -p the copy is owned by the " +
			"invoking user and its mode is the source's mode minus the umask. Preserving " +
			"ownership requires privilege; an ordinary user silently keeps their own ownership, " +
			"as with GNU cp. On Windows only the read-only attribute and modification time are " +
			"carried over.",
		UseCases: []string{
			"Duplicating a file before editing it, so the original can be restored.",
			"Copying a directory tree with -r, or a faithful archive of it with -a.",
			"Refreshing a mirror with -u so only newer files are rewritten.",
			"Filling in missing defaults without touching customized files (--update=none).",
			"Guarding existing files in scripts with -n, or interactively with -i.",
			"Building a tree of hard links (-al) or symlinks (-s) instead of duplicating data.",
			"Recreating selected files under a backup directory with their paths intact (--parents).",
			"Replacing a read-only destination in a deployment step with -f.",
			"Deploying over a file a running program has open, or one hard-linked elsewhere, with --remove-destination.",
			"Collecting build artifacts into a directory named up front with -t.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Copy a file to a new name:",
				Bash:        "cpx notes.txt notes.bak",
				PowerShell:  "cpx notes.txt notes.bak",
			},
			{
				Description: "Copy several files into a directory, reporting each one:",
				Bash:        "cpx -v a.txt b.txt backup/",
				PowerShell:  "cpx -v a.txt b.txt backup",
			},
			{
				Description: "Archive a directory tree, preserving modes, times and links:",
				Bash:        "cpx -a project/ /mnt/backup/project",
				PowerShell:  "cpx -a project D:\\backup\\project",
			},
			{
				Description: "Update a mirror, copying only files that are newer:",
				Bash:        "cpx -ru src/. mirror/",
				PowerShell:  "cpx -ru src\\. mirror",
			},
			{
				Description: "Copy into a directory named first, without overwriting anything:",
				Bash:        "cpx -n -t ~/Documents report.pdf slides.pdf",
				PowerShell:  "cpx -n -t $HOME\\Documents report.pdf slides.pdf",
			},
			{
				Description: "Ask before overwriting an existing copy:",
				Bash:        "cpx -i config.yml config.yml.orig",
				PowerShell:  "cpx -i config.yml config.yml.orig",
			},
			{
				Description: "Keep the mode and timestamps of a script while copying it:",
				Bash:        "cpx -p deploy.sh bin/",
				PowerShell:  "cpx -p deploy.sh bin",
			},
			{
				Description: "Preserve only the timestamps, or archive without carrying over ownership:",
				Bash:        "cpx --preserve=timestamps photo.jpg archive/ && cpx -a --no-preserve=ownership project/ copy/",
				PowerShell:  "cpx --preserve=timestamps photo.jpg archive; cpx -a --no-preserve=ownership project copy",
			},
			{
				Description: "Recreate the source paths under a backup directory:",
				Bash:        "cpx --parents src/app/main.go src/lib/util.go backup/",
				PowerShell:  "cpx --parents src/app/main.go src/lib/util.go backup",
			},
			{
				Description: "Make a new release tree of hard links that share the old one's data:",
				Bash:        "cpx -al releases/v1 releases/v2",
				PowerShell:  "cpx -al releases\\v1 releases\\v2",
			},
			{
				Description: "Create a symbolic link instead of a copy (use an absolute source):",
				Bash:        "cpx -s \"$PWD/config.yml\" config-link.yml",
				PowerShell:  "cpx -s \"$PWD\\config.yml\" config-link.yml",
			},
			{
				Description: "Copy a symbolic link as a link, rather than the file it points to:",
				Bash:        "cpx -P current.yml saved-link.yml",
				PowerShell:  "cpx -P current.yml saved-link.yml",
			},
			{
				Description: "Copy a tree, replacing every symbolic link with the file it points to:",
				Bash:        "cpx -rL vendor/ vendor-flat/",
				PowerShell:  "cpx -rL vendor vendor-flat",
			},
			{
				Description: "Copy through a symlinked directory named on the command line only:",
				Bash:        "cpx -rH latest snapshot",
				PowerShell:  "cpx -rH latest snapshot",
			},
			{
				Description: "Copy a template's contents into an existing directory, not inside it:",
				Bash:        "cpx -rT template/ newproject/",
				PowerShell:  "cpx -rT template newproject",
			},
			{
				Description: "Add missing default files without overwriting customized ones:",
				Bash:        "cpx -r --update=none defaults/. config/",
				PowerShell:  "cpx -r --update=none defaults\\. config",
			},
			{
				Description: "Replace a read-only destination file:",
				Bash:        "cpx -f new.conf readonly.conf",
				PowerShell:  "cpx -f new.conf readonly.conf",
			},
			{
				Description: "Back up a tree without crossing into other mounted file systems:",
				Bash:        "cpx -ax --strip-trailing-slashes data/ /mnt/backup/data",
				PowerShell:  "cpx -ax --strip-trailing-slashes data\\ D:\\backup\\data",
			},
			{
				Description: "Make a backup and continue only if the copy succeeded:",
				Bash:        "cpx config.yml config.yml.bak && echox \"backup saved\"",
				PowerShell:  "cpx config.yml config.yml.bak; if ($LASTEXITCODE -eq 0) { echox \"backup saved\" }",
			},
			{
				Description: "Copy every .conf file into a fresh directory with the gnux shell:",
				Bash:        "gnux shell -c 'mkdirx -p confbak && cpx -v *.conf confbak/'",
				PowerShell:  "gnux shell -c 'mkdirx -p confbak && cpx -v *.conf confbak/'",
			},
			{
				Description: "Replace a symbolic link with a real copy instead of writing through it:",
				Bash:        "cpx --remove-destination config/base.conf current.conf",
				PowerShell:  "cpx --remove-destination config\\base.conf current.conf",
			},
		},
	}
}

// Register installs the cpx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("archive", "a", false, "same as -dR --preserve=all")
	fs.BoolP("no-dereference-links", "d", false, "same as --no-dereference --preserve=links")
	fs.BoolP("force", "f", false, "if an existing destination file cannot be opened, remove it and try again")
	fs.BoolP("interactive", "i", false, "prompt before overwrite")
	fs.BoolP("dereference-args", "H", false, "follow command-line symbolic links in SOURCE")
	fs.BoolP("link", "l", false, "hard link files instead of copying")
	fs.BoolP("dereference", "L", false, "always follow symbolic links in SOURCE")
	fs.BoolP("no-clobber", "n", false, "do not overwrite an existing file")
	fs.BoolP("no-dereference", "P", false, "never follow symbolic links in SOURCE")
	fs.BoolP("preserve-default", "p", false, "same as --preserve="+defaultPreserve)
	fs.String("preserve", "", "preserve the listed attributes: mode, ownership, timestamps, links, all")
	fs.Lookup("preserve").NoOptDefVal = defaultPreserve
	fs.String("no-preserve", "", "don't preserve the listed attributes")
	fs.Bool("parents", false, "use the full source file name under DIRECTORY")
	fs.BoolP("recursive", "r", false, "copy directories recursively")
	fs.Bool("remove-destination", false, "remove each existing destination file before attempting to open it (contrast with --force)")
	// -R is the spelling GNU documents first; it is hidden so --help lists
	// the option once.
	fs.BoolP("recursive-upper", "R", false, "equivalent to -r")
	_ = fs.MarkHidden("recursive-upper")
	fs.BoolP("symbolic-link", "s", false, "make symbolic links instead of copying")
	fs.Bool("strip-trailing-slashes", false, "remove any trailing slashes from each SOURCE argument")
	fs.StringP("target-directory", "t", "", "copy all SOURCE arguments into DIRECTORY")
	fs.BoolP("no-target-directory", "T", false, "treat DEST as a normal file")
	fs.StringP("update", "u", "", "control which existing files are updated: all, none, or older (the default for -u)")
	fs.Lookup("update").NoOptDefVal = "older"
	fs.BoolP("verbose", "v", false, "explain what is being done")
	fs.BoolP("one-file-system", "x", false, "stay on this file system")

	// Long names GNU does not have: accepted, never an abbreviation.
	tool.OwnNames(fs, "dereference-args", "no-dereference-links", "preserve-default")
}

// buildOptions folds the parsed flags into cp.Options. GNU cp resolves
// conflicting options by position (the last one wins), which pflag does not
// expose, so conflicts resolve toward the most cautious choice instead:
// -n beats -i, and for symbolic links an explicit -P/-d beats -H beats -L
// (each of which beats the -P implied by -a).
func buildOptions(fs *pflag.FlagSet) (cp.Options, error) {
	flag := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}

	archive, noDerefLinks := flag("archive"), flag("no-dereference-links")
	o := cp.Options{
		Recursive:            flag("recursive") || flag("recursive-upper") || archive,
		Force:                flag("force"),
		Interactive:          flag("interactive"),
		NoClobber:            flag("no-clobber"),
		Verbose:              flag("verbose"),
		Link:                 flag("link"),
		SymbolicLink:         flag("symbolic-link"),
		NoTargetDirectory:    flag("no-target-directory"),
		OneFileSystem:        flag("one-file-system"),
		Parents:              flag("parents"),
		StripTrailingSlashes: flag("strip-trailing-slashes"),
		RemoveDestination:    flag("remove-destination"),
	}
	o.TargetDirectory, _ = fs.GetString("target-directory")

	switch {
	case flag("no-dereference") || noDerefLinks:
		o.Dereference = cp.DerefNever
	case flag("dereference-args"):
		o.Dereference = cp.DerefCommandLine
	case flag("dereference"):
		o.Dereference = cp.DerefAlways
	case archive:
		o.Dereference = cp.DerefNever
	}

	if archive {
		o.Preserve = cp.PreserveAll()
	}
	if noDerefLinks {
		o.Preserve.Links = true
	}
	list, _ := fs.GetString("preserve")
	if flag("preserve-default") {
		list += "," + defaultPreserve
	}
	if err := setPreserve(&o.Preserve, list, true, "--preserve"); err != nil {
		return cp.Options{}, err
	}
	list, _ = fs.GetString("no-preserve")
	if err := setPreserve(&o.Preserve, list, false, "--no-preserve"); err != nil {
		return cp.Options{}, err
	}

	if fs.Changed("update") {
		when, _ := fs.GetString("update")
		switch when {
		case "older":
			o.Update = true
		case "none":
			o.NoClobber = true
		case "all":
		default:
			return cp.Options{}, core.Usagef("invalid argument '%s' for '--update'\nValid arguments are:\n  - 'all'\n  - 'none'\n  - 'older'", when)
		}
	}
	if o.NoClobber {
		o.Interactive = false
	}
	return o, nil
}

// setPreserve turns each attribute named in the comma-separated list on or
// off in p. The SELinux context and extended attributes are accepted for
// compatibility but have no portable equivalent, so they are ignored.
func setPreserve(p *cp.Preserve, list string, on bool, option string) error {
	for _, attr := range strings.Split(list, ",") {
		switch attr {
		case "":
		case "mode":
			p.Mode = on
		case "ownership":
			p.Ownership = on
		case "timestamps":
			p.Timestamps = on
		case "links":
			p.Links = on
		case "context", "xattr":
		case "all":
			*p = cp.Preserve{Mode: on, Ownership: on, Timestamps: on, Links: on}
		default:
			return core.Usagef("invalid argument '%s' for '%s'\nValid arguments are:\n  - 'mode'\n  - 'ownership'\n  - 'timestamps'\n  - 'links'\n  - 'context'\n  - 'xattr'\n  - 'all'", attr, option)
		}
	}
	return nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return cp.New(opts, s).Run(args)
}
