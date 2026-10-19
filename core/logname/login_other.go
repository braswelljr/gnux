//go:build !unix && !windows

package logname

// login reports no login name where nobody logs in (WebAssembly).
func login(any) (string, bool) {
	return "", false
}
