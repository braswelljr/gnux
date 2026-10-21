//go:build linux

package stty

import (
	"errors"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// glibc's termios: tcflag_t is 32 bits, c_cc has 32 slots (the kernel fills
// 19), the line discipline is in c_line, and _POSIX_VDISABLE is 0.
const (
	kernelState = 0
	ncc         = 32
	flagBits    = 32
	hasLine     = true
	vdisable    = 0
	swtchIsSusp = false
	nonblock    = syscall.O_NONBLOCK
)

var c = consts{
	PARENB: unix.PARENB, PARODD: unix.PARODD, CMSPAR: unix.CMSPAR, CS5: unix.CS5, CS6: unix.CS6, CS7: unix.CS7, CS8: unix.CS8,
	CSIZE: unix.CSIZE, HUPCL: unix.HUPCL, CSTOPB: unix.CSTOPB, CREAD: unix.CREAD, CLOCAL: unix.CLOCAL, CRTSCTS: unix.CRTSCTS,

	IGNBRK: unix.IGNBRK, BRKINT: unix.BRKINT, IGNPAR: unix.IGNPAR, PARMRK: unix.PARMRK, INPCK: unix.INPCK, ISTRIP: unix.ISTRIP,
	INLCR: unix.INLCR, IGNCR: unix.IGNCR, ICRNL: unix.ICRNL, IXON: unix.IXON, IXOFF: unix.IXOFF, IUCLC: unix.IUCLC,
	IXANY: unix.IXANY, IMAXBEL: unix.IMAXBEL, IUTF8: unix.IUTF8,

	OPOST: unix.OPOST, OLCUC: unix.OLCUC, OCRNL: unix.OCRNL, ONLCR: unix.ONLCR, ONOCR: unix.ONOCR, ONLRET: unix.ONLRET,
	OFILL: unix.OFILL, OFDEL: unix.OFDEL,
	NLDLY: unix.NLDLY, NL0: unix.NL0, NL1: unix.NL1, CRDLY: unix.CRDLY, CR0: unix.CR0, CR1: unix.CR1, CR2: unix.CR2, CR3: unix.CR3,
	TABDLY: unix.TABDLY, TAB0: unix.TAB0, TAB1: unix.TAB1, TAB2: unix.TAB2, TAB3: unix.TAB3,
	BSDLY: unix.BSDLY, BS0: unix.BS0, BS1: unix.BS1, VTDLY: unix.VTDLY, VT0: unix.VT0, VT1: unix.VT1,
	FFDLY: unix.FFDLY, FF0: unix.FF0, FF1: unix.FF1,

	ISIG: unix.ISIG, ICANON: unix.ICANON, IEXTEN: unix.IEXTEN, ECHO: unix.ECHO, ECHOE: unix.ECHOE, ECHOK: unix.ECHOK,
	ECHONL: unix.ECHONL, NOFLSH: unix.NOFLSH, XCASE: unix.XCASE, TOSTOP: unix.TOSTOP, ECHOPRT: unix.ECHOPRT,
	ECHOCTL: unix.ECHOCTL, ECHOKE: unix.ECHOKE, FLUSHO: unix.FLUSHO, EXTPROC: unix.EXTPROC,

	VINTR: unix.VINTR, VQUIT: unix.VQUIT, VERASE: unix.VERASE, VKILL: unix.VKILL, VEOF: unix.VEOF, VEOL: unix.VEOL,
	VEOL2: unix.VEOL2, VSWTCH: unix.VSWTC, VSTART: unix.VSTART, VSTOP: unix.VSTOP, VSUSP: unix.VSUSP,
	VREPRINT: unix.VREPRINT, VWERASE: unix.VWERASE, VLNEXT: unix.VLNEXT, VDISCARD: unix.VDISCARD,
	VMIN: unix.VMIN, VTIME: unix.VTIME,

	has: set("CMSPAR", "CRTSCTS", "IUCLC", "IXANY", "IMAXBEL", "IUTF8", "OLCUC", "OCRNL", "ONLCR", "ONOCR", "ONLRET",
		"OFILL", "OFDEL", "NLDLY", "CRDLY", "TABDLY", "TAB0", "TAB1", "TAB2", "TAB3", "BSDLY", "VTDLY", "FFDLY",
		"IEXTEN", "XCASE", "TOSTOP", "ECHOPRT", "ECHOCTL", "ECHOKE", "FLUSHO", "EXTPROC", "VEOL2", "VSWTC"),
}

func set(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// speeds are the B constants, which on Linux are codes in c_cflag, not
// numbers of baud.
var speeds = []speedInfo{
	{"0", unix.B0, 0}, {"50", unix.B50, 50}, {"75", unix.B75, 75}, {"110", unix.B110, 110},
	{"134", unix.B134, 134}, {"134.5", unix.B134, 134}, {"150", unix.B150, 150}, {"200", unix.B200, 200},
	{"300", unix.B300, 300}, {"600", unix.B600, 600}, {"1200", unix.B1200, 1200}, {"1800", unix.B1800, 1800},
	{"2400", unix.B2400, 2400}, {"4800", unix.B4800, 4800}, {"9600", unix.B9600, 9600},
	{"19200", unix.B19200, 19200}, {"38400", unix.B38400, 38400}, {"exta", unix.B19200, 19200},
	{"extb", unix.B38400, 38400}, {"57600", unix.B57600, 57600}, {"115200", unix.B115200, 115200},
	{"230400", unix.B230400, 230400}, {"460800", unix.B460800, 460800}, {"500000", unix.B500000, 500000},
	{"576000", unix.B576000, 576000}, {"921600", unix.B921600, 921600}, {"1000000", unix.B1000000, 1000000},
	{"1152000", unix.B1152000, 1152000}, {"1500000", unix.B1500000, 1500000}, {"2000000", unix.B2000000, 2000000},
	{"2500000", unix.B2500000, 2500000}, {"3000000", unix.B3000000, 3000000}, {"3500000", unix.B3500000, 3500000},
	{"4000000", unix.B4000000, 4000000},
}

const cbaud = unix.CBAUD | unix.CBAUDEX

// glibc keeps both speeds in c_cflag; an input speed of 0 means "the same as
// the output speed".
func cfgetospeed(t *termios) uint64 { return t.cflag & cbaud }

func cfgetispeed(t *termios) uint64 {
	if t.ibaud0 {
		return 0
	}
	return t.cflag & cbaud
}

func cfsetospeed(t *termios, speed uint64) error {
	t.cflag = t.cflag&^cbaud | speed
	return nil
}

func cfsetispeed(t *termios, speed uint64) error {
	if speed == 0 {
		t.ibaud0 = true
		return nil
	}
	t.ibaud0 = false
	t.cflag = t.cflag&^cbaud | speed
	return nil
}

func getAttr(fd uintptr) (termios, error) {
	k, err := unix.IoctlGetTermios(int(fd), unix.TCGETS)
	if err != nil {
		return termios{}, err
	}
	t := termios{iflag: uint64(k.Iflag), oflag: uint64(k.Oflag), cflag: uint64(k.Cflag), lflag: uint64(k.Lflag), line: k.Line}
	copy(t.cc[:], k.Cc[:])
	return t, nil
}

func setAttr(fd uintptr, t *termios, drain bool) error {
	k := unix.Termios{Iflag: uint32(t.iflag), Oflag: uint32(t.oflag), Cflag: uint32(t.cflag), Lflag: uint32(t.lflag), Line: t.line}
	copy(k.Cc[:], t.cc[:])
	req := uint(unix.TCSETS)
	if drain {
		req = unix.TCSETSW
	}
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

// setExtproc is not needed: Linux sets extproc through tcsetattr.
func setExtproc(uintptr, bool) error { return errors.ErrUnsupported }

func isEINVAL(err error) bool { return errors.Is(err, syscall.EINVAL) }

// clearNonblock undoes the O_NONBLOCK a -F device is opened with, so that
// opening a serial line does not wait for its carrier.
func clearNonblock(f *os.File) { _ = unix.SetNonblock(int(f.Fd()), false) }
