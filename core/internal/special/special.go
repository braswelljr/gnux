// Package special creates special files: named pipes for mkfifo, and pipes
// and device files for mknod. Each platform supplies what it can; elsewhere
// the calls fail with ErrUnsupported.
package special

import "errors"

// ErrUnsupported is returned where the system has no such files.
var ErrUnsupported = errors.New("Operation not supported") //nolint:revive,staticcheck // GNU's strerror(ENOTSUP), shown as is
