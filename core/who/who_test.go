package who

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/utmp"
)

var (
	now     = time.Date(2026, 10, 12, 14, 0, 0, 0, time.UTC)
	records = []utmp.Entry{
		{Type: utmp.BootTime, Time: time.Date(2026, 10, 9, 10, 44, 0, 0, time.UTC)},
		{Type: utmp.RunLevel, PID: 'N'*256 + '5', Time: time.Date(2026, 10, 9, 10, 44, 5, 0, time.UTC)},
		{Type: utmp.LoginProcess, PID: 900, Line: "tty1", ID: "tty1", User: "LOGIN", Time: time.Date(2026, 10, 9, 10, 45, 0, 0, time.UTC)},
		{Type: utmp.UserProcess, PID: 1234, Line: "pts/0", User: "alice", Host: "10.0.0.5", Time: time.Date(2026, 10, 12, 9, 30, 0, 0, time.UTC)},
		{Type: utmp.UserProcess, PID: 1300, Line: "pts/1", User: "bob", Host: "laptop:0", Time: time.Date(2026, 10, 12, 13, 2, 0, 0, time.UTC)},
		{Type: utmp.DeadProcess, PID: 1100, Line: "pts/2", ID: "ts/2", Exit: 1, Time: time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC)},
	}
)

// fakeInfo is a terminal device as stat reports it.
type fakeInfo struct {
	os.FileInfo
	mode    os.FileMode
	modTime time.Time
}

func (f fakeInfo) Mode() os.FileMode  { return f.mode }
func (f fakeInfo) ModTime() time.Time { return f.modTime }
func (f fakeInfo) Sys() any           { return nil }

func run(t *testing.T, opts Options, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	w := New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}})
	w.Now = func() time.Time { return now }
	w.Location = time.UTC
	w.current = func() ([]utmp.Entry, error) { return records, nil }
	w.readFile = func(string) ([]utmp.Entry, error) { return nil, fs.ErrNotExist }
	w.ttyName = func(any) (string, bool) { return "/dev/pts/1", true }
	w.stat = func(p string) (os.FileInfo, error) {
		switch p {
		case "/dev/pts/0":
			return fakeInfo{mode: 0o620, modTime: now.Add(-90 * time.Minute)}, nil
		case "/dev/pts/1":
			return fakeInfo{mode: 0o600, modTime: now.Add(-20 * time.Second)}, nil
		}
		return nil, fs.ErrNotExist
	}
	err := w.Run(args)
	return out.String(), err
}

func TestDefault(t *testing.T) {
	out, err := run(t, Options{})
	want := "alice    pts/0        Oct 12 09:30 (10.0.0.5)\n" +
		"bob      pts/1        Oct 12 13:02 (laptop:0)\n"
	if err != nil || out != want {
		t.Fatalf("got\n%q\nwant\n%q", out, want)
	}
}

func TestUsersAndHeading(t *testing.T) {
	out, _ := run(t, Options{Users: true, Heading: true, Mesg: true})
	want := "NAME       LINE         TIME         IDLE          PID COMMENT\n" +
		"alice    + pts/0        Oct 12 09:30 01:30        1234 (10.0.0.5)\n" +
		"bob      - pts/1        Oct 12 13:02   .          1300 (laptop:0)\n"
	if out != want {
		t.Fatalf("got\n%s\nwant\n%s", out, want)
	}
}

func TestRecords(t *testing.T) {
	out, _ := run(t, Options{Boot: true})
	if out != "         system boot  Oct  9 10:44\n" {
		t.Fatalf("-b: %q", out)
	}
	out, _ = run(t, Options{RunLevel: true})
	if out != "         run-level 5  Oct  9 10:44                   last=S\n" {
		t.Fatalf("-r: %q", out)
	}
	out, _ = run(t, Options{Login: true})
	if out != "LOGIN    tty1         Oct  9 10:45               900 id=tty1\n" {
		t.Fatalf("-l: %q", out)
	}
	out, _ = run(t, Options{Dead: true})
	if out != "         pts/2        Oct 11 18:00              1100 id=ts/2  term=0 exit=1\n" {
		t.Fatalf("-d: %q", out)
	}
	out, _ = run(t, Options{All: true})
	if strings.Count(out, "\n") != 6 {
		t.Fatalf("-a:\n%s", out)
	}
}

func TestCountAndMyLine(t *testing.T) {
	out, _ := run(t, Options{Count: true})
	if out != "alice bob\n# users=2\n" {
		t.Fatalf("-q: %q", out)
	}
	out, _ = run(t, Options{MyLine: true})
	if out != "bob      pts/1        Oct 12 13:02 (laptop:0)\n" {
		t.Fatalf("-m: %q", out)
	}
	out, _ = run(t, Options{}, "am", "i")
	if out != "bob      pts/1        Oct 12 13:02 (laptop:0)\n" {
		t.Fatalf("who am i: %q", out)
	}
}

func TestErrors(t *testing.T) {
	_, err := run(t, Options{}, "/var/log/wtmp")
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("missing file: %v", err)
	}
	_, err = run(t, Options{}, "a", "b", "c")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "extra operand 'c'" {
		t.Fatalf("extra operand: %v", err)
	}
}

func TestLive(t *testing.T) {
	var out, errb bytes.Buffer
	err := New(Options{Boot: true}, core.Streams{Out: &out, Err: &errb}).Run(nil)
	if strings.Contains(errb.String(), "not supported") {
		t.Skip("no records here")
	}
	if err != nil || !strings.Contains(out.String(), "system boot") {
		t.Fatalf("%v %q %q", err, out.String(), errb.String())
	}
}
