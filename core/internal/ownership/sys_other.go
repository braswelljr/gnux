//go:build !unix

package ownership

import (
	"errors"
	"os"
)

// errNotSupported is the error for every change off Unix: Windows and the
// browser have no numeric owners or groups.
var errNotSupported = errors.New("operation not supported")

// ownerOf reports that the owner is unknown.
func ownerOf(_ os.FileInfo) (uid, gid int, ok bool) {
	return 0, 0, false
}

// chown fails: ownership cannot be changed here.
func chown(path string, _, _ int, _ bool) error {
	return &os.PathError{Op: "chown", Path: path, Err: errNotSupported}
}
