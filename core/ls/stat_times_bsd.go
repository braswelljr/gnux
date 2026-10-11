//go:build darwin || freebsd || netbsd

package ls

import (
	"syscall"
	"time"
)

// statTimes returns the access, status-change and birth times.
func statTimes(st *syscall.Stat_t) (atime, ctime, btime time.Time) {
	return timespec(st.Atimespec), timespec(st.Ctimespec), timespec(st.Birthtimespec)
}
