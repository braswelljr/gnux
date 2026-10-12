// Package ddx wires the ddx command (GNU dd) to its engine.
package ddx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/dd"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the ddx tool. Its operands are KEY=VALUE pairs rather than
// GNU-style options, so they are parsed by the engine (dd.Parse).
func Spec() tool.Spec {
	return tool.Spec{
		Meta:    Meta(),
		GNU:     "dd",
		Run:     run,
		RawArgs: true,
	}
}

// Meta returns the documentation metadata for ddx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    dd.Name,
		SeeAlso: []string{"catx", "headx", "truncatex", "wcx"},
		Use:     dd.Name + " [OPERAND]...",
		Short:   "Convert and copy a file, block by block.",
		Long: "ddx copies its input to its output in blocks, optionally converting it, and " +
			"reports what it copied on standard error. Operands are KEY=VALUE: if=FILE and " +
			"of=FILE name the input and output (standard input and output by default); bs=N " +
			"sets the block size for both and writes each block as read, while ibs=N and " +
			"obs=N set them separately (512 by default) and reblock the output; cbs=N is the " +
			"record size for conv=block and conv=unblock; skip=N (also iseek=N) skips N input " +
			"blocks, seek=N (also oseek=N) skips N output blocks, and count=N copies only N " +
			"input blocks. N may carry a suffix: c (1), w (2), b (512), kB, MB, GB, TB, PB, " +
			"EB (powers of 1000) or K, M, G, T, P, E, alone or followed by iB (powers of " +
			"1024), and may be a product such as 2x512. conv= takes a comma-separated list: " +
			"notrunc (do not truncate the output), noerror (continue after read errors), sync " +
			"(pad short input blocks to ibs with NUL bytes, or spaces with block/unblock), " +
			"fsync and fdatasync (flush the output before finishing), excl and nocreat (fail " +
			"if the output exists, or does not), lcase and ucase, swab (swap each pair of " +
			"bytes), block (newline-terminated lines to fixed cbs-sized records) and unblock " +
			"(the reverse). iflag= accepts fullblock, count_bytes and skip_bytes; oflag= " +
			"accepts seek_bytes and append; binary and text are accepted by both and ignored. " +
			"status=none prints nothing but errors, status=noxfer omits the transfer line. " +
			"The summary reads '1+0 records in / 1+0 records out / 512 bytes copied, 0.0001 s, " +
			"5.1 MB/s', where N+M counts full and partial blocks. Unless conv=notrunc, a " +
			"regular output file is truncated at the seek offset; an output that cannot seek " +
			"is advanced by writing NUL bytes. Differences from GNU dd: status=progress " +
			"prints only the final statistics; the conversions ascii, ebcdic, ibm, sparse " +
			"and nocache and the flags direct, dsync, sync, nonblock, noatime, noctty, " +
			"nofollow and cio are not implemented and are refused as invalid; block and " +
			"unblock do nothing without cbs; a closed output pipe ends ddx quietly instead of " +
			"by SIGPIPE; and no statistics are printed on SIGINFO or SIGUSR1.",
		Permissions: "Needs read permission on the input file and write permission on the " +
			"output file, or on its directory when ddx creates it (with mode 0666 less the " +
			"umask). Writing to a device file such as a disk needs the permissions the " +
			"operating system requires for it, usually root: ddx overwrites whatever is " +
			"there, so check of= twice.",
		UseCases: []string{
			"Creating a file of a given size filled with zeros, for tests or swap space.",
			"Copying the first or last blocks of a file or device, such as a boot sector.",
			"Patching bytes in place in the middle of a file with seek= and conv=notrunc.",
			"Measuring raw read or write throughput with the transfer statistics.",
			"Converting fixed-length records to lines and back with conv=unblock and conv=block.",
			"Changing the case of a file or swapping byte order while copying it.",
			"Cutting a byte range out of a binary file with skip=, count= and the _bytes flags.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Create a 1 MiB file of zeros:",
				Bash:        "ddx if=/dev/zero of=blank.img bs=1M count=1",
				PowerShell:  "ddx if=/dev/zero of=blank.img bs=1M count=1",
			},
			{
				Description: "Copy a file in 4 KiB blocks:",
				Bash:        "ddx if=disk.img of=copy.img bs=4K",
				PowerShell:  "ddx if=disk.img of=copy.img bs=4K",
			},
			{
				Description: "Save the first 512-byte block of a file:",
				Bash:        "ddx if=disk.img of=mbr.bin bs=512 count=1",
				PowerShell:  "ddx if=disk.img of=mbr.bin bs=512 count=1",
			},
			{
				Description: "Extract bytes 100 to 149 of a file:",
				Bash:        "ddx if=data.bin of=part.bin skip=100 count=50 iflag=skip_bytes,count_bytes",
				PowerShell:  "ddx if=data.bin of=part.bin skip=100 count=50 iflag=skip_bytes,count_bytes",
			},
			{
				Description: "Overwrite four bytes at offset 16 without truncating the file:",
				Bash:        "printfx 'ABCD' | ddx of=data.bin seek=16 oflag=seek_bytes conv=notrunc",
				PowerShell:  "printfx 'ABCD' | ddx of=data.bin seek=16 oflag=seek_bytes conv=notrunc",
			},
			{
				Description: "Upper-case a file while copying it:",
				Bash:        "ddx if=notes.txt of=NOTES.txt conv=ucase",
				PowerShell:  "ddx if=notes.txt of=NOTES.txt conv=ucase",
			},
			{
				Description: "Swap the byte order of 16-bit values:",
				Bash:        "ddx if=le16.raw of=be16.raw conv=swab",
				PowerShell:  "ddx if=le16.raw of=be16.raw conv=swab",
			},
			{
				Description: "Turn 80-byte fixed records into lines:",
				Bash:        "ddx if=cards.dat of=cards.txt cbs=80 conv=unblock",
				PowerShell:  "ddx if=cards.dat of=cards.txt cbs=80 conv=unblock",
			},
			{
				Description: "Read whole blocks from a pipe:",
				Bash:        "catx big.log | ddx of=chunk.log bs=64K count=1 iflag=fullblock",
				PowerShell:  "catx big.log | ddx of=chunk.log bs=64K count=1 iflag=fullblock",
			},
			{
				Description: "Copy silently, refusing to overwrite an existing file:",
				Bash:        "ddx if=src.bin of=dst.bin conv=excl status=none",
				PowerShell:  "ddx if=src.bin of=dst.bin conv=excl status=none",
			},
			{
				Description: "Append a file to another:",
				Bash:        "ddx if=extra.log of=all.log oflag=append conv=notrunc",
				PowerShell:  "ddx if=extra.log of=all.log oflag=append conv=notrunc",
			},
			{
				Description: "Measure write throughput and flush to disk before finishing:",
				Bash:        "ddx if=/dev/zero of=test.bin bs=1M count=256 conv=fsync",
				PowerShell:  "ddx if=/dev/zero of=test.bin bs=1M count=256 conv=fsync",
			},
			{
				Description: "Copy standard input to standard output in the gnux shell:",
				Bash:        "gnux shell -c 'echox hello | ddx status=none'",
				PowerShell:  "gnux shell -c 'echox hello | ddx status=none'",
			},
		},
	}
}

func run(_ *pflag.FlagSet, args []string, s core.Streams) error {
	return dd.New(dd.Options{}, s).Run(args)
}
