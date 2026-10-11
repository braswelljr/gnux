//go:build !windows && !darwin && !freebsd && !netbsd && !js

package ls

import (
	"syscall"
	"time"
)

// statTimes returns the access and status-change times. stat(2) does not report
// a birth time on these platforms, so --time=birth falls back to the
// modification time.
func statTimes(st *syscall.Stat_t) (atime, ctime, btime time.Time) {
	return timespec(st.Atim), timespec(st.Ctim), time.Time{}
}
