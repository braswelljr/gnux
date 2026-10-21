//go:build !unix

package pinky

// shellOf is empty: Windows users have no login shell.
func shellOf(string) string { return "" }
