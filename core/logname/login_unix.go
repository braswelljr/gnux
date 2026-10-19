//go:build unix && !linux

package logname

// login is getlogin without the C library: the login record of the terminal
// on stdin. macOS and the BSDs keep the name in the kernel session, which
// only the C library reads, so a process with no terminal has no login name
// here.
func login(stdin any) (string, bool) {
	return fromTerminal(stdin)
}
