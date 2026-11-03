//go:build !unix

package sync

import (
	"errors"
	"io/fs"
	"os"

	"github.com/braswelljr/gnux/core/fsys"
)

// Windows has no unprivileged call that flushes every volume, and js/wasm no
// storage of its own: a run with no FILE has nothing it can flush, and
// succeeds. Every write already reaches the cache all processes share.
func syncAll() error { return nil }

// A file is flushed with FlushFileBuffers (os.File.Sync) whatever the mode.
// FlushFileBuffers needs a handle open for writing, and the file was opened
// read-only as GNU opens it, so a refusal is retried on a write-only handle
// (no truncation). A file that cannot be opened for writing keeps the error.
func syncFile(f *os.File, _ Mode) error {
	err := f.Sync()
	if !errors.Is(err, fs.ErrPermission) {
		return err
	}
	w, werr := fsys.OpenFile(f.Name(), os.O_WRONLY, 0)
	if werr != nil {
		return err
	}
	defer w.Close()
	return w.Sync()
}

const nonblock = 0
