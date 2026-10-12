//go:build darwin

package utmp

// DefaultFile is where the system records who is logged in.
const DefaultFile = "/var/run/utmpx"

var decode = DecodeDarwin
