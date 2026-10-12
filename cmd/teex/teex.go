// Package teex wires the teex command (GNU tee) to its engine.
package teex

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tee"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the teex tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "tee",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for teex.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    tee.Name,
		SeeAlso: []string{"catx", "headx", "tailx"},
		Use:     tee.Name + " [OPTION]... [FILE]...",
		Short:   "Copy standard input to each FILE, and also to standard output.",
		Long: "teex reads standard input and writes it, unchanged and as soon as it arrives, to " +
			"standard output and to every FILE. Each FILE is created if it does not exist and " +
			"truncated if it does, unless -a appends to it instead. A FILE named - is a file " +
			"called '-', as in GNU tee 9, not standard output. A FILE that cannot be opened is " +
			"reported and the copy goes on to the others; the exit status is then 1. " +
			"--output-error=MODE says what happens when writing to an output fails: 'warn' " +
			"reports the error and carries on with the other outputs, 'warn-nopipe' does the " +
			"same but drops an output whose reader has gone without a word, 'exit' stops at the " +
			"first error and 'exit-nopipe' stops at the first error that is not a closed pipe. " +
			"--output-error with no MODE, and -p, mean 'warn-nopipe'. Without the option a " +
			"failing output is reported and dropped, and a closed pipe stops teex quietly, as " +
			"SIGPIPE stops GNU tee. teex stops reading once no output is left. When -p and " +
			"--output-error=MODE are both given, MODE wins whatever their order. -i is accepted " +
			"but has no effect: gnux runs tools in-process (also inside gnux shell and the " +
			"playground), so it does not change how the process handles interrupts.",
		Permissions: "Needs write permission on each existing FILE, or write and search " +
			"permission on the directory of a FILE it creates. New files get mode 0666 masked " +
			"by the umask. Without -a an existing FILE is truncated before anything is written.",
		UseCases: []string{
			"Saving a command's output to a log while still watching it on the terminal.",
			"Keeping a copy of an intermediate stage of a pipeline for debugging.",
			"Appending each run of a build or test to a running log with -a.",
			"Writing the same data to several files at once.",
			"Capturing CI output as an artifact while it streams to the job log.",
			"Keeping a pipeline going when a downstream reader exits early, with -p.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Show output and save it to a file:",
				Bash:        "lsx -l | teex listing.txt",
				PowerShell:  "lsx -l | teex listing.txt",
			},
			{
				Description: "Append to a log instead of overwriting it:",
				Bash:        "echox 'build started' | teex -a build.log",
				PowerShell:  "'build started' | teex -a build.log",
			},
			{
				Description: "Write the same input to several files:",
				Bash:        "echox 'v1.2.3' | teex VERSION dist/VERSION",
				PowerShell:  "'v1.2.3' | teex VERSION dist/VERSION",
			},
			{
				Description: "Save a copy of an intermediate pipeline stage:",
				Bash:        "catx access.log | teex raw.txt | wcx -l",
				PowerShell:  "catx access.log | teex raw.txt | wcx -l",
			},
			{
				Description: "Write a file without echoing it to the terminal:",
				Bash:        "echox 'KEY=value' | teex .env > /dev/null",
				PowerShell:  "'KEY=value' | teex .env > $null",
			},
			{
				Description: "Save test output while still seeing it:",
				Bash:        "go test ./... 2>&1 | teex test.log",
				PowerShell:  "go test ./... 2>&1 | teex test.log",
			},
			{
				Description: "Keep going when the reader stops early, without an error:",
				Bash:        "yesx | teex -p copy.txt | headx -n 3",
				PowerShell:  "yesx | teex -p copy.txt | headx -n 3",
			},
			{
				Description: "Report a closed pipe as an error and stop:",
				Bash:        "yesx | teex --output-error=exit copy.txt | headx -n 1",
				PowerShell:  "yesx | teex --output-error=exit copy.txt | headx -n 1",
			},
			{
				Description: "Carry on to the other files when one cannot be written:",
				Bash:        "echox data | teex --output-error=warn a.txt missing/b.txt c.txt",
				PowerShell:  "'data' | teex --output-error=warn a.txt missing/b.txt c.txt",
			},
			{
				Description: "Write to a file literally named '-':",
				Bash:        "echox data | teex -",
				PowerShell:  "'data' | teex -",
			},
			{
				Description: "Log a script's steps in the portable gnux shell:",
				Bash:        "gnux shell -c 'lsx -1 | teex -a files.log | wcx -l'",
				PowerShell:  "gnux shell -c 'lsx -1 | teex -a files.log | wcx -l'",
			},
		},
	}
}

// Register installs the teex flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("append", "a", false, "append to the given FILEs, do not overwrite")
	fs.BoolP("ignore-interrupts", "i", false, "ignore interrupt signals (accepted; no effect in gnux)")
	fs.BoolP("output-error-warn-nopipe", "p", false, "operate in a more appropriate MODE with pipes")
	fs.String("output-error", "",
		"set behavior on write error; `MODE` can be: warn, warn-nopipe, exit, exit-nopipe")
	fs.Lookup("output-error").NoOptDefVal = "warn-nopipe"
}

// outputErrorModes maps the --output-error argument to its mode.
var outputErrorModes = map[string]tee.OutputError{
	"warn":        tee.OutputErrorWarn,
	"warn-nopipe": tee.OutputErrorWarnNoPipe,
	"exit":        tee.OutputErrorExit,
	"exit-nopipe": tee.OutputErrorExitNoPipe,
}

// buildOptions folds the parsed flags into tee.Options. An explicit
// --output-error=MODE wins over -p in either order: it is the more specific
// request.
func buildOptions(fs *pflag.FlagSet) (tee.Options, error) {
	on := func(name string) bool {
		b, _ := fs.GetBool(name)
		return b
	}
	o := tee.Options{
		Append:           on("append"),
		IgnoreInterrupts: on("ignore-interrupts"),
	}
	if on("output-error-warn-nopipe") {
		o.OutputError = tee.OutputErrorWarnNoPipe
	}
	if fs.Changed("output-error") {
		mode, _ := fs.GetString("output-error")
		m, ok := outputErrorModes[mode]
		if !ok {
			return tee.Options{}, core.Usagef("invalid argument '%s' for '--output-error'\nValid arguments are:\n"+
				"  - 'warn'\n  - 'warn-nopipe'\n  - 'exit'\n  - 'exit-nopipe'", mode)
		}
		o.OutputError = m
	}
	return o, nil
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	opts, err := buildOptions(fs)
	if err != nil {
		return err
	}
	return tee.New(opts, s).Run(args)
}
