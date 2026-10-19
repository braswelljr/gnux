//go:build !unix

package find

import (
	"github.com/braswelljr/gnux/core/fsys"
)

// The mode bits of access(2).
const (
	accessExec  = 0x1
	accessWrite = 0x2
	accessRead  = 0x4
)

// access approximates -readable, -writable and -executable off Unix
// (Windows, WebAssembly), where there is no access(2): every existing file
// is readable, the owner write bit stands for the read-only attribute, and a
// directory or a file with an owner execute bit is executable.
func access(path string, mode uint32) bool {
	info, err := fsys.Stat(path)
	if err != nil {
		return false
	}
	perm := info.Mode().Perm()
	switch mode {
	case accessWrite:
		return perm&0o200 != 0
	case accessExec:
		return info.IsDir() || perm&0o100 != 0
	}
	return true
}
