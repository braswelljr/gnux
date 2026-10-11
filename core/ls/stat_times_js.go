//go:build js

package ls

import (
	"syscall"
	"time"
)

// statTimes returns the access and status-change times. The js runtime's
// Stat_t carries them as separate second and nanosecond fields, and has no
// birth time, so --time=birth falls back to the modification time.
func statTimes(st *syscall.Stat_t) (atime, ctime, btime time.Time) {
	return time.Unix(st.Atime, st.AtimeNsec), time.Unix(st.Ctime, st.CtimeNsec), time.Time{}
}
