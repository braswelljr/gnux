package pinky_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/utmp"
	"github.com/braswelljr/gnux/core/pinky"
)

type fileInfo struct {
	os.FileInfo
	mode os.FileMode
	mod  time.Time
}

func (f fileInfo) Mode() os.FileMode  { return f.mode }
func (f fileInfo) ModTime() time.Time { return f.mod }
func (f fileInfo) Sys() any           { return nil }

var now = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

func newEngine(t *testing.T, opts pinky.Options, home string) (*pinky.Pinky, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var out, errb bytes.Buffer
	p := pinky.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	p.Now = func() time.Time { return now }
	p.Location = time.UTC
	p.Current = func() ([]utmp.Entry, error) {
		return []utmp.Entry{
			{Type: utmp.BootTime, Time: now.Add(-48 * time.Hour)},
			{Type: utmp.UserProcess, User: "alice", Line: "pts/0", Host: "10.0.0.7", Time: now.Add(-3 * time.Hour)},
			{Type: utmp.UserProcess, User: "bob", Line: "tty1", Time: now.Add(-30 * time.Minute)},
			{Type: utmp.UserProcess, User: "carolineabc", Line: "pts/12", Host: "host:0", Time: now.Add(-time.Minute)},
		}, nil
	}
	p.Stat = func(name string) (os.FileInfo, error) {
		switch name {
		case "/dev/pts/0":
			return fileInfo{mode: 0o600, mod: now.Add(-12 * time.Minute)}, nil
		case "/dev/tty1":
			return fileInfo{mode: 0o620, mod: now.Add(-30 * time.Second)}, nil
		case "/dev/pts/12":
			return fileInfo{mode: 0o620, mod: now.Add(-50 * time.Hour)}, nil
		}
		return nil, os.ErrNotExist
	}
	p.LookupUser = func(name string) (*pinky.Account, error) {
		switch name {
		case "alice":
			return &pinky.Account{Username: "alice", Gecos: "Alice & Liddell", Home: home, Shell: "/bin/bash"}, nil
		case "bob":
			return &pinky.Account{Username: "bob", Gecos: "Robert Tables the Third of His Name"}, nil
		}
		return nil, errors.New("unknown user")
	}
	return p, &out, &errb
}

func TestShort(t *testing.T) {
	p, out, _ := newEngine(t, pinky.Options{}, "")
	if err := p.Run(nil); err != nil {
		t.Fatal(err)
	}
	want := "Login    Name                 TTY      Idle   When         Where\n" +
		"alice    Alice Alice Liddell *pts/0    00:12  Oct 20 09:00 10.0.0.7\n" +
		"bob      Robert Tables the T  tty1            Oct 20 11:30\n" +
		"carolineabc                 ???  pts/12   2d     Oct 20 11:59 host:0\n"
	if out.String() != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", out, want)
	}
}

func TestShortSelectedAndOmitted(t *testing.T) {
	p, out, _ := newEngine(t, pinky.Options{NoHeading: true, NoFullName: true, NoWhere: true, NoIdle: true}, "")
	if err := p.Run([]string{"bob", "nobody"}); err != nil {
		t.Fatal(err)
	}
	if want := "bob       tty1     Oct 20 11:30\n"; out.String() != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

func TestLong(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".plan"), []byte("Lunch at noon.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".project"), []byte("gnux\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, out, _ := newEngine(t, pinky.Options{Long: true}, home)
	if err := p.Run([]string{"alice", "zed"}); err != nil {
		t.Fatal(err)
	}
	want := "Login name: alice                       In real life:  Alice Alice Liddell\n" +
		"Directory: " + home + strings.Repeat(" ", max(29-len(home), 0)) + "Shell:  /bin/bash\n" +
		"Project: gnux\nPlan:\nLunch at noon.\n\n" +
		"Login name: zed                         In real life:  ???\n"
	if out.String() != want {
		t.Fatalf("stdout =\n%q\nwant\n%q", out, want)
	}
}

func TestLongNeedsAUser(t *testing.T) {
	p, _, _ := newEngine(t, pinky.Options{Long: true}, "")
	var usage *core.UsageError
	if err := p.Run(nil); !errors.As(err, &usage) {
		t.Fatalf("err = %v", err)
	}
}
