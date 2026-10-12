// Package nprocx wires the nprocx command (GNU nproc) to its engine.
package nprocx

import (
	"strconv"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/nproc"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the nprocx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "nproc",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for nprocx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    nproc.Name,
		SeeAlso: []string{"unamex", "envx"},
		Use:     nproc.Name + " [OPTION]...",
		Short:   "Print the number of processing units available.",
		Long: "nprocx prints the number of processing units available to the current process. " +
			"As in GNU nproc, the OMP_NUM_THREADS environment variable (its first value, when " +
			"it is a comma-separated list) replaces that count when it is a positive number, and " +
			"OMP_THREAD_LIMIT caps the result. --all prints the number of installed processors " +
			"instead and ignores both variables. --ignore=N subtracts N, but the result is never " +
			"less than 1. gnux has no cgo, so both counts come from Go's runtime.NumCPU: the " +
			"processors this process may run on. A machine with processors taken offline, or a " +
			"process restricted to fewer of them, prints the same number with and without --all.",
		Permissions: "Needs no permissions. Reads only the process's CPU affinity and its " +
			"environment.",
		UseCases: []string{
			"Choosing how many parallel jobs a build should run.",
			"Leaving a core free for the rest of the system with --ignore=1.",
			"Respecting OMP_NUM_THREADS when a job scheduler has set it.",
			"Reporting the machine's size in a CI log or a bug report.",
			"Sizing a worker pool in a portable script.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the number of processing units:",
				Bash:        "nprocx",
				PowerShell:  "nprocx",
			},
			{
				Description: "Print the number of installed processors:",
				Bash:        "nprocx --all",
				PowerShell:  "nprocx --all",
			},
			{
				Description: "Leave one core free:",
				Bash:        "nprocx --ignore=1",
				PowerShell:  "nprocx --ignore=1",
			},
			{
				Description: "Run a build with one job per core:",
				Bash:        "make -j\"$(nprocx)\"",
				PowerShell:  "make -j(nprocx)",
			},
			{
				Description: "Respect an OpenMP thread count:",
				Bash:        "OMP_NUM_THREADS=4 nprocx",
				PowerShell:  "$env:OMP_NUM_THREADS = 4; nprocx",
			},
			{
				Description: "Cap the count with OMP_THREAD_LIMIT:",
				Bash:        "OMP_THREAD_LIMIT=2 nprocx",
				PowerShell:  "$env:OMP_THREAD_LIMIT = 2; nprocx",
			},
			{
				Description: "Set the count through envx, on any platform:",
				Bash:        "envx OMP_NUM_THREADS=3 nprocx",
				PowerShell:  "envx OMP_NUM_THREADS=3 nprocx",
			},
			{
				Description: "Save the count in a variable:",
				Bash:        "jobs=$(nprocx --ignore=2); echox \"using $jobs jobs\"",
				PowerShell:  "$jobs = nprocx --ignore=2; \"using $jobs jobs\"",
			},
			{
				Description: "Print the count with a label:",
				Bash:        "printfx 'cores: %d\\n' \"$(nprocx)\"",
				PowerShell:  "printfx 'cores: %d\\n' (nprocx)",
			},
			{
				Description: "Show the count in the portable gnux shell:",
				Bash:        "gnux shell -c 'nprocx --all'",
				PowerShell:  "gnux shell -c 'nprocx --all'",
			},
		},
	}
}

// Register installs the nprocx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.Bool("all", false, "print the number of installed processors")
	fs.String("ignore", "0", "if possible, exclude `N` processing units")
}

// buildOptions folds the parsed flags into nproc.Options.
func buildOptions(fs *pflag.FlagSet) (nproc.Options, error) {
	all, _ := fs.GetBool("all")
	ignore, _ := fs.GetString("ignore")
	n, err := strconv.ParseUint(ignore, 10, 64)
	if err != nil {
		return nproc.Options{}, core.Usagef("invalid number: '%s'", ignore)
	}
	return nproc.Options{All: all, Ignore: n}, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return nproc.New(opts, s).Run(args)
}
