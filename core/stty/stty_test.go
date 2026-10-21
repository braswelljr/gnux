package stty

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
)

func newEngine() (*Stty, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	s := New(Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	s.Getenv = func(string) string { return "" }
	s.maxCol = 80
	s.lastIn, s.lastOut = -1, -1
	return s, &out, &errb
}

func sane() termios {
	var t termios
	saneMode(&t)
	_ = cfsetospeed(&t, speeds[len(speeds)-1].speed)
	_ = cfsetispeed(&t, speeds[len(speeds)-1].speed)
	return t
}

func TestVisible(t *testing.T) {
	for ch, want := range map[byte]string{3: "^C", 'a': "a", 127: "^?", 128 + 'a': "M-a", 128 + 3: "M-^C", 255: "M-^?"} {
		if ch == vdisable {
			continue
		}
		if got := visible(ch); got != want {
			t.Errorf("visible(%d) = %q, want %q", ch, got, want)
		}
	}
	if got := visible(vdisable); got != "<undef>" {
		t.Errorf("visible(vdisable) = %q", got)
	}
}

func TestApplyModes(t *testing.T) {
	s, _, _ := newEngine()
	m := sane()
	if err := s.apply(false, []string{"-echo", "-icanon", "min", "0", "time", "5", "intr", "^X", "erase", "^?", "kill", "undef", "eof", "04", "quit", "q"}, &m); err != nil {
		t.Fatal(err)
	}
	if m.lflag&c.ECHO != 0 || m.lflag&c.ICANON != 0 {
		t.Errorf("echo or icanon still on: %#x", m.lflag)
	}
	if m.cc[c.VMIN] != 0 || m.cc[c.VTIME] != 5 || m.cc[c.VINTR] != 0x18 || m.cc[c.VERASE] != 127 ||
		m.cc[c.VKILL] != vdisable || m.cc[c.VEOF] != 4 || m.cc[c.VQUIT] != 'q' {
		t.Errorf("control characters = %v", m.cc)
	}
	if !s.setAttrs {
		t.Error("settings not marked to be written")
	}
	if err := s.apply(false, []string{"raw"}, &m); err != nil || m.iflag != 0 || m.oflag&c.OPOST != 0 || m.cc[c.VMIN] != 1 {
		t.Errorf("raw: %v %+v", err, m)
	}
	if err := s.apply(false, []string{"sane"}, &m); err != nil || m != sane() {
		t.Errorf("sane: %v\n%+v\n%+v", err, m, sane())
	}
	if err := s.apply(false, []string{"cs7", "parenb"}, &m); err != nil || m.cflag&c.CSIZE != c.CS7 || m.cflag&c.PARENB == 0 {
		t.Errorf("cs7 parenb: %v %#x", err, m.cflag)
	}
	if err := s.apply(false, []string{"-evenp"}, &m); err != nil || m.cflag&c.CSIZE != c.CS8 || m.cflag&c.PARENB != 0 {
		t.Errorf("-evenp: %v %#x", err, m.cflag)
	}
}

func TestApplyMistakes(t *testing.T) {
	tests := []struct {
		args  []string
		msg   string
		usage bool
	}{
		{[]string{"bogus"}, "invalid argument 'bogus'", true},
		{[]string{"-cs8"}, "invalid argument '-cs8'", true},
		{[]string{"intr"}, "missing argument to 'intr'", true},
		{[]string{"ispeed", "x"}, "invalid ispeed 'x'", true},
		{[]string{"min", "x"}, "invalid integer argument: 'x'", false},
		{[]string{"min", "256"}, "invalid integer argument: '256': Numerical result out of range", false},
	}
	for _, tt := range tests {
		s, _, _ := newEngine()
		m := sane()
		err := s.apply(true, tt.args, &m)
		var u *usageError
		var f *fatal
		switch {
		case tt.usage && errors.As(err, &u) && u.msg == tt.msg:
		case !tt.usage && errors.As(err, &f) && f.msg == tt.msg:
		default:
			t.Errorf("%v: err = %v, want %q", tt.args, err, tt.msg)
		}
	}
}

func TestRecoverableRoundTrip(t *testing.T) {
	s, out, _ := newEngine()
	m := sane()
	m.cc[c.VINTR] = 0x18
	m.lflag &^= c.ECHO
	s.displayRecoverable(&m)
	saved := strings.TrimSpace(out.String())
	if got := strings.Count(saved, ":"); got != 3+ncc {
		t.Fatalf("-g printed %d fields: %q", got+1, saved)
	}
	var back termios
	if !recoverMode(saved, &back) || back.iflag != m.iflag || back.oflag != m.oflag || back.cflag&^cbaudMask() != m.cflag&^cbaudMask() || back.lflag != m.lflag || back.cc != m.cc {
		t.Fatalf("recovered %+v from %q, want %+v", back, saved, m)
	}
	if recoverMode("1:2:3", &back) {
		t.Fatal("a short saved state was accepted")
	}
}

// cbaudMask is the part of c_cflag that holds the speed, where it does.
func cbaudMask() uint64 {
	var t termios
	_ = cfsetospeed(&t, speeds[len(speeds)-1].speed)
	return t.cflag
}

func TestDisplayChanged(t *testing.T) {
	s, out, _ := newEngine()
	m := sane()
	s.displayChanged(&m)
	want := "speed " + speeds[len(speeds)-1].name + " baud;"
	if hasLine {
		want += " line = 0;"
	}
	if out.String() != want+"\n" {
		t.Fatalf("a sane terminal shows %q, want %q", out.String(), want+"\n")
	}
	out.Reset()
	m.lflag &^= c.ECHO | c.ICANON
	m.cc[c.VINTR] = 0x18
	s.displayChanged(&m)
	lines := strings.Split(out.String(), "\n")
	if len(lines) < 4 || lines[1] != "intr = ^X; min = 1; time = 0;" || !strings.Contains(out.String(), "-icanon -echo") {
		t.Fatalf("changed settings shown as %q", out.String())
	}
}

func TestDisplayAllWraps(t *testing.T) {
	s, out, _ := newEngine()
	s.maxCol = 40
	m := sane()
	s.displayAll(&m)
	for _, line := range strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n") {
		if len(line) > 40 {
			t.Errorf("line longer than the screen: %q", line)
		}
	}
	if !strings.Contains(out.String(), "intr = ^C;") || !strings.Contains(out.String(), " echo ") && !strings.Contains(out.String(), "\necho ") {
		t.Errorf("-a shows %q", out.String())
	}
}

func TestNotATerminal(t *testing.T) {
	s, _, errb := newEngine()
	if err := s.Run(nil); !errors.Is(err, core.ErrFailed) || errb.String() != "sttyx: 'standard input': Inappropriate ioctl for device\n" {
		t.Fatalf("err=%v stderr=%q", err, errb)
	}
}
