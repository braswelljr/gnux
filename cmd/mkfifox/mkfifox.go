// Package mkfifox wires the mkfifox command (GNU mkfifo) to its engine.
package mkfifox

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/mkfifo"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the mkfifox tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "mkfifo",
		Flags: Register,
		Run:   run,
	}
}

// Register installs the mkfifox flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.StringP("mode", "m", "", "set file permission bits to `MODE`, not a=rw - umask")
	RegisterContext(fs)
}

// RegisterContext installs -Z and --context[=CTX], which mkfifox and mknodx
// accept as GNU does on a kernel without SELinux.
func RegisterContext(fs *pflag.FlagSet) {
	fs.StringP("context", "Z", "", "set the SELinux security context to default type, or to `CTX` (ignored: gnux does not support SELinux)")
	fs.Lookup("context").NoOptDefVal = " "
}

// Context is the --context=CTX value: empty for -Z, or --context with none.
func Context(fs *pflag.FlagSet) string {
	if c, _ := fs.GetString("context"); c != " " {
		return c
	}
	return ""
}

// buildOptions folds the parsed flags into mkfifo.Options.
func buildOptions(fs *pflag.FlagSet) mkfifo.Options {
	mode, _ := fs.GetString("mode")
	return mkfifo.Options{Mode: mode, Context: Context(fs)}
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return mkfifo.New(buildOptions(fs), s).Run(args)
}

// Meta returns the documentation metadata for mkfifox.
func Meta() docmeta.Meta {
	n := mkfifo.Name
	return docmeta.Meta{
		Name:    n,
		SeeAlso: []string{"mknodx", "mkdirx", "teex"},
		Use:     n + " [OPTION]... NAME...",
		Short:   "Make named pipes (FIFOs).",
		Long: n + " creates a named pipe, a FIFO, at each NAME: a file one program writes to " +
			"and another reads from, each waiting for the other. The pipes are readable and " +
			"writable by everyone, less the umask; -m MODE sets their permissions exactly, " +
			"octal or symbolic as chmod takes it, applied to a=rw (only the permission bits: " +
			"a mode with the setuid, setgid or sticky bit is refused). A NAME that cannot be " +
			"created is reported and the rest are still made. -Z and --context are accepted " +
			"and ignored, as by GNU mkfifo on a kernel without SELinux. Windows has no named " +
			"pipes in its file system, so there every NAME fails with 'Operation not supported'.",
		Permissions: "Needs write and search permission on the directory each pipe is created in.",
		UseCases: []string{
			"Connecting two programs that are not started from the same shell pipeline.",
			"Feeding a command's output to a program that only reads from a file name.",
			"Simple job queues and logs between processes on one machine.",
		},
		Examples: []docmeta.Example{
			{Description: "Pass data from one terminal to another:", Bash: n + " /tmp/channel\ncatx /tmp/channel        # in one terminal: waits for a writer\necho hello > /tmp/channel   # in another"},
			{Description: "Make a pipe only its owner can use:", Bash: n + " -m 600 private.fifo"},
			{Description: "Compress a log as it is written:", Bash: n + " app.log && gzip < app.log > app.log.gz &\n./app --log app.log"},
			{Description: "Make several pipes at once:", Bash: n + " in.fifo out.fifo"},
		},
	}
}
