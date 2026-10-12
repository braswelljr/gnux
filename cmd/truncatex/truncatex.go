// Package truncatex wires the truncatex command (GNU truncate) to its engine.
package truncatex

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/truncate"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the truncatex tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "truncate",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for truncatex.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    truncate.Name,
		SeeAlso: []string{"touchx", "headx", "wcx"},
		Use:     truncate.Name + " OPTION... FILE...",
		Short:   "Shrink or extend the size of each FILE to the specified size.",
		Long: "truncatex sets the size of each FILE. A FILE that does not exist is created, " +
			"unless -c is given. A file larger than the size loses the data past it; a file " +
			"shorter than the size is extended with zero bytes (on most file systems as a " +
			"sparse hole that takes no disk space). -s SIZE gives the size: a number with an " +
			"optional suffix (K, M, G, T, P, E for powers of 1024, also written KiB, MiB; KB, " +
			"MB for powers of 1000), optionally preceded by + (extend by), - (reduce by), " +
			"< (at most), > (at least), / (round down to a multiple of) or % (round up to a " +
			"multiple of). -r FILE takes the size from another file; with -s as well, the " +
			"relative SIZE adjusts the reference size. -o counts SIZE in I/O blocks rather " +
			"than bytes. One of -s and -r is required. Off Unix the I/O block size used by -o " +
			"is 512 bytes, because the platform does not report one.",
		Permissions: "Needs write permission on each FILE, and write permission on its " +
			"directory to create it. -r needs search permission on the reference file's " +
			"directory only, since only its size is read.",
		UseCases: []string{
			"Emptying a log file in place without changing its owner, mode or open handles.",
			"Creating a large sparse file of an exact size for testing or as a disk image.",
			"Padding a file to a block multiple with -s %4K.",
			"Making a file exactly the size of another with -r.",
			"Capping a file at a maximum size with -s '<1M'.",
			"Growing a file by a fixed amount with -s +1M.",
			"Resizing only files that already exist with -c in a cleanup script.",
			"Preallocating a placeholder file before a download fills it.",
		},
		Examples: []docmeta.Example{
			{Description: "Empty a log file without deleting it:", Bash: "truncatex -s 0 app.log", PowerShell: "truncatex -s 0 app.log"},
			{Description: "Create a 100 MiB (sparse) file:", Bash: "truncatex -s 100M disk.img", PowerShell: "truncatex -s 100M disk.img"},
			{Description: "Create a file of exactly one million bytes:", Bash: "truncatex -s 1MB data.bin", PowerShell: "truncatex -s 1MB data.bin"},
			{Description: "Grow a file by 1 KiB of zero bytes:", Bash: "truncatex -s +1K data.bin", PowerShell: "truncatex -s +1K data.bin"},
			{Description: "Remove the last 10 bytes of a file:", Bash: "truncatex -s -10 data.bin", PowerShell: "truncatex -s -10 data.bin"},
			{Description: "Cap a file at 1 MiB, leaving smaller files alone:", Bash: "truncatex -s '<1M' cache.db", PowerShell: "truncatex -s '<1M' cache.db"},
			{Description: "Make sure a file is at least 4 KiB:", Bash: "truncatex -s '>4K' header.bin", PowerShell: "truncatex -s '>4K' header.bin"},
			{Description: "Round a file's size up to a multiple of 512 bytes:", Bash: "truncatex -s %512 image.raw", PowerShell: "truncatex -s %512 image.raw"},
			{Description: "Round a file's size down to a multiple of 4 KiB:", Bash: "truncatex -s /4K image.raw", PowerShell: "truncatex -s /4K image.raw"},
			{Description: "Make a file the same size as another:", Bash: "truncatex -r original.bin copy.bin", PowerShell: "truncatex -r original.bin copy.bin"},
			{Description: "Make a file 100 bytes larger than another:", Bash: "truncatex -r original.bin -s +100 copy.bin", PowerShell: "truncatex -r original.bin -s +100 copy.bin"},
			{Description: "Empty every log that exists, without creating missing ones:", Bash: "truncatex -c -s 0 a.log b.log c.log", PowerShell: "truncatex -c -s 0 a.log b.log c.log"},
			{Description: "Size a file in I/O blocks rather than bytes:", Bash: "truncatex -o -s 8 blocks.bin", PowerShell: "truncatex -o -s 8 blocks.bin"},
			{Description: "Check the result with wcx:", Bash: "truncatex -s 2K f && wcx -c f", PowerShell: "truncatex -s 2K f; wcx -c f"},
		},
	}
}

// Register installs the truncatex flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("no-create", "c", false, "do not create any files")
	fs.BoolP("io-blocks", "o", false, "treat SIZE as number of IO blocks instead of bytes")
	fs.StringP("reference", "r", "", "base size on `RFILE`")
	fs.StringP("size", "s", "", "set or adjust the file size by `SIZE` bytes")
}

// buildOptions folds the parsed flags into truncate.Options.
func buildOptions(fs *pflag.FlagSet) truncate.Options {
	var o truncate.Options
	o.NoCreate, _ = fs.GetBool("no-create")
	o.IOBlocks, _ = fs.GetBool("io-blocks")
	o.Reference, _ = fs.GetString("reference")
	o.Size, _ = fs.GetString("size")
	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return truncate.New(buildOptions(fs), s).Run(args)
}
