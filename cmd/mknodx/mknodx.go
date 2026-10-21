// Package mknodx wires the mknodx command (GNU mknod) to its engine.
package mknodx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/cmd/mkfifox"
	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/mknod"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the mknodx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "mknod",
		Flags: Register,
		Run:   run,
	}
}

// Register installs the mknodx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.StringP("mode", "m", "", "set file permission bits to `MODE`, not a=rw - umask")
	mkfifox.RegisterContext(fs)
}

// buildOptions folds the parsed flags into mknod.Options.
func buildOptions(fs *pflag.FlagSet) mknod.Options {
	mode, _ := fs.GetString("mode")
	return mknod.Options{Mode: mode, Context: mkfifox.Context(fs)}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return mknod.New(buildOptions(fs), s).Run(args)
}

// Meta returns the documentation metadata for mknodx.
func Meta() docmeta.Meta {
	n := mknod.Name
	return docmeta.Meta{
		Name:    n,
		SeeAlso: []string{"mkfifox", "statx", "lsx"},
		Use:     n + " [OPTION]... NAME TYPE [MAJOR MINOR]",
		Short:   "Make block or character special files, or named pipes.",
		Long: n + " creates the special file NAME of TYPE: b for a block device, c (or u) " +
			"for a character device, both with their MAJOR and MINOR device numbers, or p for " +
			"a named pipe, which takes none. Only TYPE's first letter counts, so 'character' " +
			"works. Device numbers are decimal, octal with a leading 0, or hexadecimal with " +
			"0x. The file is readable and writable by everyone, less the umask; -m MODE sets " +
			"its permissions exactly, octal or symbolic as chmod takes it, applied to a=rw " +
			"(only the permission bits). Creating a device file normally needs root. -Z and " +
			"--context are accepted and ignored, as by GNU mknod on a kernel without SELinux. " +
			"Windows has no special files, so there every TYPE fails with 'Operation not supported'.",
		Permissions: "Needs write and search permission on the directory, and for a device " +
			"file the privilege to create one (root, or CAP_MKNOD on Linux).",
		UseCases: []string{
			"Recreating a missing /dev entry in a chroot, container image or rescue system.",
			"Creating a named pipe where only mknod is at hand in a script.",
		},
		Examples: []docmeta.Example{
			{Description: "Recreate /dev/null in a chroot (Linux, as root):", Bash: n + " -m 666 /srv/chroot/dev/null c 1 3"},
			{Description: "Make a named pipe, as mkfifox does:", Bash: n + " jobs.fifo p"},
			{Description: "Create a loop device node (Linux, as root):", Bash: n + " -m 660 /dev/loop8 b 7 8"},
		},
	}
}
