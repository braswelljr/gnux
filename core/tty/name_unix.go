//go:build unix

package tty

import (
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/braswelljr/gnux/core"
)

// Lookup reports whether stream is a terminal and finds its file name the way
// ttyname(3) does: through /proc where the system has it, otherwise by
// looking in /dev for the character device with the same device number.
func Lookup(stream any) (string, bool) {
	f, ok := stream.(*os.File)
	if !ok || !core.IsTerminal(f) {
		return "", false
	}
	if name, err := os.Readlink("/proc/self/fd/" + strconv.Itoa(int(f.Fd()))); err == nil && filepath.IsAbs(name) {
		return name, true
	}
	info, err := f.Stat()
	if err != nil {
		return "", true
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", true
	}
	rdev := uint64(st.Rdev) //nolint:unconvert // int32 on darwin
	for _, dir := range []string{"/dev/pts", "/dev"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.Type()&os.ModeCharDevice == 0 {
				continue
			}
			p := filepath.Join(dir, e.Name())
			var cand syscall.Stat_t
			if syscall.Stat(p, &cand) == nil && uint64(cand.Rdev) == rdev && p != "/dev/tty" { //nolint:unconvert // int32 on darwin
				return p, true
			}
		}
	}
	return "", true
}
