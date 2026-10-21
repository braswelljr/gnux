//go:build freebsd

package stty

import "golang.org/x/sys/unix"

// FreeBSD: tcflag_t is 32 bits and c_cc has 20 slots.
const (
	ncc      = 20
	flagBits = 32
)

var c = consts{
	PARENB: unix.PARENB, PARODD: unix.PARODD, CS5: unix.CS5, CS6: unix.CS6, CS7: unix.CS7, CS8: unix.CS8,
	CSIZE: unix.CSIZE, HUPCL: unix.HUPCL, CSTOPB: unix.CSTOPB, CREAD: unix.CREAD, CLOCAL: unix.CLOCAL, CRTSCTS: unix.CRTSCTS,

	IGNBRK: unix.IGNBRK, BRKINT: unix.BRKINT, IGNPAR: unix.IGNPAR, PARMRK: unix.PARMRK, INPCK: unix.INPCK, ISTRIP: unix.ISTRIP,
	INLCR: unix.INLCR, IGNCR: unix.IGNCR, ICRNL: unix.ICRNL, IXON: unix.IXON, IXOFF: unix.IXOFF,
	IXANY: unix.IXANY, IMAXBEL: unix.IMAXBEL,

	OPOST: unix.OPOST, OCRNL: unix.OCRNL, ONLCR: unix.ONLCR, ONOCR: unix.ONOCR, ONLRET: unix.ONLRET,
	TABDLY: unix.TABDLY, TAB0: unix.TAB0, TAB3: unix.TAB3, OXTABS: unix.OXTABS,

	ISIG: unix.ISIG, ICANON: unix.ICANON, IEXTEN: unix.IEXTEN, ECHO: unix.ECHO, ECHOE: unix.ECHOE, ECHOK: unix.ECHOK,
	ECHONL: unix.ECHONL, NOFLSH: unix.NOFLSH, TOSTOP: unix.TOSTOP, ECHOPRT: unix.ECHOPRT,
	ECHOCTL: unix.ECHOCTL, ECHOKE: unix.ECHOKE, FLUSHO: unix.FLUSHO, EXTPROC: unix.EXTPROC,

	VINTR: unix.VINTR, VQUIT: unix.VQUIT, VERASE: unix.VERASE, VKILL: unix.VKILL, VEOF: unix.VEOF, VEOL: unix.VEOL,
	VEOL2: unix.VEOL2, VSTART: unix.VSTART, VSTOP: unix.VSTOP, VSUSP: unix.VSUSP, VDSUSP: unix.VDSUSP,
	VREPRINT: unix.VREPRINT, VWERASE: unix.VWERASE, VLNEXT: unix.VLNEXT, VDISCARD: unix.VDISCARD, VSTATUS: unix.VSTATUS,
	VMIN: unix.VMIN, VTIME: unix.VTIME,

	has: set("CRTSCTS", "IXANY", "IMAXBEL", "OCRNL", "ONLCR", "ONOCR", "ONLRET", "TABDLY", "TAB0", "TAB3",
		"IEXTEN", "TOSTOP", "ECHOPRT", "ECHOCTL", "ECHOKE", "FLUSHO", "EXTPROC", "TIOCEXT", "VEOL2", "VDSUSP", "VSTATUS"),
}

var speeds = []speedInfo{
	{"0", 0, 0}, {"50", 50, 50}, {"75", 75, 75}, {"110", 110, 110}, {"134", 134, 134}, {"134.5", 134, 134},
	{"150", 150, 150}, {"200", 200, 200}, {"300", 300, 300}, {"600", 600, 600}, {"1200", 1200, 1200},
	{"1800", 1800, 1800}, {"2400", 2400, 2400}, {"4800", 4800, 4800}, {"9600", 9600, 9600},
	{"19200", 19200, 19200}, {"38400", 38400, 38400}, {"exta", 19200, 19200}, {"extb", 38400, 38400},
	{"57600", 57600, 57600}, {"115200", 115200, 115200}, {"230400", 230400, 230400},
	{"460800", 460800, 460800}, {"921600", 921600, 921600},
}

func fromUnix(k *unix.Termios) termios {
	t := termios{iflag: uint64(k.Iflag), oflag: uint64(k.Oflag), cflag: uint64(k.Cflag), lflag: uint64(k.Lflag),
		ispeed: uint64(k.Ispeed), ospeed: uint64(k.Ospeed)}
	copy(t.cc[:], k.Cc[:])
	return t
}

func toUnix(t *termios) unix.Termios {
	k := unix.Termios{Iflag: uint32(t.iflag), Oflag: uint32(t.oflag), Cflag: uint32(t.cflag), Lflag: uint32(t.lflag),
		Ispeed: uint32(t.ispeed), Ospeed: uint32(t.ospeed)}
	copy(k.Cc[:], t.cc[:])
	return k
}
