//go:build !linux && !darwin && !freebsd && !windows

package fsstat

// Statfs is not available here.
func Statfs(string) (FS, error) { return FS{}, ErrUnsupported }

// Mounts is not available here.
func Mounts() ([]Mount, error) { return nil, ErrUnsupported }
