//go:build !unix

package mode

// Umask reports no file-creation mask: the concept does not exist off Unix.
func Umask() uint32 {
	return 0
}
