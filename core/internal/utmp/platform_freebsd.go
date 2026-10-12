//go:build freebsd

package utmp

// DefaultFile is where the system records who is logged in.
const DefaultFile = "/var/run/utx.active"

var decode = DecodeFreeBSD
