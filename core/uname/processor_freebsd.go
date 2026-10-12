//go:build freebsd

package uname

import "syscall"

// processor reports hw.machine_arch, which uname -p prints on FreeBSD.
func processor() string {
	v, err := syscall.Sysctl("hw.machine_arch")
	if err != nil {
		return unknown
	}
	return v
}
