//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package pathchk

import "golang.org/x/sys/unix"

// The pathconf names, the same on every BSD since 4.4BSD's <unistd.h>.
const (
	pcNameMax = 4
	pcPathMax = 5
)

func nameMax(dir string) (int, error) { return unix.Pathconf(dir, pcNameMax) }

func pathMax(dir string) (int, error) { return unix.Pathconf(dir, pcPathMax) }
