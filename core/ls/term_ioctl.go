//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package ls

import (
	"os"
	"syscall"
	"unsafe"
)

// terminalWidth asks the terminal behind f for its width in columns, returning
// 0 when f is not a terminal.
func terminalWidth(f *os.File) int {
	// struct winsize from <sys/ioctl.h>.
	var ws struct {
		rows, cols, xpixel, ypixel uint16
	}
	//nolint:gosec // the ioctl fills a fixed-size struct we own
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws)))
	if errno != 0 {
		return 0
	}
	return int(ws.cols)
}
