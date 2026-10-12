//go:build !unix

package tty

import (
	"os"

	"github.com/braswelljr/gnux/core"
)

// Lookup reports whether stream is a terminal. Windows has no terminal
// file names; a console is the device CON, so that is its name.
func Lookup(stream any) (string, bool) {
	f, ok := stream.(*os.File)
	if !ok || !core.IsTerminal(f) {
		return "", false
	}
	return "CON", true
}
