//go:build gnux_rooted

package fsys

import (
	"os"
	"path/filepath"
)

// The gnux_rooted build sends every file access of the engines through a
// Rooted file system of the whole disk, so that `go test -tags gnux_rooted
// ./...` runs every engine's tests the way a sandbox runs them (through
// os.Root), and shows what the Capsicum sandbox would break.
func init() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	vol := filepath.VolumeName(wd) + string(filepath.Separator)
	r, err := NewRooted([]string{vol}, nil, wd)
	if err != nil {
		panic(err)
	}
	r.processCwd = true
	Use(r)
}
