//go:build !linux && !darwin && !freebsd

package stty

import (
	"errors"
	"os"
)

// Off Linux, macOS and FreeBSD there is no terminal to set: every setting
// is still checked, against the POSIX names and Linux's values, and then
// reaching the terminal fails.
const (
	kernelState = 0
	ncc         = 32
	flagBits    = 32
	hasLine     = false
	vdisable    = 0
	swtchIsSusp = true
	nonblock    = 0
)

// errUnsupported is ENOTSUP's text.
var errUnsupported = errors.New("Operation not supported") //nolint:revive,staticcheck // strerror(ENOTSUP)

var c = consts{
	PARENB: 0o400, PARODD: 0o1000, CS5: 0, CS6: 0o20, CS7: 0o40, CS8: 0o60, CSIZE: 0o60, HUPCL: 0o2000,
	CSTOPB: 0o100, CREAD: 0o200, CLOCAL: 0o4000,
	IGNBRK: 0o1, BRKINT: 0o2, IGNPAR: 0o4, PARMRK: 0o10, INPCK: 0o20, ISTRIP: 0o40, INLCR: 0o100,
	IGNCR: 0o200, ICRNL: 0o400, IXON: 0o2000, IXOFF: 0o10000, IXANY: 0o4000,
	OPOST: 0o1, ONLCR: 0o4,
	ISIG: 0o1, ICANON: 0o2, ECHO: 0o10, ECHOE: 0o20, ECHOK: 0o40, ECHONL: 0o100, NOFLSH: 0o200, TOSTOP: 0o400,
	VINTR: 0, VQUIT: 1, VERASE: 2, VKILL: 3, VEOF: 4, VTIME: 5, VMIN: 6, VSTART: 8, VSTOP: 9, VSUSP: 10,
	VEOL: 11, VREPRINT: 12, VDISCARD: 13, VWERASE: 14, VLNEXT: 15,
	has: map[string]bool{"IXANY": true, "ONLCR": true, "TOSTOP": true},
}

var speeds = []speedInfo{
	{"0", 0, 0}, {"50", 50, 50}, {"75", 75, 75}, {"110", 110, 110}, {"134", 134, 134}, {"134.5", 134, 134},
	{"150", 150, 150}, {"200", 200, 200}, {"300", 300, 300}, {"600", 600, 600}, {"1200", 1200, 1200},
	{"1800", 1800, 1800}, {"2400", 2400, 2400}, {"4800", 4800, 4800}, {"9600", 9600, 9600},
	{"19200", 19200, 19200}, {"38400", 38400, 38400}, {"exta", 19200, 19200}, {"extb", 38400, 38400},
	{"57600", 57600, 57600}, {"115200", 115200, 115200}, {"230400", 230400, 230400},
}

func cfgetospeed(t *termios) uint64 { return t.ospeed }
func cfgetispeed(t *termios) uint64 { return t.ispeed }

func cfsetospeed(t *termios, speed uint64) error { t.ospeed = speed; return nil }
func cfsetispeed(t *termios, speed uint64) error { t.ispeed = speed; return nil }

func getAttr(uintptr) (termios, error)               { return termios{}, errUnsupported }
func setAttr(uintptr, *termios, bool) error          { return errUnsupported }
func getWinsize(uintptr) (rows, cols int, err error) { return 0, 0, errUnsupported }
func setWinsize(uintptr, int, int) error             { return errUnsupported }
func setExtproc(uintptr, bool) error                 { return errUnsupported }
func isEINVAL(error) bool                            { return false }
func clearNonblock(*os.File)                         {}
