//go:build js

package sysstat

import (
	"syscall"
	"time"
)

// times returns the access and change times; the js runtime has no birth
// time.
func times(st *syscall.Stat_t) (atime, ctime, btime time.Time) {
	return time.Unix(st.Atime, st.AtimeNsec), time.Unix(st.Ctime, st.CtimeNsec), time.Time{}
}
