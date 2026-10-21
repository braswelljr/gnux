//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package hostid

import "golang.org/x/sys/unix"

// hostID is the BSD gethostid: the kern.hostid sysctl, which may be zero.
func hostID() uint32 {
	id, err := unix.SysctlUint32("kern.hostid")
	if err != nil {
		return fromHostname()
	}
	return id
}
