//go:build windows

package ls

import (
	"os"
	"syscall"
	"unsafe"
)

var getConsoleScreenBufferInfo = syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleScreenBufferInfo")

// terminalWidth asks the console behind f for its window width in columns,
// returning 0 when f is not a console.
func terminalWidth(f *os.File) int {
	// CONSOLE_SCREEN_BUFFER_INFO from <wincon.h>.
	var info struct {
		sizeX, sizeY             int16
		cursorX, cursorY         int16
		attributes               uint16
		left, top, right, bottom int16
		maxWindowX, maxWindowY   int16
	}
	//nolint:gosec // the call fills a fixed-size struct we own
	ok, _, _ := getConsoleScreenBufferInfo.Call(f.Fd(), uintptr(unsafe.Pointer(&info)))
	if ok == 0 {
		return 0
	}
	return int(info.right-info.left) + 1
}
