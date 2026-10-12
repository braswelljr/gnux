//go:build !unix

package install

import (
	"errors"
	"os"
	"strconv"
)

// errNotSupported reads "Operation not supported" through core.ErrText.
var errNotSupported = errors.New("operation not supported")

// lookupUser accepts only numeric IDs: there are no Unix accounts here.
func lookupUser(name string) (int, bool) {
	id, err := strconv.Atoi(name)
	return id, err == nil && id >= 0
}

// lookupGroup accepts only numeric IDs: there are no Unix groups here.
func lookupGroup(name string) (int, bool) {
	return lookupUser(name)
}

// chown is not supported off Unix: files have no Unix owner or group.
func chown(string, int, int) error {
	return errNotSupported
}

// sameOwner has nothing to compare off Unix.
func sameOwner(os.FileInfo, int, int) bool {
	return true
}
