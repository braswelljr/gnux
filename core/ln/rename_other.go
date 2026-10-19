//go:build !windows

package ln

import (
	"github.com/braswelljr/gnux/core/fsys"
)

// renameOver moves the new link at tmp onto dst, replacing it in one step.
func renameOver(tmp, dst string) error {
	return fsys.Rename(tmp, dst)
}
