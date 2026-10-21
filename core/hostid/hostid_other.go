//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package hostid

// Windows has no gethostid: the identifier is derived from the host's
// address, as glibc does when /etc/hostid is missing.
func hostID() uint32 { return fromHostname() }
