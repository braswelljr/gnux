//go:build darwin || freebsd

package fsstat

import "syscall"

// mntNoWait asks getfsstat for the cached statistics, without waiting on
// slow (network) file systems.
const mntNoWait = 2

// Statfs reports the file system holding path.
func Statfs(path string) (FS, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return FS{}, err
	}
	return fromStatfs(&st), nil
}

// Mounts asks getfsstat for every mounted file system.
func Mounts() ([]Mount, error) {
	n, err := syscall.Getfsstat(nil, mntNoWait)
	if err != nil {
		return nil, err
	}
	buf := make([]syscall.Statfs_t, n)
	if n, err = syscall.Getfsstat(buf, mntNoWait); err != nil {
		return nil, err
	}
	out := make([]Mount, 0, n)
	for _, st := range buf[:n] {
		m := Mount{
			Source: cString(st.Mntfromname[:]),
			Target: cString(st.Mntonname[:]),
			Type:   cString(st.Fstypename[:]),
		}
		m.Dummy = IsDummy(m.Type) || m.Source == "map auto_home" || m.Source == "map -hosts"
		m.Remote = isRemote(m.Type, m.Source)
		out = append(out, m)
	}
	return out, nil
}
