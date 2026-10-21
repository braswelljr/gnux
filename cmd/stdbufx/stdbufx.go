// Package stdbufx wires the stdbufx command (GNU stdbuf) to its engine.
package stdbufx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/stdbuf"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the stdbufx tool. Options stop at COMMAND, whose own
// options follow, and a command-line mistake exits 125, as in GNU.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:           Meta(),
		GNU:            "stdbuf",
		Flags:          Register,
		Run:            run,
		NoInterspersed: true,
		UsageExit:      stdbuf.ExitCanceled,
	}
}

// Register installs the stdbufx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.StringP("input", "i", "", "adjust standard input stream buffering to `MODE`")
	fs.StringP("output", "o", "", "adjust standard output stream buffering to `MODE`")
	fs.StringP("error", "e", "", "adjust standard error stream buffering to `MODE`")
}

// buildOptions folds the parsed flags into stdbuf.Options.
func buildOptions(fs *pflag.FlagSet) stdbuf.Options {
	var o stdbuf.Options
	o.Input, _ = fs.GetString("input")
	o.Output, _ = fs.GetString("output")
	o.Error, _ = fs.GetString("error")
	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	b := stdbuf.New(buildOptions(fs), s)
	b.Tool = tool.RunTool
	return b.Run(args)
}

// Meta returns the documentation metadata for stdbufx.
func Meta() docmeta.Meta {
	n := stdbuf.Name
	return docmeta.Meta{
		Name:    n,
		SeeAlso: []string{"envx", "nicex", "timeoutx", "teex"},
		Use:     n + " OPTION... COMMAND [ARG]...",
		Short:   "Run a command with modified buffering of its standard streams.",
		Long: n + " runs COMMAND with its standard input, output or error buffered as asked: " +
			"MODE L buffers by line (not for input), 0 not at all, and a size such as 4096, " +
			"64K or 1MB uses a buffer that large. A COMMAND named after a gnux tool runs in " +
			"this process, unchanged: gnux tools already write out each line as soon as " +
			"their input runs dry, which is what -oL is for, so a pipeline of gnux tools needs " +
			"no stdbufx. Any other COMMAND gets GNU coreutils' libstdbuf.so preloaded " +
			"(LD_PRELOAD, or DYLD_INSERT_LIBRARIES on macOS), exactly as GNU stdbuf does, which " +
			"affects programs that use the C library's stdio and have not set their own " +
			"buffering. stdbufx looks for libstdbuf.so beside itself and where GNU coreutils " +
			"packages install it, and without it refuses to run COMMAND ('failed to find " +
			"libstdbuf.so', status 125), as GNU stdbuf does; on Windows only gnux tools can " +
			"be run.",
		Permissions: "Needs execute permission on COMMAND. Changes nothing itself.",
		ExitStatus: []docmeta.Status{
			{Code: "125", Meaning: "stdbufx itself failed: a command-line mistake, or libstdbuf.so was not found"},
			{Code: "126", Meaning: "COMMAND was found but could not be run"},
			{Code: "127", Meaning: "COMMAND was not found"},
			{Code: "N", Meaning: "otherwise, COMMAND's own exit status"},
		},
		UseCases: []string{
			"Seeing a program's output line by line when it writes into a pipe, which it would otherwise buffer.",
			"Watching a log through a filter that holds its output back, such as an awk or python script.",
		},
		Examples: []docmeta.Example{
			{Description: "See each line of a program's output as soon as it is written, through a pipe:",
				Bash: "tail -f app.log | " + n + " -oL awk '/ERROR/' | teex errors.log"},
			{Description: "Make a program's error output unbuffered:", Bash: n + " -e0 ./build.sh 2> build.err"},
			{Description: "gnux tools need none of this, but take it in their stride:", Bash: "tail -f app.log | " + n + " -oL grepx ERROR",
				PowerShell: "Get-Content -Wait app.log | " + n + " -oL grepx ERROR"},
		},
	}
}
