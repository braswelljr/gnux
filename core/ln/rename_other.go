//go:build !windows

package ln

import "os"

// renameOver moves the new link at tmp onto dst, replacing it in one step.
func renameOver(tmp, dst string) error {
	return os.Rename(tmp, dst)
}
