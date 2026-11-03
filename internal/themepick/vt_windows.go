//go:build windows

package themepick

import (
	"os"

	"golang.org/x/sys/windows"
)

// enableVT turns on escape-sequence output for a Windows console (Windows
// Terminal has it on; the older console host does not) and returns how to
// put the console back.
func enableVT(out *os.File) func() {
	h := windows.Handle(out.Fd())
	var mode uint32
	if windows.GetConsoleMode(h, &mode) != nil {
		return func() {}
	}
	if windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) != nil {
		return func() {}
	}
	return func() { _ = windows.SetConsoleMode(h, mode) }
}
