//go:build !unix

package chmod

import (
	"errors"
	"os"
)

// errNoLinkMode stands for EOPNOTSUPP where symbolic links have no mode.
var errNoLinkMode = errors.New("operation not supported")

// lchmod refuses: off Unix a symbolic link has no mode of its own.
func lchmod(path string, _ os.FileMode) error {
	return &os.PathError{Op: "lchmod", Path: path, Err: errNoLinkMode}
}

// linkModeUnsupported reports whether err says symbolic links have no mode
// of their own on this system.
func linkModeUnsupported(err error) bool {
	return errors.Is(err, errNoLinkMode)
}
