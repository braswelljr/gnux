//go:build darwin

package uname

import "syscall"

// processor reports what GNU uname -p prints on macOS: the CPU family, arm
// on Apple silicon and i386 on Intel.
func processor() string {
	m, err := syscall.Sysctl("hw.machine")
	if err != nil {
		return unknown
	}
	switch m {
	case "arm64":
		return "arm"
	case "x86_64", "i386":
		return "i386"
	}
	return unknown
}
