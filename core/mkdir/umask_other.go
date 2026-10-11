//go:build !unix

package mkdir

// umask reports no file-creation mask: the concept does not exist off Unix.
func umask() uint32 {
	return 0
}
