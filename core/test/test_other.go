//go:build !unix

package test

import (
	"os"

	"github.com/braswelljr/gnux/core/fsys"
)

// access approximates -r/-w/-x off Unix, where there is no access(2): every
// existing file is readable, the owner write bit stands for the read-only
// attribute, and a directory or a file with an owner execute bit is
// executable.
func access(path string, mode uint32) bool {
	info, err := fsys.Stat(path)
	if err != nil {
		return false
	}
	perm := info.Mode().Perm()
	switch mode {
	case 2:
		return perm&0o200 != 0
	case 1:
		return info.IsDir() || perm&0o100 != 0
	}
	return true
}

// owner is unknown off Unix, so -O and -G are false there.
func owner(os.FileInfo) (uid, gid int, ok bool) { return 0, 0, false }

func euid() int { return -1 }
func egid() int { return -1 }
