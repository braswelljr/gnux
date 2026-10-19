//go:build !windows && !darwin && !freebsd && !netbsd && !js && !wasip1

package sysstat

import (
	"syscall"
	"time"
)

// times returns the access and change times; stat(2) reports no birth time
// here.
func times(st *syscall.Stat_t) (atime, ctime, btime time.Time) {
	return timespec(st.Atim), timespec(st.Ctim), time.Time{}
}
