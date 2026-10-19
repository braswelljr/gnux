package nice_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/nice"
)

// engine builds a nice engine on buffers whose niceness starts at cur and
// records what it was set to.
func engine(t *testing.T, opts nice.Options, cur int, setErr error) (n *nice.Nice, out, errb *bytes.Buffer, set *int) {
	t.Helper()
	out, errb = &bytes.Buffer{}, &bytes.Buffer{}
	n = nice.New(opts, core.Streams{In: strings.NewReader("input\n"), Out: out, Err: errb})
	set = new(int)
	*set = -99
	n.GetNiceness = func() (int, error) { return cur, nil }
	n.SetNiceness = func(pid, niceness int) error {
		if pid <= 0 {
			t.Errorf("SetNiceness for pid %d", pid)
		}
		*set = niceness
		return setErr
	}
	return n, out, errb, set
}

func status(err error) int {
	if err == nil {
		return 0
	}
	var exit *core.ExitError
	if errors.As(err, &exit) {
		return exit.Code
	}
	return -1
}

func helperArgv(t *testing.T, args ...string) []string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Skip(err)
	}
	return append([]string{exe}, args...)
}

func TestPrintNiceness(t *testing.T) {
	n, out, _, _ := engine(t, nice.Options{}, 5, nil)
	if err := n.Run(nil); err != nil || out.String() != "5\n" {
		t.Fatalf("out = %q, err = %v", out.String(), err)
	}

	n, _, errb, _ := engine(t, nice.Options{}, 0, nil)
	n.GetNiceness = func() (int, error) { return 0, fs.ErrPermission }
	if err := n.Run(nil); status(err) != nice.ExitCanceled || errb.String() != "nicex: cannot get niceness: Permission denied\n" {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
}

func TestAdjustmentNeedsACommand(t *testing.T) {
	n, _, _, _ := engine(t, nice.Options{Adjust: true, Adjustment: 5}, 0, nil)
	var usage *core.UsageError
	if err := n.Run(nil); !errors.As(err, &usage) || usage.Msg != "a command must be given with an adjustment" {
		t.Fatalf("err = %v", err)
	}
}

func TestTargetNiceness(t *testing.T) {
	tests := []struct {
		name string
		opts nice.Options
		cur  int
		want int
	}{
		{"default adjustment", nice.Options{}, 0, 10},
		{"default from 5", nice.Options{}, 5, 15},
		{"explicit", nice.Options{Adjust: true, Adjustment: 3}, 2, 5},
		{"explicit zero", nice.Options{Adjust: true, Adjustment: 0}, 4, 4},
		{"capped at 19", nice.Options{Adjust: true, Adjustment: 100}, 10, 19},
		{"floored at -20", nice.Options{Adjust: true, Adjustment: -100}, 0, -20},
		{"huge does not overflow", nice.Options{Adjust: true, Adjustment: int(^uint(0) >> 1)}, 19, 19},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, _, _, set := engine(t, tt.opts, tt.cur, nil)
			if err := n.Run(helperArgv(t, "exit", "0")); err != nil {
				t.Fatalf("Run: %v", err)
			}
			if *set != tt.want {
				t.Fatalf("niceness set to %d; want %d", *set, tt.want)
			}
		})
	}
}

func TestCommandStreamsAndStatus(t *testing.T) {
	n, out, _, _ := engine(t, nice.Options{}, 0, nil)
	if err := n.Run(helperArgv(t, "cat")); err != nil || out.String() != "input\n" {
		t.Fatalf("out = %q, err = %v", out.String(), err)
	}
	n, _, errb, _ := engine(t, nice.Options{}, 0, nil)
	if err := n.Run(helperArgv(t, "stderr", "oops")); err != nil || errb.String() != "oops\n" {
		t.Fatalf("stderr = %q, err = %v", errb.String(), err)
	}
	n, _, _, _ = engine(t, nice.Options{}, 0, nil)
	if err := n.Run(helperArgv(t, "exit", "7")); status(err) != 7 {
		t.Fatalf("err = %v; want status 7", err)
	}
}

// TestPermissionIsAWarning: like GNU nice, a niceness that cannot be set for
// lack of privilege is reported and the command still runs.
func TestPermissionIsAWarning(t *testing.T) {
	n, out, errb, _ := engine(t, nice.Options{Adjust: true, Adjustment: -5}, 0, fs.ErrPermission)
	if err := n.Run(helperArgv(t, "echo", "ran")); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.String() != "ran\n" || errb.String() != "nicex: cannot set niceness: Permission denied\n" {
		t.Fatalf("out = %q, stderr = %q", out.String(), errb.String())
	}
}

func TestUnsupportedIsAWarning(t *testing.T) {
	n, out, errb, _ := engine(t, nice.Options{}, 0, nice.ErrUnsupported)
	n.GetNiceness = func() (int, error) { return 0, nice.ErrUnsupported }
	if err := n.Run(helperArgv(t, "echo", "ran")); err != nil || out.String() != "ran\n" {
		t.Fatalf("out = %q, err = %v", out.String(), err)
	}
	if !strings.HasPrefix(errb.String(), "nicex: cannot set niceness: ") {
		t.Fatalf("stderr = %q", errb.String())
	}
}

// TestOtherSetFailureStopsTheCommand: any other failure is nicex's own, 125,
// and the command it started does not carry on.
func TestOtherSetFailureStopsTheCommand(t *testing.T) {
	n, out, errb, _ := engine(t, nice.Options{}, 0, errors.New("invalid argument"))
	err := n.Run(helperArgv(t, "sleep", "30"))
	if status(err) != nice.ExitCanceled || errb.String() != "nicex: cannot set niceness: Invalid argument\n" || out.Len() != 0 {
		t.Fatalf("err = %v, stderr = %q, out = %q", err, errb.String(), out.String())
	}
}

func TestCommandNotFound(t *testing.T) {
	n, _, errb, _ := engine(t, nice.Options{}, 0, nil)
	err := n.Run([]string{"gnux-no-such-program-anywhere"})
	if status(err) != nice.ExitNotFound || errb.String() != "nicex: 'gnux-no-such-program-anywhere': No such file or directory\n" {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
}

func TestCommandNotExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows decides what is executable by extension")
	}
	f := filepath.Join(t.TempDir(), "script")
	if err := os.WriteFile(f, []byte("not a program"), 0o644); err != nil {
		t.Fatal(err)
	}
	n, _, errb, _ := engine(t, nice.Options{}, 0, nil)
	err := n.Run([]string{f})
	if status(err) != nice.ExitCannotInvoke || !strings.HasPrefix(errb.String(), "nicex: '"+f+"': ") {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
}

// TestRealNiceness runs with the system calls: lowering priority needs no
// privilege where niceness exists.
func TestRealNiceness(t *testing.T) {
	var out, errb bytes.Buffer
	s := core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb}
	if err := nice.New(nice.Options{}, s).Run(nil); err != nil {
		if strings.Contains(errb.String(), "not supported") {
			t.Skip("no niceness on this platform")
		}
		t.Fatalf("print: %v (%s)", err, errb.String())
	}
	out.Reset()
	if err := nice.New(nice.Options{Adjust: true, Adjustment: 1}, s).Run(helperArgv(t, "echo", "ok")); err != nil || out.String() != "ok\n" {
		t.Fatalf("run: out = %q, err = %v, stderr = %q", out.String(), err, errb.String())
	}
}
