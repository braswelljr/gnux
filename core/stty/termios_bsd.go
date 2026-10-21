//go:build darwin || freebsd

package stty

import (
	"errors"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// The BSDs keep the speeds in fields of their own, in baud, and have no
// line discipline field; _POSIX_VDISABLE is 0xff.
const (
	kernelState = unix.PENDIN
	hasLine     = false
	vdisable    = 0xff
	swtchIsSusp = true
	nonblock    = syscall.O_NONBLOCK
)

func set(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

func cfgetospeed(t *termios) uint64 { return t.ospeed }
func cfgetispeed(t *termios) uint64 { return t.ispeed }

func cfsetospeed(t *termios, speed uint64) error {
	t.ospeed = speed
	return nil
}

func cfsetispeed(t *termios, speed uint64) error {
	t.ispeed = speed
	return nil
}

func getAttr(fd uintptr) (termios, error) {
	k, err := unix.IoctlGetTermios(int(fd), unix.TIOCGETA)
	if err != nil {
		return termios{}, err
	}
	return fromUnix(k), nil
}

func setAttr(fd uintptr, t *termios, drain bool) error {
	req := uint(unix.TIOCSETA)
	if drain {
		req = unix.TIOCSETAW
	}
	k := toUnix(t)
	return unix.IoctlSetTermios(int(fd), req, &k)
}

func getWinsize(fd uintptr) (rows, cols int, err error) {
	ws, err := unix.IoctlGetWinsize(int(fd), unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, err
	}
	return int(ws.Row), int(ws.Col), nil
}

func setWinsize(fd uintptr, rows, cols int) error {
	ws, err := unix.IoctlGetWinsize(int(fd), unix.TIOCGWINSZ)
	if err != nil {
		ws = &unix.Winsize{}
	}
	ws.Row, ws.Col = uint16(rows), uint16(cols)
	return unix.IoctlSetWinsize(int(fd), unix.TIOCSWINSZ, ws)
}

// setExtproc sets the BSD extproc flag, which has an ioctl of its own.
func setExtproc(fd uintptr, on bool) error {
	v := 0
	if on {
		v = 1
	}
	return unix.IoctlSetPointerInt(int(fd), unix.TIOCEXT, v)
}

func isEINVAL(err error) bool { return errors.Is(err, syscall.EINVAL) }

func clearNonblock(f *os.File) { _ = unix.SetNonblock(int(f.Fd()), false) }
