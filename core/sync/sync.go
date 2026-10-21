// Package sync implements the engine behind syncx, a GNU-compatible
// implementation of the GNU/UNIX sync command: flush cached writes to
// persistent storage, for the whole system or for the files named.
package sync

import (
	"os"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fsys"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "syncx"

// Mode is what is flushed for each FILE.
type Mode int

const (
	// File flushes the file's data and metadata (fsync).
	File Mode = iota
	// Data flushes the file's data and only the metadata needed to read it
	// back (fdatasync), with -d.
	Data
	// FileSystem flushes the whole file system the file is on (syncfs), with -f.
	FileSystem
)

// Options mirrors the GNU sync flag surface. The zero value is plain sync.
type Options struct {
	// Data (-d) syncs only file data, no unneeded metadata.
	Data bool
	// FileSystem (-f) syncs the file systems that contain the files.
	FileSystem bool
}

// Sync is the engine. Construct it with New.
type Sync struct {
	Options

	// SyncAll flushes every file system, for a run with no FILE. SyncFile
	// flushes one open file in the given mode. They are the injection points
	// tests use; they default to the system calls.
	SyncAll  func() error
	SyncFile func(f *os.File, m Mode) error

	diag *core.Diag
}

// New builds a sync engine writing its diagnostics to s.
func New(opts Options, s core.Streams) *Sync {
	return &Sync{Options: opts, SyncAll: syncAll, SyncFile: syncFile, diag: core.NewDiag(Name, s)}
}

// Run flushes the system, or each FILE in turn. A file that cannot be opened
// or flushed is reported and the rest are still flushed.
func (y *Sync) Run(args []string) error {
	if y.Data && y.FileSystem {
		y.diag.Errorf("cannot specify both --data and --file-system")
		return y.diag.Result()
	}
	if y.Data && len(args) == 0 {
		y.diag.Errorf("--data needs at least one argument")
		return y.diag.Result()
	}
	if len(args) == 0 {
		if err := y.SyncAll(); err != nil {
			y.diag.Errorf("%s", core.ErrText(err))
		}
		return y.diag.Result()
	}
	mode := File
	switch {
	case y.Data:
		mode = Data
	case y.FileSystem:
		mode = FileSystem
	}
	for _, name := range args {
		y.file(name, mode)
	}
	return y.diag.Result()
}

func (y *Sync) file(name string, mode Mode) {
	// GNU opens read-only, and write-only when reading is refused, so that a
	// file one may only write can still be flushed.
	f, err := fsys.OpenFile(name, os.O_RDONLY|nonblock, 0)
	if err != nil {
		if wf, werr := fsys.OpenFile(name, os.O_WRONLY|nonblock, 0); werr == nil {
			f, err = wf, nil
		}
	}
	if err != nil {
		y.diag.Errorf("error opening '%s': %s", name, core.ErrText(err))
		return
	}
	defer f.Close()
	if err := y.SyncFile(f, mode); err != nil {
		y.diag.Errorf("error syncing '%s': %s", name, core.ErrText(err))
	}
}
