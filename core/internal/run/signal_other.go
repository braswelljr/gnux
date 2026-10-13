//go:build !unix

package run

import "os"

// signalOf has no signal to report: Windows and js/wasm end a process with an
// exit status only.
func signalOf(*os.ProcessState) int {
	return 0
}
