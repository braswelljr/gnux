// Package readlinkx wires the readlinkx command (GNU readlink) to its engine.
package readlinkx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/readlink"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the readlinkx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "readlink",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for readlinkx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    readlink.Name,
		SeeAlso: []string{"realpathx", "lnx", "lsx"},
		Use:     readlink.Name + " [OPTION]... FILE...",
		Short:   "Print resolved symbolic links or canonical file names.",
		Long: "readlinkx prints the value of each FILE, which must be a symbolic link: the " +
			"target exactly as it was stored, not resolved. With -f, -e or -m it prints the " +
			"canonical name instead: absolute, with every symbolic link and every '.' and '..' " +
			"component resolved. -f requires every component but the last to exist, -e requires " +
			"all of them, and -m requires none. As in GNU readlink, failures are silent by " +
			"default (-q and -s say so explicitly) and only the exit status reports them; -v " +
			"prints a diagnostic for each. -n drops the trailing newline, and is ignored with a " +
			"warning when there is more than one FILE; -z ends each name with NUL. GNU " +
			"readlink lets the last of -f, -e and -m win; readlinkx cannot see option order, " +
			"so it picks the most demanding (-e, then -f, then -m), and -v wins over -q and -s.",
		Permissions: "Needs search permission on every directory leading to FILE, and on " +
			"every directory a link leads through. Reading a link's value needs no permission " +
			"on the link itself. Nothing is ever modified.",
		UseCases: []string{
			"Finding where a symbolic link points, exactly as it was created.",
			"Getting the absolute, link-free path of a script or binary with -f.",
			"Checking in a script that a path fully exists after resolution with -e.",
			"Normalizing a path that does not exist yet with -m.",
			"Detecting whether a file is a symbolic link from the exit status.",
			"Comparing two paths for identity after resolving their links.",
			"Producing NUL-separated output for names with newlines with -z.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print where a link points:",
				Bash:        "readlinkx current",
				PowerShell:  "readlinkx current",
			},
			{
				Description: "Print the canonical, absolute name of a file:",
				Bash:        "readlinkx -f ./bin/../bin/tool",
				PowerShell:  "readlinkx -f ./bin/../bin/tool",
			},
			{
				Description: "Require the whole path to exist:",
				Bash:        "readlinkx -e config/app.yaml",
				PowerShell:  "readlinkx -e config/app.yaml",
			},
			{
				Description: "Normalize a path that may not exist yet:",
				Bash:        "readlinkx -m build/../dist/out",
				PowerShell:  "readlinkx -m build/../dist/out",
			},
			{
				Description: "Explain why a name cannot be resolved:",
				Bash:        "readlinkx -v -e missing/file",
				PowerShell:  "readlinkx -v -e missing/file",
			},
			{
				Description: "Test whether a file is a symbolic link:",
				Bash:        "readlinkx current > /dev/null && echox 'a link'",
				PowerShell:  "readlinkx current > $null; if ($LASTEXITCODE -eq 0) { 'a link' }",
			},
			{
				Description: "Capture a resolved path without a trailing newline:",
				Bash:        "printfx 'cd %s\\n' \"$(readlinkx -n -f link)\"",
				PowerShell:  "$p = readlinkx -n -f link; \"cd $p\"",
			},
			{
				Description: "Resolve several links at once, NUL-separated:",
				Bash:        "readlinkx -z -f a b c | trx '\\0' '\\n'",
				PowerShell:  "readlinkx -z -f a b c | trx '\\0' '\\n'",
			},
			{
				Description: "Find the directory a script really lives in:",
				Bash:        "dirnamex \"$(readlinkx -f \"$0\")\"",
				PowerShell:  "dirnamex (readlinkx -f $PSCommandPath)",
			},
			{
				Description: "Check that two names reach the same file:",
				Bash:        "[ \"$(readlinkx -f a)\" = \"$(readlinkx -f b)\" ] && echox same",
				PowerShell:  "if ((readlinkx -f a) -eq (readlinkx -f b)) { 'same' }",
			},
			{
				Description: "Resolve a link in the portable gnux shell:",
				Bash:        "gnux shell -c 'readlinkx -f current'",
				PowerShell:  "gnux shell -c 'readlinkx -f current'",
			},
		},
	}
}

// Register installs the readlinkx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("canonicalize", "f", false,
		"canonicalize by following every symlink in every component recursively; all but the last component must exist")
	fs.BoolP("canonicalize-existing", "e", false,
		"canonicalize by following every symlink in every component recursively; all components must exist")
	fs.BoolP("canonicalize-missing", "m", false,
		"canonicalize by following every symlink in every component recursively, without requirements on existence")
	fs.BoolP("no-newline", "n", false, "do not output the trailing delimiter")
	fs.BoolP("quiet", "q", false, "suppress most error messages (on by default)")
	fs.BoolP("silent", "s", false, "suppress most error messages (on by default)")
	fs.BoolP("verbose", "v", false, "report error messages")
	fs.BoolP("zero", "z", false, "end each output line with NUL, not newline")
}

// buildOptions folds the parsed flags into readlink.Options. The most
// demanding canonicalization wins, and -v wins over -q and -s.
func buildOptions(fs *pflag.FlagSet) readlink.Options {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := readlink.Options{
		NoNewline: on("no-newline"),
		Verbose:   on("verbose"),
		Zero:      on("zero"),
	}
	switch {
	case on("canonicalize-existing"):
		o.Mode = readlink.ModeExisting
	case on("canonicalize"):
		o.Mode = readlink.ModeFollow
	case on("canonicalize-missing"):
		o.Mode = readlink.ModeMissing
	}
	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return readlink.New(buildOptions(fs), s).Run(args)
}
