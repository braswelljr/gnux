// Package chgrpx wires the chgrpx command (GNU chgrp) to its engine.
package chgrpx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/chgrp"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the chgrpx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "chgrp",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for chgrpx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    chgrp.Name,
		SeeAlso: []string{"chownx", "chmodx", "idx", "lsx"},
		Use:     chgrp.Name + " [OPTION]... GROUP FILE...",
		Short:   "Change group ownership.",
		Long: "chgrpx changes the group of each FILE to GROUP, a group name or numeric ID. An " +
			"empty GROUP changes nothing but still checks that each FILE exists. " +
			"--reference=RFILE takes the group from RFILE instead, and then every operand is " +
			"a FILE. A symbolic link given as a FILE is followed, and its referent changed, " +
			"unless -h is given; when both -h and --dereference are given, -h wins. -R " +
			"changes directory trees, following no symbolic link by default (-P); -H follows " +
			"the ones given on the command line and -L every one, and when several are given " +
			"the most cautious wins (-P, then -H, then -L). -c reports each change, -v every " +
			"file, and -f suppresses most error messages. Unlike GNU chgrp, which leaves the " +
			"safeguard off unless --preserve-root is given, chgrpx refuses to work " +
			"recursively on '/' unless --no-preserve-root is given; when both are given, " +
			"--preserve-root wins. On Windows, which has no Unix groups, every change fails " +
			"with 'Operation not supported'.",
		Permissions: "The owner of a file may change its group to any group they belong to; " +
			"any other change requires root (or CAP_CHOWN). Search permission is needed on " +
			"the directories leading to each FILE, and with -R read and search permission on " +
			"every directory in the tree.",
		UseCases: []string{
			"Sharing a project directory with a team's group.",
			"Giving a web server's group read access to a site with -R.",
			"Fixing group ownership after extracting an archive.",
			"Copying the group of one file onto others with --reference.",
			"Checking which files would change with -c on a test run.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Change the group of a file:",
				Bash:        "chgrpx staff report.txt",
				PowerShell:  "chgrpx staff report.txt",
			},
			{
				Description: "Use a numeric group ID:",
				Bash:        "chgrpx 1001 report.txt",
				PowerShell:  "chgrpx 1001 report.txt",
			},
			{
				Description: "Change a whole directory tree:",
				Bash:        "chgrpx -R developers project/",
				PowerShell:  "chgrpx -R developers project\\",
			},
			{
				Description: "Follow symbolic links met while recursing:",
				Bash:        "chgrpx -R -L developers project/",
				PowerShell:  "chgrpx -R -L developers project\\",
			},
			{
				Description: "Change a symbolic link itself:",
				Bash:        "chgrpx -h staff link",
				PowerShell:  "chgrpx -h staff link",
			},
			{
				Description: "Copy the group of another file:",
				Bash:        "chgrpx --reference=template.conf new.conf",
				PowerShell:  "chgrpx --reference=template.conf new.conf",
			},
			{
				Description: "Report only the files that changed:",
				Bash:        "chgrpx -c -R staff shared/",
				PowerShell:  "chgrpx -c -R staff shared\\",
			},
			{
				Description: "Report every file processed:",
				Bash:        "chgrpx -v staff a.txt b.txt",
				PowerShell:  "chgrpx -v staff a.txt b.txt",
			},
			{
				Description: "Suppress most error messages:",
				Bash:        "chgrpx -f staff maybe-missing.txt",
				PowerShell:  "chgrpx -f staff maybe-missing.txt",
			},
			{
				Description: "Give a web server's group the site files:",
				Bash:        "sudo chgrpx -R www-data /var/www/site",
				PowerShell:  "chgrpx -R www-data site",
			},
			{
				Description: "Change groups in the portable gnux shell:",
				Bash:        "gnux shell -c 'chgrpx -R staff shared && lsx -l shared'",
				PowerShell:  "gnux shell -c 'chgrpx -R staff shared && lsx -l shared'",
			},
		},
	}
}

// Register installs the chgrpx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("changes", "c", false, "like verbose but report only when a change is made")
	fs.BoolP("silent", "f", false, "suppress most error messages")
	fs.Bool("quiet", false, "same as --silent")
	fs.BoolP("verbose", "v", false, "output a diagnostic for every file processed")
	fs.Bool("dereference", false, "affect the referent of each symbolic link (this is the default), rather than the symbolic link itself")
	fs.BoolP("no-dereference", "h", false, "affect symbolic links instead of any referenced file")
	fs.Bool("no-preserve-root", false, "do not treat '/' specially")
	fs.Bool("preserve-root", false, "fail to operate recursively on '/' (the default)")
	fs.String("reference", "", "use `RFILE`'s group rather than specifying a GROUP value")
	fs.BoolP("recursive", "R", false, "operate on files and directories recursively")
	fs.BoolP("dereference-args", "H", false, "with -R, follow symbolic links to directories given on the command line")
	fs.BoolP("dereference-all", "L", false, "with -R, traverse every symbolic link to a directory encountered")
	fs.BoolP("physical", "P", false, "with -R, do not traverse any symbolic links (default)")
}

// buildOptions folds the parsed flags into chgrp.Options.
func buildOptions(fs *pflag.FlagSet) chgrp.Options {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := chgrp.Options{
		Changes:        on("changes"),
		Verbose:        on("verbose"),
		Silent:         on("silent") || on("quiet"),
		Recursive:      on("recursive"),
		NoDereference:  on("no-dereference"),
		Dereference:    on("dereference") && !on("no-dereference"),
		NoPreserveRoot: on("no-preserve-root") && !on("preserve-root"),
	}
	o.Reference, _ = fs.GetString("reference")
	// The most cautious traversal wins, whatever the order.
	switch {
	case on("physical"):
		o.Traversal = chgrp.TraversePhysical
	case on("dereference-args"):
		o.Traversal = chgrp.TraverseCommandLine
	case on("dereference-all"):
		o.Traversal = chgrp.TraverseLogical
	}
	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return chgrp.New(buildOptions(fs), s).Run(args)
}
