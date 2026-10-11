//go:build !(linux || openbsd || dragonfly || solaris || darwin || freebsd || netbsd || windows)

package touch

import (
	"os"
	"time"
)

// AccessTime returns the time info's file was last accessed. This platform's
// stat record is not known here, so the modification time stands in for it.
func AccessTime(info os.FileInfo) time.Time {
	return info.ModTime()
}
