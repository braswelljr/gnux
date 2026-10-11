// Package verbose implements rm's -v/--verbose flag: report each removal.
package verbose

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/rm"
	"github.com/braswelljr/gnux/internal/docmeta"
)

// Meta returns the documentation metadata for the verbose flag.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:  "verbose",
		Use:   "verbose [FILE]...",
		Short: "Explain what is being removed (-v/--verbose).",
		Long: "-v/--verbose prints a line for each item removed, e.g. removed 'notes.txt'. To " +
			"keep the report in the order you listed the arguments, verbose removal runs " +
			"sequentially rather than concurrently.",
		Permissions: "-v does not change permission handling or what may be removed; it only " +
			"reports the items that were successfully removed. Files it could not remove are " +
			"reported as errors on stderr as usual.",
		UseCases: []string{
			"Auditing exactly what a recursive or wildcard delete removed.",
			"Confirming a script deleted the files you expected.",
			"Troubleshooting by pairing -v with -i to see and approve each removal.",
			"Keeping a log of a cleanup by saving the report to a file.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Remove several files, printing a line per file:",
				Bash:        "rmx -v a.txt b.txt",
				PowerShell:  "rmx -v a.txt b.txt",
			},
			{
				Description: "Recursively remove a directory, reporting each removal:",
				Bash:        "rmx -rv dist/",
				PowerShell:  "rmx -rv dist",
			},
			{
				Description: "Save the removal report to a log file:",
				Bash:        "rmx -rv cache/ > cleanup.log",
				PowerShell:  "rmx -rv cache | Out-File cleanup.log",
			},
			{
				Description: "Count the entries removed:",
				Bash:        "rmx -rv tmp/ | wcx -l",
				PowerShell:  "rmx -rv tmp | wcx -l",
			},
			{
				Description: "Report removals of files that may not exist, with the gnux shell:",
				Bash:        "gnux shell -c 'rmx -fv *.bak'",
				PowerShell:  "gnux shell -c 'rmx -fv *.bak'",
			},
		},
	}
}

// Register installs the -v/--verbose flag on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("verbose", "v", false, "explain what is being done")
}

// Apply sets o.Verbose from the -v flag.
func Apply(fs *pflag.FlagSet, o *rm.Options) {
	o.Verbose, _ = fs.GetBool("verbose")
}
