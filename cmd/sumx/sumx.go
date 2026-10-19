// Package sumx wires the sumx command (GNU sum) to its engine.
package sumx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sum"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the sumx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "sum",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for sumx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    sum.Name,
		SeeAlso: []string{"cksumx", "md5sumx", "sha256sumx", "b2sumx"},
		Use:     sum.Name + " [OPTION]... [FILE]...",
		Short:   "Print the checksum and block count of each FILE.",
		Long: "sumx prints a 16-bit checksum of each FILE and the number of blocks it takes up, " +
			"followed by its name when FILE operands are given, as GNU sum 9 does. With no FILE, " +
			"or when FILE is -, it reads standard input. By default (and with -r) it computes " +
			"the BSD checksum, printed with five digits, and counts 1024-byte blocks; -s " +
			"(--sysv) computes the System V checksum and counts 512-byte blocks. Both are the " +
			"checksums cksumx prints with -a bsd and -a sysv. They exist to compare with old " +
			"systems' output and catch accidental corruption only: use sha256sumx to verify a " +
			"download. GNU lets the last of -r and -s win; gnux cannot see their order, so -s " +
			"wins whenever it is given. A FILE that cannot be read is reported and the rest are " +
			"still summed.",
		Permissions: "Needs read permission on each FILE. Writes nothing.",
		UseCases: []string{
			"Comparing a file with a checksum published by an old Unix system or a legacy manual.",
			"Quickly spotting that two copies of a file differ.",
			"Reproducing the output of sum in a script that expects it.",
			"Checking System V checksums with -s.",
			"Getting a file's size in kilobyte blocks along with its checksum.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Print the BSD checksum and 1K-block count of a file:",
				Bash:        "sumx README.md",
				PowerShell:  "sumx README.md",
				Output:      "36979     1 README.md",
			},
			{
				Description: "Print the System V checksum and 512-byte block count:",
				Bash:        "sumx -s README.md",
				PowerShell:  "sumx -s README.md",
			},
			{
				Description: "Checksum several files at once:",
				Bash:        "sumx *.txt",
				PowerShell:  "sumx (Get-ChildItem *.txt).Name",
			},
			{
				Description: "Checksum standard input:",
				Bash:        "printfx 'hello\\n' | sumx",
				PowerShell:  "printfx 'hello\\n' | sumx",
				Output:      "36979     1",
			},
			{
				Description: "Compare two copies of a file:",
				Bash:        "[ \"$(sumx < a.bin)\" = \"$(sumx < b.bin)\" ] && echox same",
				PowerShell:  "if ((Get-Content a.bin -Raw | sumx) -eq (Get-Content b.bin -Raw | sumx)) { 'same' }",
			},
			{
				Description: "Ask for the BSD algorithm explicitly:",
				Bash:        "sumx -r archive.tar",
				PowerShell:  "sumx -r archive.tar",
			},
			{
				Description: "Mix standard input with files:",
				Bash:        "catx header.txt | sumx - body.txt",
				PowerShell:  "catx header.txt | sumx - body.txt",
			},
			{
				Description: "Keep a checksum list for later comparison:",
				Bash:        "sumx -s *.cfg > sums.txt",
				PowerShell:  "sumx -s (Get-ChildItem *.cfg).Name > sums.txt",
			},
			{
				Description: "Get the same checksum through cksumx:",
				Bash:        "cksumx -a bsd README.md",
				PowerShell:  "cksumx -a bsd README.md",
			},
			{
				Description: "Checksum every file under a directory in the portable gnux shell:",
				Bash:        "gnux shell -c 'findx . -type f | xargsx sumx'",
				PowerShell:  "gnux shell -c 'findx . -type f | xargsx sumx'",
			},
		},
	}
}

// Register installs the sumx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("sysv", "s", false, "use the System V sum algorithm, use 512-byte blocks")

	// GNU spells -r with a letter only. pflag wants a long name for each
	// option, so it gets a hidden one.
	fs.BoolP(flagR, "r", false, "use the BSD sum algorithm (the default), use 1K blocks")
	fs.Lookup(flagR).Hidden = true
}

// flagR is the hidden long name of -r, which GNU spells with a letter only.
const flagR = "bsd-r"

// buildOptions folds the parsed flags into sum.Options. -s wins over -r,
// whatever their order.
func buildOptions(fs *pflag.FlagSet) sum.Options {
	sysv, _ := fs.GetBool("sysv")
	return sum.Options{SysV: sysv}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return sum.New(buildOptions(fs), s).Run(args)
}
