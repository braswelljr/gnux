//go:build linux

package uname

import (
	"syscall"
)

// system reads uname(2).
func system() (Info, error) {
	var u syscall.Utsname
	if err := syscall.Uname(&u); err != nil {
		return Info{}, err
	}
	return Info{
		KernelName:       chars(u.Sysname[:]),
		Nodename:         chars(u.Nodename[:]),
		KernelRelease:    chars(u.Release[:]),
		KernelVersion:    chars(u.Version[:]),
		Machine:          chars(u.Machine[:]),
		Processor:        unknown,
		HardwarePlatform: unknown,
		OperatingSystem:  "GNU/Linux",
	}, nil
}

// chars converts a NUL-terminated utsname field, whose element type is int8
// or uint8 depending on the architecture.
func chars[T int8 | uint8](field []T) string {
	b := make([]byte, 0, len(field))
	for _, c := range field {
		if c == 0 {
			break
		}
		b = append(b, byte(c))
	}
	return string(b)
}
