package mkdir

import (
	"os"

	filemode "github.com/braswelljr/gnux/core/internal/mode"
)

// parseMode turns a chmod-style MODE into Unix permission bits. Symbolic
// modes are applied to a=rwx, which is the point of departure GNU mkdir
// documents.
func parseMode(s string, umask func() uint32) (uint32, error) {
	return filemode.Parse(s, 0o777, true, umask)
}

// fileMode converts Unix permission bits to the os.FileMode fsys.Chmod expects.
func fileMode(bits uint32) os.FileMode { return filemode.FileMode(bits) }
