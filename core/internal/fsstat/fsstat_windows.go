//go:build windows

package fsstat

import (
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// getDiskFreeSpace is GetDiskFreeSpaceW, which x/sys/windows does not wrap;
// it gives the cluster size that the byte counts are divided into.
var getDiskFreeSpace = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetDiskFreeSpaceW")

// Statfs reports the volume holding path.
func Statfs(path string) (FS, error) {
	root, err := volumeRoot(path)
	if err != nil {
		return FS{}, err
	}
	rootp, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return FS{}, err
	}
	var avail, total, free uint64
	if err := windows.GetDiskFreeSpaceEx(rootp, &avail, &total, &free); err != nil {
		return FS{}, err
	}
	var sectorsPerCluster, bytesPerSector, freeClusters, totalClusters uint32
	cluster := uint64(4096)
	if r, _, _ := getDiskFreeSpace.Call(uintptr(unsafe.Pointer(rootp)),
		uintptr(unsafe.Pointer(&sectorsPerCluster)), uintptr(unsafe.Pointer(&bytesPerSector)),
		uintptr(unsafe.Pointer(&freeClusters)), uintptr(unsafe.Pointer(&totalClusters))); r != 0 {
		if c := uint64(sectorsPerCluster) * uint64(bytesPerSector); c > 0 {
			cluster = c
		}
	}
	var serial, maxComponent, flags uint32
	name := make([]uint16, windows.MAX_PATH+1)
	_ = windows.GetVolumeInformation(rootp, nil, 0, &serial, &maxComponent, &flags, &name[0], uint32(len(name)))
	typeName := windows.UTF16ToString(name)
	return FS{
		TypeName: typeName,
		Remote:   windows.GetDriveType(rootp) == windows.DRIVE_REMOTE,
		IOSize:   cluster, BlockSize: cluster,
		Blocks: total / cluster, Bfree: free / cluster, Bavail: int64(avail / cluster),
		NameMax: uint64(maxComponent),
		ID:      uint64(serial),
	}, nil
}

// volumeRoot returns the root of the volume holding path, such as C:\.
func volumeRoot(path string) (string, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	buf := make([]uint16, windows.MAX_PATH+1)
	if err := windows.GetVolumePathName(p, &buf[0], uint32(len(buf))); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buf), nil
}

// Mounts lists the drives (C:\, D:\, ...) as mount points.
func Mounts() ([]Mount, error) {
	buf := make([]uint16, 512)
	n, err := windows.GetLogicalDriveStrings(uint32(len(buf)), &buf[0])
	if err != nil {
		return nil, err
	}
	// The roots are NUL-separated, ending with an empty one.
	var out []Mount
	start := 0
	for i := 0; i < int(n); i++ {
		if buf[i] == 0 {
			if i > start {
				out = append(out, driveMount(windows.UTF16ToString(buf[start:i])))
			}
			start = i + 1
		}
	}
	return out, nil
}

func driveMount(root string) Mount {
	m := Mount{Source: strings.TrimSuffix(root, `\`), Target: root}
	if p, err := windows.UTF16PtrFromString(root); err == nil {
		m.Remote = windows.GetDriveType(p) == windows.DRIVE_REMOTE
		name := make([]uint16, windows.MAX_PATH+1)
		if windows.GetVolumeInformation(p, nil, 0, nil, nil, nil, &name[0], uint32(len(name))) == nil {
			m.Type = windows.UTF16ToString(name)
		}
	}
	return m
}
