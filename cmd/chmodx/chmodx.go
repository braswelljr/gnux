// Package chmodx wires the chmodx command (GNU chmod) to its engine.
package chmodx

import (
	"strings"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/chmod"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the chmodx tool. A mode may be written as an option
// ("chmodx -w FILE"), which a flag parser would reject, so the arguments are
// parsed here: anything shaped like a mode is set aside first, as GNU's
// getopt string does, and the rest go through the flags.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:    Meta(),
		GNU:     "chmod",
		Flags:   Register,
		Run:     run,
		RawArgs: true,
	}
}

// Meta returns the documentation metadata for chmodx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    chmod.Name,
		SeeAlso: []string{"chownx", "chgrpx", "lsx", "statx", "installx"},
		Use:     chmod.Name + " [OPTION]... MODE[,MODE]... FILE...",
		Short:   "Change file mode bits.",
		Long: "chmodx changes the permission bits of each FILE to MODE, which is either an " +
			"octal number (755, 0644, 4755) or a comma-separated list of symbolic changes. " +
			"Each change names who (u, g, o or a for user, group, others or all), an " +
			"operator (+ adds, - removes, = sets exactly) and permissions: r, w, x, X " +
			"(execute only on directories and on files already executable by someone), s " +
			"(setuid/setgid) and t (sticky), or u, g or o to copy another class's bits. A symbolic change without a " +
			"who honors the umask. --reference=RFILE copies RFILE's mode instead, and then " +
			"every operand is a FILE. -R changes directory trees; -H (the default) follows " +
			"symbolic links given on the command line, -L follows every symbolic link and -P " +
			"none, and when several are given the most cautious wins (-P, then -H, then -L). " +
			"Symbolic links met while recursing are never changed. -c reports each change, -v " +
			"every file, and -f suppresses most error messages. A mode may also be written as " +
			"an option, as in 'chmodx -w FILE'; several such options combine as if joined " +
			"with commas, and then, as in GNU, a file left with bits the mode would have " +
			"cleared but for the umask is reported ('new permissions are ..., not ...') and " +
			"counts as a failure. Unlike GNU chmod, which leaves the safeguard off unless " +
			"--preserve-root is given, chmodx refuses to work recursively on '/' unless " +
			"--no-preserve-root is given; when both are given, --preserve-root wins. GNU " +
			"chmod 9.6's -h and --dereference are not implemented. On Windows only the " +
			"owner's write bit maps onto the file system (the read-only attribute), so the " +
			"other bits are accepted but have no effect.",
		Permissions: "Changing a file's mode requires owning it, or running as root (or with " +
			"CAP_FOWNER). Search permission is needed on the directories leading to each " +
			"FILE, and with -R read and search permission on every directory in the tree.",
		UseCases: []string{
			"Making a script executable with +x.",
			"Locking down a private key or config file with 600.",
			"Giving a group write access to a shared directory tree with -R g+w.",
			"Making directories, but not files, searchable with -R a+X.",
			"Copying the permissions of one file onto others with --reference.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Make a script executable:",
				Bash:        "chmodx +x deploy.sh",
				PowerShell:  "chmodx +x deploy.sh",
			},
			{
				Description: "Set exact permissions with an octal mode:",
				Bash:        "chmodx 600 ~/.ssh/id_ed25519",
				PowerShell:  "chmodx 600 $HOME\\.ssh\\id_ed25519",
			},
			{
				Description: "Remove write permission for group and others:",
				Bash:        "chmodx go-w shared.txt",
				PowerShell:  "chmodx go-w shared.txt",
			},
			{
				Description: "Combine several changes:",
				Bash:        "chmodx u=rwx,g=rx,o= app",
				PowerShell:  "chmodx u=rwx,g=rx,o= app",
			},
			{
				Description: "Give a group write access to a whole tree:",
				Bash:        "chmodx -R g+w project/",
				PowerShell:  "chmodx -R g+w project\\",
			},
			{
				Description: "Make directories searchable without making files executable:",
				Bash:        "chmodx -R a+X public/",
				PowerShell:  "chmodx -R a+X public\\",
			},
			{
				Description: "Report only the files that changed:",
				Bash:        "chmodx -c 644 *.txt",
				PowerShell:  "chmodx -c 644 (Get-ChildItem *.txt -Name)",
			},
			{
				Description: "Copy the mode of one file onto another:",
				Bash:        "chmodx --reference=template.sh new.sh",
				PowerShell:  "chmodx --reference=template.sh new.sh",
			},
			{
				Description: "Write the mode as an option, as old scripts do:",
				Bash:        "chmodx -w notes.txt",
				PowerShell:  "chmodx -w notes.txt",
			},
			{
				Description: "Set the sticky bit on a shared directory:",
				Bash:        "chmodx 1777 /srv/drop",
				PowerShell:  "chmodx 1777 C:\\srv\\drop",
			},
			{
				Description: "Show every file processed:",
				Bash:        "chmodx -v -R u+rw docs/",
				PowerShell:  "chmodx -v -R u+rw docs\\",
			},
			{
				Description: "Change modes in the portable gnux shell:",
				Bash:        "gnux shell -c 'chmodx +x bin/* && lsx -l bin'",
				PowerShell:  "gnux shell -c 'chmodx +x bin/* && lsx -l bin'",
			},
		},
	}
}

// Register installs the chmodx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("changes", "c", false, "like verbose but report only when a change is made")
	fs.BoolP("silent", "f", false, "suppress most error messages")
	fs.Bool("quiet", false, "same as --silent")
	fs.BoolP("verbose", "v", false, "output a diagnostic for every file processed")
	fs.Bool("no-preserve-root", false, "do not treat '/' specially")
	fs.Bool("preserve-root", false, "fail to operate recursively on '/' (the default)")
	fs.String("reference", "", "use `RFILE`'s mode instead of specifying MODE values")
	fs.BoolP("recursive", "R", false, "change files and directories recursively")
	fs.BoolP("dereference-args", "H", false, "with -R, follow symbolic links given on the command line (the default)")
	fs.BoolP("dereference-all", "L", false, "with -R, follow every symbolic link to a directory")
	fs.BoolP("no-dereference", "P", false, "with -R, do not follow any symbolic links")
}

// parsed is a command line split into its parts.
type parsed struct {
	opts     chmod.Options
	operands []string
	help     bool
	version  bool
}

// isMode reports whether a is a mode written as an option: '-' followed by
// characters that can only belong to a mode ("-w", "-rwx", "-u+x,g-w").
func isMode(a string) bool {
	return len(a) >= 2 && a[0] == '-' && a[1] != '-' &&
		strings.Trim(a[1:], "rwxXstugoa,+-=01234567") == ""
}

// parseArgs sets aside the modes written as options, parses the flags on fs
// and folds them into chmod.Options. The operands come back with the mode
// first, ready for the engine.
func parseArgs(fs *pflag.FlagSet, args []string) (parsed, error) {
	var p parsed
	var modes, rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			rest = append(rest, args[i:]...)
			i = len(args)
		case isMode(a):
			modes = append(modes, a)
		default:
			rest = append(rest, a)
			if a == "--reference" && i+1 < len(args) {
				i++
				rest = append(rest, args[i])
			}
		}
	}
	if err := fs.Parse(rest); err != nil {
		return p, core.Usagef("%s", err)
	}
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	p.help, p.version = on("help"), on("version")
	p.opts = chmod.Options{
		Changes:        on("changes"),
		Silent:         on("silent") || on("quiet"),
		Verbose:        on("verbose"),
		Recursive:      on("recursive"),
		NoPreserveRoot: on("no-preserve-root") && !on("preserve-root"),
	}
	p.opts.Reference, _ = fs.GetString("reference")
	// The most cautious traversal wins, whatever the order.
	switch {
	case on("no-dereference"):
		p.opts.Traversal = chmod.TraversePhysical
	case on("dereference-args"):
		p.opts.Traversal = chmod.TraverseCommandLine
	case on("dereference-all"):
		p.opts.Traversal = chmod.TraverseLogical
	}
	p.operands = fs.Args()
	if len(modes) > 0 {
		if p.opts.Reference != "" {
			return p, core.Usagef("cannot combine mode and --reference options")
		}
		p.opts.DiagnoseSurprises = true
		p.operands = append([]string{strings.Join(modes, ",")}, p.operands...)
	}
	return p, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	p, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	switch {
	case p.help:
		Spec().WriteHelp(s.Out, fs, tool.StyleFor(s.Out))
		return nil
	case p.version:
		tool.PrintVersion(s.Out, chmod.Name)
		return nil
	}
	return chmod.New(p.opts, s).Run(p.operands)
}
