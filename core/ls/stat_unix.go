//go:build !windows

package ls

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
	"time"
)

// statSys extracts the stat(2) fields behind info.
func statSys(info os.FileInfo) sysInfo {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return sysInfo{nlink: 1}
	}
	// The widths (and signedness) of these fields differ between Linux, darwin
	// and the BSDs, so each conversion is required somewhere even where
	// unconvert, run on a single platform, flags it.
	s := sysInfo{
		ino:    uint64(st.Ino),   //nolint:unconvert
		nlink:  uint64(st.Nlink), //nolint:unconvert
		uid:    uint32(st.Uid),   //nolint:unconvert
		gid:    uint32(st.Gid),   //nolint:unconvert
		blocks: int64(st.Blocks), //nolint:unconvert
		rdev:   uint64(st.Rdev),  //nolint:unconvert
	}
	s.atime, s.ctime, s.btime = statTimes(st)
	return s
}

// timespec converts a stat timestamp to a time.Time.
func timespec(ts syscall.Timespec) time.Time {
	return time.Unix(int64(ts.Sec), int64(ts.Nsec)) //nolint:unconvert // int32 on 32-bit platforms
}

// idCache memoizes user and group name lookups, which would otherwise hit the
// account database once per listed file.
type idCache struct {
	users, groups map[uint32]string
}

// user returns the login name for uid, or false when there is none.
func (c *idCache) user(uid uint32) (string, bool) {
	if c.users == nil {
		c.users = make(map[uint32]string)
	}
	name, seen := c.users[uid]
	if !seen {
		if u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10)); err == nil {
			name = u.Username
		}
		c.users[uid] = name
	}
	return name, name != ""
}

// group returns the group name for gid, or false when there is none.
func (c *idCache) group(gid uint32) (string, bool) {
	if c.groups == nil {
		c.groups = make(map[uint32]string)
	}
	name, seen := c.groups[gid]
	if !seen {
		if g, err := user.LookupGroupId(strconv.FormatUint(uint64(gid), 10)); err == nil {
			name = g.Name
		}
		c.groups[gid] = name
	}
	return name, name != ""
}
