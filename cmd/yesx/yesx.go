// Package yesx wires the yesx command (GNU yes) to its engine.
package yesx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/yes"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the yesx tool. It has no options of its own, so every
// argument is taken as text to repeat.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:    Meta(),
		GNU:     "yes",
		Run:     run,
		RawArgs: true,
	}
}

// Meta returns the documentation metadata for yesx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    yes.Name,
		SeeAlso: []string{"headx", "echox", "rmx"},
		ExitStatus: []docmeta.Status{
			{Code: "0", Meaning: "the reader went away (a closed pipe is the normal way yesx ends)"},
			{Code: "1", Meaning: "a write error other than a closed pipe"},
		},
		Use:   yes.Name + " [STRING]...",
		Short: "Output a string repeatedly until killed.",
		Long: "yesx repeatedly writes a line made of every STRING joined by single spaces, or " +
			"'y' when none is given, until it is killed or its output is closed. Output is " +
			"written in large batches of whole lines, so it is fast enough to use as a data " +
			"source. When the reader goes away (as in `yesx | head`) yesx stops quietly; any " +
			"other write failure is reported and exits with status 1. A first '--' is taken as " +
			"the end of options and not printed; --help and --version are honored only when " +
			"they are the sole argument, and every other argument is repeated as text.",
		Permissions: "Needs no filesystem permissions; it only writes to standard output.",
		UseCases: []string{
			"Answering every prompt of an interactive command automatically.",
			"Generating an endless or, with head, a fixed amount of filler input.",
			"Loading a CPU core or a pipe for a quick throughput test.",
			"Declining every prompt with 'yesx n' to see what a command would ask.",
			"Creating a test file of an exact size with headx -c.",
			"Producing numbered or repeated sample rows for testing other tools.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Answer 'y' to every prompt of a command:",
				Bash:        "yesx | rmx -i *.tmp",
				PowerShell:  "yesx | rmx -i (Get-ChildItem -Name *.tmp)",
			},
			{
				Description: "Answer 'n' instead:",
				Bash:        "yesx n | rmx -i *.tmp",
				PowerShell:  "yesx n | rmx -i (Get-ChildItem -Name *.tmp)",
			},
			{
				Description: "Confirm every step of an interactive recursive removal:",
				Bash:        "yesx | rmx -ri build",
				PowerShell:  "yesx | rmx -ri build",
			},
			{
				Description: "Produce five copies of a line:",
				Bash:        "yesx 'all work and no play' | headx -n 5",
				PowerShell:  "yesx 'all work and no play' | headx -n 5",
			},
			{
				Description: "Join several arguments into one repeated line:",
				Bash:        "yesx a b c | headx -n 2",
				PowerShell:  "yesx a b c | headx -n 2",
			},
			{
				Description: "Repeat text that starts with a dash after --:",
				Bash:        "yesx -- -n | headx -n 2",
				PowerShell:  "yesx -- -n | headx -n 2",
			},
			{
				Description: "Create a file of exactly one mebibyte:",
				Bash:        "yesx 0123456789 | headx -c 1M > filler.txt && wcx -c filler.txt",
				PowerShell:  "yesx 0123456789 | headx -c 1M > filler.txt; wcx -c filler.txt",
			},
			{
				Description: "Number a few repeated rows:",
				Bash:        "yesx row | headx -n 4 | catx -n",
				PowerShell:  "yesx row | headx -n 4 | catx -n",
			},
			{
				Description: "Measure pipe throughput by counting 100 MiB:",
				Bash:        "yesx | headx -c 100M | wcx -c",
				PowerShell:  "yesx | headx -c 100M | wcx -c",
			},
			{
				Description: "Stop cleanly inside a gnux shell pipeline:",
				Bash:        "gnux shell -c 'yesx hello | headx -n 3 > out.txt; wcx -l out.txt'",
				PowerShell:  "gnux shell -c 'yesx hello | headx -n 3 > out.txt; wcx -l out.txt'",
			},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return yes.New(s).Run(args)
}
