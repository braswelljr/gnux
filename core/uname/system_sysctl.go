//go:build darwin || freebsd

package uname

import (
	"syscall"
)

// system reads the kern and hw sysctl nodes, which is where uname(3) gets
// its answers on macOS and FreeBSD.
func system() (Info, error) {
	read := func(name string) string {
		v, err := syscall.Sysctl(name)
		if err != nil {
			return unknown
		}
		return v
	}
	sysname, err := syscall.Sysctl("kern.ostype")
	if err != nil {
		return Info{}, err
	}
	return Info{
		KernelName:       sysname,
		Nodename:         read("kern.hostname"),
		KernelRelease:    read("kern.osrelease"),
		KernelVersion:    singleLine(read("kern.version")),
		Machine:          read("hw.machine"),
		Processor:        processor(),
		HardwarePlatform: unknown,
		OperatingSystem:  sysname,
	}, nil
}
