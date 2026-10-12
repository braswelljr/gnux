//go:build windows

package uname

import (
	"os"
	"runtime"
	"strconv"
	"syscall"
	"unsafe"
)

// currentVersion is the registry key Windows records its version under.
const currentVersion = `SOFTWARE\Microsoft\Windows NT\CurrentVersion`

// system reports Windows_NT, the version from the registry (major.minor.build
// as the release, the feature update such as 23H2 as the version), and the
// architecture gnux was built for.
func system() (Info, error) {
	node, err := os.Hostname()
	if err != nil {
		node = unknown
	}
	info := Info{
		KernelName:       "Windows_NT",
		Nodename:         node,
		KernelRelease:    unknown,
		KernelVersion:    unknown,
		Machine:          goarchMachine[runtime.GOARCH],
		Processor:        unknown,
		HardwarePlatform: unknown,
		OperatingSystem:  "Windows",
	}

	key, err := openKey(currentVersion)
	if err != nil {
		return info, nil
	}
	defer syscall.RegCloseKey(key) //nolint:errcheck // best-effort close of a read-only key

	major, okMajor := dword(key, "CurrentMajorVersionNumber")
	minor, okMinor := dword(key, "CurrentMinorVersionNumber")
	build := str(key, "CurrentBuildNumber")
	if okMajor && okMinor && build != "" {
		info.KernelRelease = strconv.FormatUint(uint64(major), 10) + "." +
			strconv.FormatUint(uint64(minor), 10) + "." + build
	}
	for _, name := range []string{"DisplayVersion", "ReleaseId"} {
		if v := str(key, name); v != "" {
			info.KernelVersion = v
			break
		}
	}
	return info, nil
}

func openKey(path string) (syscall.Handle, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var key syscall.Handle
	err = syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, p, 0, syscall.KEY_READ, &key)
	return key, err
}

func dword(key syscall.Handle, name string) (uint32, bool) {
	p, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0, false
	}
	var typ, v uint32
	n := uint32(4)
	if err := syscall.RegQueryValueEx(key, p, nil, &typ, (*byte)(unsafe.Pointer(&v)), &n); err != nil || typ != syscall.REG_DWORD {
		return 0, false
	}
	return v, true
}

func str(key syscall.Handle, name string) string {
	p, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return ""
	}
	buf := make([]uint16, 256)
	n := uint32(len(buf) * 2)
	var typ uint32
	if err := syscall.RegQueryValueEx(key, p, nil, &typ, (*byte)(unsafe.Pointer(&buf[0])), &n); err != nil || typ != syscall.REG_SZ {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
