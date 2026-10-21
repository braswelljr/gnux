//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package pathchk

import (
	"errors"
	"io/fs"

	"github.com/braswelljr/gnux/core/fsys"
)

// Windows has no pathconf: the limits are NTFS's, 255 characters for a
// component and MAX_PATH (260, with the terminating null) for a path.
func nameMax(dir string) (int, error) {
	if _, err := fsys.Stat(dir); err != nil && errors.Is(err, fs.ErrNotExist) {
		return 0, err
	}
	return 255, nil
}

func pathMax(string) (int, error) { return 260, nil }
