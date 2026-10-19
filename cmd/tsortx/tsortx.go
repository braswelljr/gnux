// Package tsortx wires the tsortx command (GNU tsort) to its engine.
package tsortx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tsort"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the tsortx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta: Meta(),
		GNU:  "tsort",
		Run:  run,
	}
}

// Meta returns the documentation metadata for tsortx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    tsort.Name,
		SeeAlso: []string{"sortx", "uniqx", "joinx"},
		Use:     tsort.Name + " [OPTION] [FILE]",
		Short:   "Write a totally ordered list consistent with the partial ordering in FILE.",
		Long: "tsortx reads pairs of strings, separated by spaces, tabs and newlines, from FILE " +
			"(standard input when there is none, or when FILE is -). Each pair 'A B' says that A " +
			"comes before B; a pair of the same string twice just names that string. It prints " +
			"every string once, each after every string it was paired before. It follows GNU's " +
			"algorithm, so the order is GNU's: the strings that nothing comes before are taken " +
			"in byte order, then strings in the order the ones before them are printed. When " +
			"the pairs contain a loop, tsortx reports 'input contains a loop:' and the strings " +
			"of the loop on standard error, drops one pair of it and carries on, so every " +
			"string is still printed, and exits 1. An odd number of strings is an error and " +
			"prints nothing.",
		Permissions: "Needs read permission on FILE. Writes nothing.",
		UseCases: []string{
			"Ordering build steps, packages or modules so each comes after what it depends on.",
			"Finding a dependency loop: tsortx names the strings in it.",
			"Ordering database migrations or initialization scripts by their prerequisites.",
			"Scheduling tasks from a list of 'before after' pairs.",
			"Ordering object files for a linker from their symbol dependencies.",
		},
		ExitStatus: []docmeta.Status{
			{Code: "0", Meaning: "every string was printed in an order consistent with every pair"},
			{Code: "1", Meaning: "the input contains a loop (every string is still printed), an odd number of strings, or FILE cannot be read"},
		},
		Examples: []docmeta.Example{
			{
				Description: "Order tasks from 'before after' pairs:",
				Bash:        "printfx 'fetch build\\nbuild test\\nbuild package\\n' | tsortx",
				PowerShell:  "printfx 'fetch build\\nbuild test\\nbuild package\\n' | tsortx",
				Output:      "fetch\nbuild\npackage\ntest",
			},
			{
				Description: "Order the pairs listed in a file:",
				Bash:        "tsortx deps.txt",
				PowerShell:  "tsortx deps.txt",
			},
			{
				Description: "Put several pairs on one line:",
				Bash:        "echox 'a b b c c d' | tsortx",
				PowerShell:  "echox 'a b b c c d' | tsortx",
				Output:      "a\nb\nc\nd",
			},
			{
				Description: "Name a string that has no relations by pairing it with itself:",
				Bash:        "printfx 'a b\\nlonely lonely\\n' | tsortx",
				PowerShell:  "printfx 'a b\\nlonely lonely\\n' | tsortx",
			},
			{
				Description: "Find a dependency loop:",
				Bash:        "printfx 'a b\\nb c\\nc a\\n' | tsortx",
				PowerShell:  "printfx 'a b\\nb c\\nc a\\n' | tsortx",
				Output:      "tsortx: -: input contains a loop:\ntsortx: a\ntsortx: b\ntsortx: c\na\nb\nc",
			},
			{
				Description: "Fail a build script when the dependencies contain a loop:",
				Bash:        "tsortx deps.txt > order.txt || echox 'dependency loop' >&2",
				PowerShell:  "tsortx deps.txt > order.txt; if ($LASTEXITCODE -ne 0) { 'dependency loop' }",
			},
			{
				Description: "Print the order reversed, dependents first:",
				Bash:        "tsortx deps.txt | tacx",
				PowerShell:  "tsortx deps.txt | tacx",
			},
			{
				Description: "Number the steps of the order:",
				Bash:        "tsortx deps.txt | nlx",
				PowerShell:  "tsortx deps.txt | nlx",
			},
			{
				Description: "Read the pairs from standard input given as -:",
				Bash:        "catx deps.txt | tsortx -",
				PowerShell:  "catx deps.txt | tsortx -",
			},
			{
				Description: "Run steps in dependency order in the portable gnux shell:",
				Bash:        "gnux shell -c 'for step in $(tsortx deps.txt); do echox \"running $step\"; done'",
				PowerShell:  "gnux shell -c 'for step in $(tsortx deps.txt); do echox \"running $step\"; done'",
			},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return tsort.New(tsort.Options{}, s).Run(args)
}
