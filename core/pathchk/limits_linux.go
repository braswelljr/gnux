//go:build linux

package pathchk

import "golang.org/x/sys/unix"

// nameMax is glibc's pathconf(_PC_NAME_MAX): the file system's f_namelen.
func nameMax(dir string) (int, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(dir, &st); err != nil {
		return 0, err
	}
	return int(st.Namelen), nil
}

// pathMax is glibc's pathconf(_PC_PATH_MAX), the same for every file system.
func pathMax(string) (int, error) { return unix.PathMax, nil }
