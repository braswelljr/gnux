// Package mktempx wires the mktempx command (GNU mktemp) to its engine.
package mktempx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/mktemp"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// defaultDir is what --tmpdir means without a value: $TMPDIR, else the
// system temporary directory.
const defaultDir = "$TMPDIR"

// Spec describes the mktempx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "mktemp",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for mktempx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    mktemp.Name,
		SeeAlso: []string{"mkdirx", "touchx", "rmx"},
		Use:     mktemp.Name + " [OPTION]... [TEMPLATE]",
		Short:   "Create a temporary file or directory, safely, and print its name.",
		Long: "mktempx creates a new, empty file (or a directory with -d) whose name is " +
			"TEMPLATE with its run of trailing X's replaced by random letters and digits, and " +
			"prints the name. TEMPLATE must contain at least 3 consecutive X's at the end of " +
			"the name; whatever follows the last X is kept as a suffix, and --suffix adds one " +
			"explicitly. With no TEMPLATE, tmp.XXXXXXXXXX is used and the file is created in " +
			"the temporary directory. A relative TEMPLATE is resolved against the current " +
			"directory unless -p DIR or --tmpdir[=DIR] is given; then it goes under DIR, else " +
			"$TMPDIR, else the system temporary directory (/tmp on Unix, the user's temporary " +
			"folder on Windows), and must not be absolute. The file is created with mode 600 " +
			"and the directory with mode 700, and creation is atomic: an existing name is " +
			"never reused. -u only prints a name that did not exist at the time, which is " +
			"unsafe. -q suppresses the diagnostic when creation fails. -t is the deprecated " +
			"form: TEMPLATE is a single name component under $TMPDIR, else -p DIR, else the " +
			"temporary directory. -p and -t also answer to the gnux long names --dir and --in-tmpdir.",
		Permissions: "Needs write and search permission on the directory the name is " +
			"created in. The new file is readable and writable by its owner only (600), and " +
			"a new directory is accessible by its owner only (700), whatever the umask. On " +
			"Windows only the read-only attribute is representable, so the entry inherits the " +
			"folder's access control list.",
		UseCases: []string{
			"Getting a scratch file in a script that cannot collide with another run.",
			"Creating a private working directory with -d and removing it at the end.",
			"Staging output before moving it into place atomically with mvx.",
			"Keeping temporary files of a job together under a directory of your choice with -p.",
			"Giving a temporary file the extension a tool expects with --suffix.",
			"Writing portable scripts that do not hard-code /tmp, which does not exist on Windows.",
			"Generating a unique name for a lock or marker file.",
			"Reserving a name without creating it with -u, when another program must create it.",
		},
		Examples: []docmeta.Example{
			{Description: "Create a temporary file and print its name:", Bash: "mktempx", PowerShell: "mktempx"},
			{Description: "Create a temporary directory:", Bash: "mktempx -d", PowerShell: "mktempx -d"},
			{Description: "Use the name in a script and clean up afterwards:", Bash: "tmp=$(mktempx) && echox hello > \"$tmp\" && catx \"$tmp\" && rmx \"$tmp\"", PowerShell: "$tmp = mktempx; echox hello > $tmp; catx $tmp; rmx $tmp"},
			{Description: "Work in a private scratch directory:", Bash: "work=$(mktempx -d) && cpx -r src \"$work\" && rmx -r \"$work\"", PowerShell: "$work = mktempx -d; cpx -r src $work; rmx -r $work"},
			{Description: "Create a file from a template in the current directory:", Bash: "mktempx build.XXXXXX", PowerShell: "mktempx build.XXXXXX"},
			{Description: "Create it under the temporary directory instead:", Bash: "mktempx --tmpdir build.XXXXXX", PowerShell: "mktempx --tmpdir build.XXXXXX"},
			{Description: "Create it under a directory of your choice:", Bash: "mktempx -p out report.XXXX", PowerShell: "mktempx -p out report.XXXX"},
			{Description: "Give the file an extension:", Bash: "mktempx --suffix=.json data.XXXXXX", PowerShell: "mktempx --suffix=.json data.XXXXXX"},
			{Description: "Keep a suffix written after the X's:", Bash: "mktempx notes.XXXX.md", PowerShell: "mktempx notes.XXXX.md"},
			{Description: "Print an unused name without creating anything:", Bash: "mktempx -u", PowerShell: "mktempx -u"},
			{Description: "Fail quietly and handle it yourself:", Bash: "mktempx -q -p /nonexistent || echox 'no temp dir'", PowerShell: "mktempx -q -p C:\\nonexistent; if ($LASTEXITCODE -ne 0) { echox 'no temp dir' }"},
			{Description: "Use the deprecated -t form, relative to $TMPDIR:", Bash: "mktempx -t job.XXXXXX", PowerShell: "mktempx -t job.XXXXXX"},
			{Description: "Create a temporary file in the portable gnux shell:", Bash: "gnux shell -c 'mktempx -p . scratch.XXXX'", PowerShell: "gnux shell -c 'mktempx -p . scratch.XXXX'"},
		},
	}
}

// Register installs the mktempx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("directory", "d", false, "create a directory, not a file")
	fs.BoolP("dry-run", "u", false, "do not create anything; merely print a name (unsafe)")
	fs.BoolP("quiet", "q", false, "suppress diagnostics about file/dir-creation failure")
	fs.String("suffix", "", "append `SUFF` to TEMPLATE; SUFF must not contain a slash")
	fs.String("tmpdir", "", "interpret TEMPLATE relative to `DIR`; if DIR is not given, use $TMPDIR if set, else the temporary directory")
	fs.Lookup("tmpdir").NoOptDefVal = defaultDir
	fs.StringP("dir", "p", "", "use `DIR` as the directory, like --tmpdir=DIR")
	fs.BoolP("in-tmpdir", "t", false, "interpret TEMPLATE as a single file name component, relative to $TMPDIR, else DIR (-p), else the temporary directory (deprecated)")
}

// buildOptions folds the parsed flags into mktemp.Options. -p and --tmpdir
// are the same option; given both, an explicit DIR wins over the default.
func buildOptions(fs *pflag.FlagSet) mktemp.Options {
	var o mktemp.Options
	o.Directory, _ = fs.GetBool("directory")
	o.DryRun, _ = fs.GetBool("dry-run")
	o.Quiet, _ = fs.GetBool("quiet")
	o.Legacy, _ = fs.GetBool("in-tmpdir")
	if fs.Changed("suffix") {
		o.Suffix, _ = fs.GetString("suffix")
		o.SuffixSet = true
	}
	if fs.Changed("tmpdir") {
		o.UseTmpDir = true
		if dir, _ := fs.GetString("tmpdir"); dir != defaultDir {
			o.TmpDir = dir
		}
	}
	if fs.Changed("dir") {
		dir, _ := fs.GetString("dir")
		o.TmpDir = dir
		// -t uses -p DIR as a fallback only, without --tmpdir's rules.
		o.UseTmpDir = o.UseTmpDir || !o.Legacy
	}
	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return mktemp.New(buildOptions(fs), s).Run(args)
}
