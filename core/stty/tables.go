package stty

// termios is the terminal state, in a form the same on every platform: the
// four flag words, the line discipline (Linux), the control characters and
// the speeds as the system's speed_t values.
type termios struct {
	iflag, oflag, cflag, lflag uint64
	line                       byte
	cc                         [32]byte
	ispeed, ospeed             uint64 //nolint:unused // the speeds, where they have fields of their own (not on Linux)
	ibaud0                     bool   //nolint:unused // glibc: input speed 0, "same as output"; used on Linux
}

// consts are a platform's termios constants. A mode whose constant the
// platform does not define is left out of the tables, as GNU's #ifdefs do;
// has says which of the optional ones are defined (a defined constant may
// be zero, as CS5 and NL0 are).
type consts struct {
	PARENB, PARODD, CMSPAR, CS5, CS6, CS7, CS8, CSIZE, HUPCL, CSTOPB, CREAD, CLOCAL, CRTSCTS uint64

	IGNBRK, BRKINT, IGNPAR, PARMRK, INPCK, ISTRIP, INLCR, IGNCR, ICRNL, IXON, IXOFF, IUCLC, IXANY, IMAXBEL, IUTF8 uint64

	OPOST, OLCUC, OCRNL, ONLCR, ONOCR, ONLRET, OFILL, OFDEL uint64
	NLDLY, NL0, NL1, CRDLY, CR0, CR1, CR2, CR3              uint64
	TABDLY, TAB0, TAB1, TAB2, TAB3, OXTABS                  uint64
	BSDLY, BS0, BS1, VTDLY, VT0, VT1, FFDLY, FF0, FF1       uint64

	ISIG, ICANON, IEXTEN, ECHO, ECHOE, ECHOK, ECHONL, NOFLSH, XCASE, TOSTOP, ECHOPRT, ECHOCTL, ECHOKE, FLUSHO, EXTPROC uint64

	VINTR, VQUIT, VERASE, VKILL, VEOF, VEOL, VEOL2, VSWTCH, VSTART, VSTOP, VSUSP, VDSUSP, VREPRINT, VWERASE, VLNEXT, VDISCARD, VSTATUS, VMIN, VTIME int

	has map[string]bool
}

// buildModes is GNU stty's mode table for the platform's constants.
func buildModes(c *consts) []modeInfo {
	var m []modeInfo
	add := func(name string, k kind, flags int, bits, mask uint64) {
		m = append(m, modeInfo{name, k, flags, bits, mask})
	}
	has := func(name string) bool { return c.has[name] }

	add("parenb", control, rev, c.PARENB, 0)
	add("parodd", control, rev, c.PARODD, 0)
	if has("CMSPAR") {
		add("cmspar", control, rev, c.CMSPAR, 0)
	}
	add("cs5", control, 0, c.CS5, c.CSIZE)
	add("cs6", control, 0, c.CS6, c.CSIZE)
	add("cs7", control, 0, c.CS7, c.CSIZE)
	add("cs8", control, 0, c.CS8, c.CSIZE)
	add("hupcl", control, rev, c.HUPCL, 0)
	add("hup", control, rev|omit, c.HUPCL, 0)
	add("cstopb", control, rev, c.CSTOPB, 0)
	add("cread", control, saneSet|rev, c.CREAD, 0)
	add("clocal", control, rev, c.CLOCAL, 0)
	if has("CRTSCTS") {
		add("crtscts", control, rev, c.CRTSCTS, 0)
	}

	add("ignbrk", input, saneUnset|rev, c.IGNBRK, 0)
	add("brkint", input, saneSet|rev, c.BRKINT, 0)
	add("ignpar", input, rev, c.IGNPAR, 0)
	add("parmrk", input, rev, c.PARMRK, 0)
	add("inpck", input, rev, c.INPCK, 0)
	add("istrip", input, rev, c.ISTRIP, 0)
	add("inlcr", input, saneUnset|rev, c.INLCR, 0)
	add("igncr", input, saneUnset|rev, c.IGNCR, 0)
	add("icrnl", input, saneSet|rev, c.ICRNL, 0)
	add("ixon", input, rev, c.IXON, 0)
	add("ixoff", input, saneUnset|rev, c.IXOFF, 0)
	add("tandem", input, rev|omit, c.IXOFF, 0)
	if has("IUCLC") {
		add("iuclc", input, saneUnset|rev, c.IUCLC, 0)
	}
	if has("IXANY") {
		add("ixany", input, saneUnset|rev, c.IXANY, 0)
	}
	if has("IMAXBEL") {
		add("imaxbel", input, saneSet|rev, c.IMAXBEL, 0)
	}
	if has("IUTF8") {
		add("iutf8", input, saneUnset|rev, c.IUTF8, 0)
	}

	add("opost", output, saneSet|rev, c.OPOST, 0)
	for _, o := range []struct {
		name, constName string
		flags           int
		bits            uint64
	}{
		{"olcuc", "OLCUC", saneUnset | rev, c.OLCUC},
		{"ocrnl", "OCRNL", saneUnset | rev, c.OCRNL},
		{"onlcr", "ONLCR", saneSet | rev, c.ONLCR},
		{"onocr", "ONOCR", saneUnset | rev, c.ONOCR},
		{"onlret", "ONLRET", saneUnset | rev, c.ONLRET},
		{"ofill", "OFILL", saneUnset | rev, c.OFILL},
		{"ofdel", "OFDEL", saneUnset | rev, c.OFDEL},
	} {
		if has(o.constName) {
			add(o.name, output, o.flags, o.bits, 0)
		}
	}
	if has("NLDLY") {
		add("nl1", output, saneUnset, c.NL1, c.NLDLY)
		add("nl0", output, saneSet, c.NL0, c.NLDLY)
	}
	if has("CRDLY") {
		add("cr3", output, saneUnset, c.CR3, c.CRDLY)
		add("cr2", output, saneUnset, c.CR2, c.CRDLY)
		add("cr1", output, saneUnset, c.CR1, c.CRDLY)
		add("cr0", output, saneSet, c.CR0, c.CRDLY)
	}
	switch {
	case has("TABDLY"):
		if has("TAB3") {
			add("tab3", output, saneUnset, c.TAB3, c.TABDLY)
		}
		if has("TAB2") {
			add("tab2", output, saneUnset, c.TAB2, c.TABDLY)
		}
		if has("TAB1") {
			add("tab1", output, saneUnset, c.TAB1, c.TABDLY)
		}
		if has("TAB0") {
			add("tab0", output, saneSet, c.TAB0, c.TABDLY)
		}
	case has("OXTABS"):
		add("tab3", output, saneUnset, c.OXTABS, 0)
	}
	if has("BSDLY") {
		add("bs1", output, saneUnset, c.BS1, c.BSDLY)
		add("bs0", output, saneSet, c.BS0, c.BSDLY)
	}
	if has("VTDLY") {
		add("vt1", output, saneUnset, c.VT1, c.VTDLY)
		add("vt0", output, saneSet, c.VT0, c.VTDLY)
	}
	if has("FFDLY") {
		add("ff1", output, saneUnset, c.FF1, c.FFDLY)
		add("ff0", output, saneSet, c.FF0, c.FFDLY)
	}

	add("isig", local, saneSet|rev, c.ISIG, 0)
	add("icanon", local, saneSet|rev, c.ICANON, 0)
	if has("IEXTEN") {
		add("iexten", local, saneSet|rev, c.IEXTEN, 0)
	}
	add("echo", local, saneSet|rev, c.ECHO, 0)
	add("echoe", local, saneSet|rev, c.ECHOE, 0)
	add("crterase", local, rev|omit, c.ECHOE, 0)
	add("echok", local, saneSet|rev, c.ECHOK, 0)
	add("echonl", local, saneUnset|rev, c.ECHONL, 0)
	add("noflsh", local, saneUnset|rev, c.NOFLSH, 0)
	if has("XCASE") {
		add("xcase", local, saneUnset|rev, c.XCASE, 0)
	}
	if has("TOSTOP") {
		add("tostop", local, saneUnset|rev, c.TOSTOP, 0)
	}
	if has("ECHOPRT") {
		add("echoprt", local, saneUnset|rev, c.ECHOPRT, 0)
		add("prterase", local, rev|omit, c.ECHOPRT, 0)
	}
	if has("ECHOCTL") {
		add("echoctl", local, saneSet|rev, c.ECHOCTL, 0)
		add("ctlecho", local, rev|omit, c.ECHOCTL, 0)
	}
	if has("ECHOKE") {
		add("echoke", local, saneSet|rev, c.ECHOKE, 0)
		add("crtkill", local, rev|omit, c.ECHOKE, 0)
	}
	if has("FLUSHO") {
		add("flusho", local, saneUnset|rev, c.FLUSHO, 0)
	}
	switch {
	case has("TIOCEXT"):
		add("extproc", local, saneUnset|rev|noSetAttr, c.EXTPROC, 0)
	case has("EXTPROC"):
		add("extproc", local, saneUnset|rev, c.EXTPROC, 0)
	}

	for _, name := range []string{"evenp", "parity", "oddp", "nl"} {
		add(name, combination, rev|omit, 0, 0)
	}
	add("ek", combination, omit, 0, 0)
	add("sane", combination, omit, 0, 0)
	for _, name := range []string{"cooked", "raw", "pass8", "litout", "cbreak"} {
		add(name, combination, rev|omit, 0, 0)
	}
	if has("IXANY") {
		add("decctlq", combination, rev|omit, 0, 0)
	}
	if has("TABDLY") || has("OXTABS") {
		add("tabs", combination, rev|omit, 0, 0)
	}
	if has("XCASE") && has("IUCLC") && has("OLCUC") {
		add("lcase", combination, rev|omit, 0, 0)
		add("LCASE", combination, rev|omit, 0, 0)
	}
	add("crt", combination, omit, 0, 0)
	add("dec", combination, omit, 0, 0)
	return m
}

func ctrl(ch byte) byte { return ch & 0x1f }

// buildControls is GNU stty's control character table: names, the values
// 'sane' sets, and their slots.
func buildControls(c *consts, disabled byte) []controlInfo {
	has := func(name string) bool { return c.has[name] }
	cs := []controlInfo{
		{"intr", ctrl('c'), c.VINTR},
		{"quit", 28, c.VQUIT},
		{"erase", 127, c.VERASE},
		{"kill", ctrl('u'), c.VKILL},
		{"eof", ctrl('d'), c.VEOF},
		{"eol", disabled, c.VEOL},
	}
	if has("VEOL2") {
		cs = append(cs, controlInfo{"eol2", disabled, c.VEOL2})
	}
	if has("VSWTC") {
		cs = append(cs, controlInfo{"swtch", disabled, c.VSWTCH})
	} else {
		// Where there is no VSWTCH, swtch is another name for susp.
		cs = append(cs, controlInfo{"swtch", ctrl('z'), c.VSUSP})
	}
	cs = append(cs,
		controlInfo{"start", ctrl('q'), c.VSTART},
		controlInfo{"stop", ctrl('s'), c.VSTOP},
		controlInfo{"susp", ctrl('z'), c.VSUSP},
	)
	if has("VDSUSP") {
		cs = append(cs, controlInfo{"dsusp", ctrl('y'), c.VDSUSP})
	}
	cs = append(cs,
		controlInfo{"rprnt", ctrl('r'), c.VREPRINT},
		controlInfo{"werase", ctrl('w'), c.VWERASE},
		controlInfo{"lnext", ctrl('v'), c.VLNEXT},
		controlInfo{"flush", ctrl('o'), c.VDISCARD},
		controlInfo{"discard", ctrl('o'), c.VDISCARD},
	)
	if has("VSTATUS") {
		cs = append(cs, controlInfo{"status", ctrl('t'), c.VSTATUS})
	}
	return append(cs, controlInfo{"min", 1, c.VMIN}, controlInfo{"time", 0, c.VTIME})
}

// The tables for this platform.
var (
	modes    = buildModes(&c)
	controls = buildControls(&c, vdisable)
)
