//go:build freebsd

package sandbox

import (
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/unix"

	"github.com/braswelljr/gnux/core/fsys"
)

// On FreeBSD the sandbox is Capsicum's capability mode: once in it, the
// process can open nothing by path, only relative to directories it already
// holds. gnux opens the policy's directories (and, read-only, the system's)
// as os.Root handles, points every engine's file access at them (core/fsys),
// and enters capability mode, so that the kernel refuses whatever does not
// go through them. Programs cannot be started by path in capability mode, so
// a command can run gnux's own tools only.

// systemRead are what stays readable.
var systemRead = []string{"/usr/share", "/etc", "/usr/local/share"}

func enter(p Policy) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("sandbox: %w", err)
	}
	rooted, err := fsys.NewRooted(p.Write, append(append([]string(nil), p.Read...), existing(systemRead)...), cwd)
	if err != nil {
		return fmt.Errorf("sandbox: %w", err)
	}
	// What is read by path only once, and lazily, must be read now: the
	// local time zone.
	_ = time.Local.String()
	fsys.Use(rooted)
	if err := unix.CapEnter(); err != nil {
		fsys.Use(fsys.OS{})
		rooted.Close()
		if errors.Is(err, unix.ENOSYS) {
			return fmt.Errorf("sandbox: %w: this kernel has no Capsicum (options CAPABILITY_MODE)", ErrUnsupported)
		}
		return fmt.Errorf("sandbox: entering capability mode: %w", err)
	}
	return nil
}

// refusedHere reports whether err is capability mode refusing a path.
func refusedHere(err error) bool { return errors.Is(err, unix.ECAPMODE) }
