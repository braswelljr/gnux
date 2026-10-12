//go:build !unix

package chmod

// processUmask reports no file-creation mask: the concept does not exist off
// Unix.
func processUmask() uint32 {
	return 0
}
