//go:build darwin

package stty

import "golang.org/x/sys/unix"

// macOS: tcflag_t is unsigned long and c_cc has 20 slots.
const (
	ncc      = 20
	flagBits = 64
)

var c = consts{
	PARENB: unix.PARENB, PARODD: unix.PARODD, CS5: unix.CS5, CS6: unix.CS6, CS7: unix.CS7, CS8: unix.CS8,
	CSIZE: unix.CSIZE, HUPCL: unix.HUPCL, CSTOPB: unix.CSTOPB, CREAD: unix.CREAD, CLOCAL: unix.CLOCAL, CRTSCTS: unix.CRTSCTS,

	IGNBRK: unix.IGNBRK, BRKINT: unix.BRKINT, IGNPAR: unix.IGNPAR, PARMRK: unix.PARMRK, INPCK: unix.INPCK, ISTRIP: unix.ISTRIP,
	INLCR: unix.INLCR, IGNCR: unix.IGNCR, ICRNL: unix.ICRNL, IXON: unix.IXON, IXOFF: unix.IXOFF,
	IXANY: unix.IXANY, IMAXBEL: unix.IMAXBEL, IUTF8: unix.IUTF8,

	OPOST: unix.OPOST, OCRNL: unix.OCRNL, ONLCR: unix.ONLCR, ONOCR: unix.ONOCR, ONLRET: unix.ONLRET,
	OFILL: unix.OFILL, OFDEL: unix.OFDEL,
	NLDLY: unix.NLDLY, NL0: unix.NL0, NL1: unix.NL1, CRDLY: unix.CRDLY, CR0: unix.CR0, CR1: unix.CR1, CR2: unix.CR2, CR3: unix.CR3,
	TABDLY: unix.TABDLY, TAB0: unix.TAB0, TAB1: unix.TAB1, TAB2: unix.TAB2, TAB3: unix.TAB3, OXTABS: unix.OXTABS,
	BSDLY: unix.BSDLY, BS0: unix.BS0, BS1: unix.BS1, VTDLY: unix.VTDLY, VT0: unix.VT0, VT1: unix.VT1,
	FFDLY: unix.FFDLY, FF0: unix.FF0, FF1: unix.FF1,

	ISIG: unix.ISIG, ICANON: unix.ICANON, IEXTEN: unix.IEXTEN, ECHO: unix.ECHO, ECHOE: unix.ECHOE, ECHOK: unix.ECHOK,
	ECHONL: unix.ECHONL, NOFLSH: unix.NOFLSH, TOSTOP: unix.TOSTOP, ECHOPRT: unix.ECHOPRT,
	ECHOCTL: unix.ECHOCTL, ECHOKE: unix.ECHOKE, FLUSHO: unix.FLUSHO, EXTPROC: unix.EXTPROC,

	VINTR: unix.VINTR, VQUIT: unix.VQUIT, VERASE: unix.VERASE, VKILL: unix.VKILL, VEOF: unix.VEOF, VEOL: unix.VEOL,
	VEOL2: unix.VEOL2, VSTART: unix.VSTART, VSTOP: unix.VSTOP, VSUSP: unix.VSUSP, VDSUSP: unix.VDSUSP,
	VREPRINT: unix.VREPRINT, VWERASE: unix.VWERASE, VLNEXT: unix.VLNEXT, VDISCARD: unix.VDISCARD, VSTATUS: unix.VSTATUS,
	VMIN: unix.VMIN, VTIME: unix.VTIME,

	has: set("CRTSCTS", "IXANY", "IMAXBEL", "IUTF8", "OCRNL", "ONLCR", "ONOCR", "ONLRET", "OFILL", "OFDEL",
		"NLDLY", "CRDLY", "TABDLY", "TAB0", "TAB1", "TAB2", "TAB3", "BSDLY", "VTDLY", "FFDLY", "IEXTEN", "TOSTOP",
		"ECHOPRT", "ECHOCTL", "ECHOKE", "FLUSHO", "EXTPROC", "TIOCEXT", "VEOL2", "VDSUSP", "VSTATUS"),
}

var speeds = []speedInfo{
	{"0", 0, 0}, {"50", 50, 50}, {"75", 75, 75}, {"110", 110, 110}, {"134", 134, 134}, {"134.5", 134, 134},
	{"150", 150, 150}, {"200", 200, 200}, {"300", 300, 300}, {"600", 600, 600}, {"1200", 1200, 1200},
	{"1800", 1800, 1800}, {"2400", 2400, 2400}, {"4800", 4800, 4800}, {"9600", 9600, 9600},
	{"19200", 19200, 19200}, {"38400", 38400, 38400}, {"exta", 19200, 19200}, {"extb", 38400, 38400},
	{"57600", 57600, 57600}, {"115200", 115200, 115200}, {"230400", 230400, 230400},
}

func fromUnix(k *unix.Termios) termios {
	t := termios{iflag: k.Iflag, oflag: k.Oflag, cflag: k.Cflag, lflag: k.Lflag, ispeed: k.Ispeed, ospeed: k.Ospeed}
	copy(t.cc[:], k.Cc[:])
	return t
}

func toUnix(t *termios) unix.Termios {
	k := unix.Termios{Iflag: t.iflag, Oflag: t.oflag, Cflag: t.cflag, Lflag: t.lflag, Ispeed: t.ispeed, Ospeed: t.ospeed}
	copy(k.Cc[:], t.cc[:])
	return k
}
