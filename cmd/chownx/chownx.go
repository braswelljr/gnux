// Package chownx wires the chownx command (GNU chown) to its engine.
package chownx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/chown"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the chownx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "chown",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for chownx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    chown.Name,
		SeeAlso: []string{"chgrpx", "chmodx", "idx", "lsx", "statx"},
		Use:     chown.Name + " [OPTION]... [OWNER][:[GROUP]] FILE...",
		Short:   "Change file owner and group.",
		Long: "chownx changes the owner and/or group of each FILE. OWNER and GROUP are names " +
			"or numeric IDs: 'OWNER' changes the owner only, 'OWNER:GROUP' both, 'OWNER:' the " +
			"owner and the group to OWNER's login group, and ':GROUP' the group only; an " +
			"empty spec changes nothing but still checks that each FILE exists. " +
			"--reference=RFILE takes the owner and group from RFILE instead, and then every " +
			"operand is a FILE. --from=CURRENT_OWNER:CURRENT_GROUP changes only the files " +
			"currently owned as stated (either part may be omitted). A symbolic link given as " +
			"a FILE is followed, and its referent changed, unless -h is given; when both -h " +
			"and --dereference are given, -h wins. -R changes directory trees, following no " +
			"symbolic link by default (-P); -H follows the ones given on the command line and " +
			"-L every one, and when several are given the most cautious wins (-P, then -H, " +
			"then -L). -c reports each change, -v every file, and -f suppresses most error " +
			"messages. Unlike GNU chown, which leaves the safeguard off unless --preserve-root " +
			"is given, chownx refuses to work recursively on '/' unless --no-preserve-root is " +
			"given; when both are given, --preserve-root wins. On Windows, which has no Unix " +
			"owners, every change fails with 'Operation not supported'.",
		Permissions: "Changing a file's owner requires root (or CAP_CHOWN). Without it, the " +
			"owner of a file may change its group, but only to a group they belong to. " +
			"Search permission is needed on the directories leading to each FILE, and with -R " +
			"read and search permission on every directory in the tree.",
		UseCases: []string{
			"Handing files created as root back to the user who will maintain them.",
			"Giving a service account ownership of its data directory with -R.",
			"Fixing ownership after copying files between machines or containers.",
			"Changing only the files still owned by a departed user with --from.",
			"Copying the ownership of one file onto others with --reference.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Give a file to another user:",
				Bash:        "sudo chownx alice report.txt",
				PowerShell:  "chownx alice report.txt",
			},
			{
				Description: "Change the owner and group:",
				Bash:        "sudo chownx alice:staff report.txt",
				PowerShell:  "chownx alice:staff report.txt",
			},
			{
				Description: "Change only the group:",
				Bash:        "chownx :staff report.txt",
				PowerShell:  "chownx :staff report.txt",
			},
			{
				Description: "Use the owner's login group:",
				Bash:        "sudo chownx alice: report.txt",
				PowerShell:  "chownx alice: report.txt",
			},
			{
				Description: "Use numeric IDs:",
				Bash:        "sudo chownx 1000:1000 data.db",
				PowerShell:  "chownx 1000:1000 data.db",
			},
			{
				Description: "Change a whole directory tree:",
				Bash:        "sudo chownx -R www-data:www-data /var/www/site",
				PowerShell:  "chownx -R www-data:www-data site",
			},
			{
				Description: "Change a symbolic link itself:",
				Bash:        "sudo chownx -h alice link",
				PowerShell:  "chownx -h alice link",
			},
			{
				Description: "Change only the files a departed user still owns:",
				Bash:        "sudo chownx -R --from=bob alice /srv/shared",
				PowerShell:  "chownx -R --from=bob alice shared",
			},
			{
				Description: "Copy the ownership of another file:",
				Bash:        "sudo chownx --reference=template.conf new.conf",
				PowerShell:  "chownx --reference=template.conf new.conf",
			},
			{
				Description: "Report only the files that changed:",
				Bash:        "sudo chownx -c -R alice project/",
				PowerShell:  "chownx -c -R alice project\\",
			},
			{
				Description: "Change ownership in the portable gnux shell:",
				Bash:        "sudo gnux shell -c 'chownx -R alice:staff project && lsx -l project'",
				PowerShell:  "gnux shell -c 'chownx -R alice:staff project && lsx -l project'",
			},
		},
	}
}

// Register installs the chownx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("changes", "c", false, "like verbose but report only when a change is made")
	fs.BoolP("silent", "f", false, "suppress most error messages")
	fs.Bool("quiet", false, "same as --silent")
	fs.BoolP("verbose", "v", false, "output a diagnostic for every file processed")
	fs.Bool("dereference", false, "affect the referent of each symbolic link (this is the default), rather than the symbolic link itself")
	fs.BoolP("no-dereference", "h", false, "affect symbolic links instead of any referenced file")
	fs.String("from", "", "change the ownership of each file only if its current owner and/or group match those specified here; either may be omitted (`CURRENT_OWNER:CURRENT_GROUP`)")
	fs.Bool("no-preserve-root", false, "do not treat '/' specially")
	fs.Bool("preserve-root", false, "fail to operate recursively on '/' (the default)")
	fs.String("reference", "", "use `RFILE`'s ownership rather than specifying values")
	fs.BoolP("recursive", "R", false, "operate on files and directories recursively")
	fs.BoolP("dereference-args", "H", false, "with -R, follow symbolic links to directories given on the command line")
	fs.BoolP("dereference-all", "L", false, "with -R, traverse every symbolic link to a directory encountered")
	fs.BoolP("physical", "P", false, "with -R, do not traverse any symbolic links (default)")
}

// buildOptions folds the parsed flags into chown.Options.
func buildOptions(fs *pflag.FlagSet) chown.Options {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := chown.Options{
		Changes:        on("changes"),
		Verbose:        on("verbose"),
		Silent:         on("silent") || on("quiet"),
		Recursive:      on("recursive"),
		NoDereference:  on("no-dereference"),
		Dereference:    on("dereference") && !on("no-dereference"),
		NoPreserveRoot: on("no-preserve-root") && !on("preserve-root"),
	}
	o.Reference, _ = fs.GetString("reference")
	o.From, _ = fs.GetString("from")
	// The most cautious traversal wins, whatever the order.
	switch {
	case on("physical"):
		o.Traversal = chown.TraversePhysical
	case on("dereference-args"):
		o.Traversal = chown.TraverseCommandLine
	case on("dereference-all"):
		o.Traversal = chown.TraverseLogical
	}
	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return chown.New(buildOptions(fs), s).Run(args)
}
