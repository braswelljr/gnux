// Package lnx wires the lnx command (GNU ln) to its link engine.
package lnx

import (
	"os"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/ln"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the lnx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "ln",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for lnx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    ln.Name,
		SeeAlso: []string{"cpx", "realpathx", "lsx", "rmx"},
		Use:     ln.Name + " [OPTION]... TARGET... [LINK_NAME|DIRECTORY]",
		Short:   "Make hard or symbolic links between files.",
		Long: "lnx creates a link to TARGET. It accepts ln's four forms: TARGET LINK_NAME; " +
			"TARGET alone (a link of the same name in the current directory); TARGET... " +
			"DIRECTORY; and -t DIRECTORY TARGET.... Links are hard links by default and " +
			"symbolic with -s; a symbolic link stores TARGET exactly as written unless -r " +
			"rewrites it relative to the link's own location. An existing destination is an " +
			"error unless -f (replace it) or -i (ask first) is given, and the replacement is " +
			"atomic, so the old entry survives a failure. -n treats a LINK_NAME that is a " +
			"symbolic link to a directory as the entry to replace rather than a directory to " +
			"link into, and -T always treats LINK_NAME as a plain file. " +
			"--backup[=CONTROL] (or -b) first renames each existing destination to a backup: " +
			"CONTROL simple (or never) appends a suffix, numbered (t) appends .~N~ with N one " +
			"past the highest already there, existing (nil, the default) numbers the backup " +
			"only when numbered backups already exist, and none (off) makes none; without a " +
			"CONTROL the VERSION_CONTROL variable chooses. The suffix is ~, or SUFFIX with -S " +
			"(which asks for backups by itself), or the SIMPLE_BACKUP_SUFFIX variable. A backup " +
			"replaces the destination as -f would, and -v reports it as 'BACKUP' ~ 'LINK' => " +
			"'TARGET'. For hard links, -L " +
			"links to what a symbolic link TARGET points at and -P (the default) to the link " +
			"itself. GNU ln lets the last of conflicting options win; lnx cannot see option " +
			"order, so it prefers the most cautious (-i over -f) and -L over -P.",
		Permissions: "Creating a link needs write and search permission on the directory that " +
			"will hold it. A symbolic link needs no access to TARGET, which may not even exist; " +
			"a hard link requires TARGET to exist on the same filesystem and cannot point at a " +
			"directory. On Windows, creating symbolic links requires Developer Mode or an " +
			"elevated prompt; a link's target is stored with backslashes, the only separator a " +
			"Windows link follows; and replacing a link to a directory (-sfn) removes the old " +
			"link before renaming the new one in, since Windows cannot rename over it, so it is " +
			"not atomic there.",
		UseCases: []string{
			"Pointing a stable name (current, latest) at a versioned file or directory.",
			"Atomically switching a symlink to a new release with -sfn.",
			"Creating relocatable links inside a tree with -sr.",
			"Giving one file a second name without duplicating its data.",
			"Exposing scripts on PATH by linking them into ~/bin with -t.",
			"Installing the gnux per-tool commands as links to one binary.",
			"Snapshotting a file with a hard link before it is replaced.",
			"Linking dotfiles from a repository into the home directory.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Create a symbolic link:",
				Bash:        "lnx -s /opt/app/releases/v2 current",
				PowerShell:  "lnx -s C:\\app\\releases\\v2 current",
			},
			{
				Description: "Repoint an existing symlink to a directory, in one atomic step:",
				Bash:        "lnx -sfn releases/v3 current",
				PowerShell:  "lnx -sfn releases\\v3 current",
			},
			{
				Description: "Create a hard link, reporting what was done:",
				Bash:        "lnx -v data.db data.db.snapshot",
				PowerShell:  "lnx -v data.db data.db.snapshot",
			},
			{
				Description: "Link several files into a directory with relative symlinks:",
				Bash:        "lnx -sr -t ~/bin tools/build tools/deploy",
				PowerShell:  "lnx -sr -t $HOME\\bin tools\\build tools\\deploy",
			},
			{
				Description: "Create a link with the same name in the current directory:",
				Bash:        "lnx -s ../shared/config.yml",
				PowerShell:  "lnx -s ..\\shared\\config.yml",
			},
			{
				Description: "Link several targets into an existing directory:",
				Bash:        "lnx -sv ../dotfiles/.vimrc ../dotfiles/.gitconfig home/",
				PowerShell:  "lnx -sv ..\\dotfiles\\.vimrc ..\\dotfiles\\.gitconfig home",
			},
			{
				Description: "Ask before replacing an existing link:",
				Bash:        "lnx -si settings.prod.json settings.json",
				PowerShell:  "lnx -si settings.prod.json settings.json",
			},
			{
				Description: "Replace an existing file with a link:",
				Bash:        "lnx -sf app-v2.conf app.conf",
				PowerShell:  "lnx -sf app-v2.conf app.conf",
			},
			{
				Description: "Treat the link name as a plain file, never as a directory to link into:",
				Bash:        "lnx -sT releases/v3 current-release",
				PowerShell:  "lnx -sT releases\\v3 current-release",
			},
			{
				Description: "Hard-link the file a symlink points to, instead of the symlink itself:",
				Bash:        "lnx -L current.log saved.log",
				PowerShell:  "lnx -L current.log saved.log",
			},
			{
				Description: "Hard-link the symlink itself (the default, spelled out):",
				Bash:        "lnx -P current.log current-link.log",
				PowerShell:  "lnx -P current.log current-link.log",
			},
			{
				Description: "Create the per-tool commands as links to the gnux binary:",
				Bash:        "for t in $(gnux --list); do lnx -sf gnux \"$t\"; done",
				PowerShell:  "gnux --list | ForEach-Object { lnx -sf gnux.exe \"$_.exe\" }",
			},
			{
				Description: "Report whether linking succeeded:",
				Bash:        "lnx -s target.txt link.txt && echox linked || echox \"link exists already\"",
				PowerShell:  "lnx -s target.txt link.txt; if ($LASTEXITCODE -eq 0) { echox linked } else { echox \"link exists already\" }",
			},
			{
				Description: "Create a release directory and point current at it with the gnux shell:",
				Bash:        "gnux shell -c 'mkdirx -p releases/v4 && lnx -sfnv releases/v4 current'",
				PowerShell:  "gnux shell -c 'mkdirx -p releases/v4 && lnx -sfnv releases/v4 current'",
			},
			{
				Description: "Repoint a symbolic link, keeping the old one as current~ in case you need to roll back:",
				Bash:        "lnx -sbv releases/v5 current",
				PowerShell:  "lnx -sbv releases\\v5 current",
			},
		},
	}
}

// Register installs the lnx flags on fs.
func Register(fs *pflag.FlagSet) {
	tool.RegisterBackup(fs)
	fs.BoolP("force", "f", false, "remove existing destination files")
	fs.BoolP("interactive", "i", false, "prompt whether to remove destinations")
	fs.BoolP("logical", "L", false, "dereference TARGETs that are symbolic links")
	fs.BoolP("no-dereference", "n", false, "treat LINK_NAME as a normal file if it is a symbolic link to a directory")
	fs.BoolP("physical", "P", false, "make hard links directly to symbolic links (default)")
	fs.BoolP("relative", "r", false, "with -s, create links relative to link location")
	fs.BoolP("symbolic", "s", false, "make symbolic links instead of hard links")
	fs.StringP("target-directory", "t", "", "specify the DIRECTORY in which to create the links")
	fs.BoolP("no-target-directory", "T", false, "treat LINK_NAME as a normal file always")
	fs.BoolP("verbose", "v", false, "print name of each linked file")
}

// buildOptions folds the parsed flags into ln.Options. GNU ln lets the last
// of conflicting options win, which pflag cannot reproduce because it does
// not expose option order; -i (the cautious one) beats -f, and -L beats -P.
func buildOptions(fs *pflag.FlagSet) (ln.Options, error) {
	flag := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}

	o := ln.Options{
		Symbolic:          flag("symbolic"),
		Force:             flag("force"),
		Interactive:       flag("interactive"),
		NoDereference:     flag("no-dereference"),
		Verbose:           flag("verbose"),
		Relative:          flag("relative"),
		NoTargetDirectory: flag("no-target-directory"),
		Logical:           flag("logical"),
	}
	o.TargetDirectory, _ = fs.GetString("target-directory")

	o.Physical = !o.Logical
	if o.Interactive {
		o.Force = false
	}
	backup, err := tool.BackupOptions(fs, os.Getenv)
	if err != nil {
		return ln.Options{}, err
	}
	o.Backup = backup
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return ln.New(opts, s).Run(args)
}
