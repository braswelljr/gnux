// Package syncx wires the syncx command (GNU sync) to its engine.
package syncx

import (
	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sync"
	"github.com/braswelljr/gnux/internal/docmeta"
	"github.com/braswelljr/gnux/internal/tool"
)

// Spec describes the syncx tool.
func Spec() tool.Spec {
	return tool.Spec{
		Meta:  Meta(),
		GNU:   "sync",
		Flags: Register,
		Run:   run,
	}
}

// Meta returns the documentation metadata for syncx.
func Meta() docmeta.Meta {
	return docmeta.Meta{
		Name:    sync.Name,
		SeeAlso: []string{"ddx", "shredx", "dfx"},
		Use:     sync.Name + " [OPTION] [FILE]...",
		Short:   "Synchronize cached writes to persistent storage.",
		Long: "syncx asks the system to write data held in memory out to disk. With no FILE " +
			"it flushes every file system; with FILEs it flushes only those files (fsync), " +
			"-d only their data and the metadata needed to read it back (fdatasync), and -f " +
			"the whole file system each one is on (syncfs). -d needs at least one FILE, and " +
			"-d and -f cannot be combined. A FILE that cannot be opened or flushed is reported " +
			"and the others are still flushed. Off Linux, -d flushes as plain fsync does and " +
			"-f flushes every file system, since those systems have neither fdatasync nor " +
			"syncfs. On Windows, a FILE is flushed with FlushFileBuffers, and a run with no " +
			"FILE succeeds without flushing anything: Windows has no unprivileged call that " +
			"flushes every volume.",
		Permissions: "Needs read or write permission on each FILE (it is opened read-only, " +
			"or write-only when reading is refused). Flushing the whole system needs no " +
			"permission. Nothing is ever modified.",
		UseCases: []string{
			"Making sure an image written with ddx has reached a USB stick before unplugging it.",
			"Flushing a file a script has just written before cutting the power or snapshotting a VM.",
			"Forcing a database dump or a backup archive to disk before reporting success.",
		},
		Examples: []docmeta.Example{
			{
				Description: "Flush everything before pulling out a USB stick:",
				Bash:        "syncx",
				PowerShell:  "syncx",
			},
			{
				Description: "Make sure a backup you just wrote is on disk:",
				Bash:        "tar -czf backup.tgz data && syncx backup.tgz",
				PowerShell:  "tar -czf backup.tgz data; syncx backup.tgz",
			},
			{
				Description: "Flush only the file data, the cheaper call, after appending to a log:",
				Bash:        "syncx -d app.log",
				PowerShell:  "syncx -d app.log",
			},
			{
				Description: "Flush the whole file system a mount point is on:",
				Bash:        "syncx -f /mnt/usb",
				PowerShell:  "syncx -f E:\\",
			},
		},
	}
}

// Register installs the syncx flags on fs.
func Register(fs *pflag.FlagSet) {
	fs.BoolP("data", "d", false, "sync only file data, no unneeded metadata")
	fs.BoolP("file-system", "f", false, "sync the file systems that contain the files")
}

// buildOptions folds the parsed flags into sync.Options.
func buildOptions(fs *pflag.FlagSet) sync.Options {
	var o sync.Options
	o.Data, _ = fs.GetBool("data")
	o.FileSystem, _ = fs.GetBool("file-system")
	return o
}

func run(fs *pflag.FlagSet, args []string, s core.Streams) error {
	return sync.New(buildOptions(fs), s).Run(args)
}
