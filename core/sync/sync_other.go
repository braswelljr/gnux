//go:build !unix

package sync

import "os"

// Windows has no unprivileged call that flushes every volume, and js/wasm no
// storage of its own: a run with no FILE has nothing it can flush, and
// succeeds. Every write already reaches the cache all processes share.
func syncAll() error { return nil }

// A file is flushed with FlushFileBuffers (os.File.Sync) whatever the mode.
func syncFile(f *os.File, _ Mode) error { return f.Sync() }

const nonblock = 0
