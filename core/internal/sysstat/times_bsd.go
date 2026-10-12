//go:build darwin || freebsd || netbsd

package sysstat

import (
	"syscall"
	"time"
)

// times returns the access, change and birth times.
func times(st *syscall.Stat_t) (atime, ctime, btime time.Time) {
	return timespec(st.Atimespec), timespec(st.Ctimespec), timespec(st.Birthtimespec)
}
