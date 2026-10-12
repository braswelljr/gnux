package timeout_test

import (
	"bytes"
	"errors"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/timeout"
)

// run executes the engine with the test binary as the command (see helper).
func run(t *testing.T, opts timeout.Options, stdin string, args ...string) (stdout, stderr string, status int) {
	t.Helper()
	var out, errb bytes.Buffer
	e := timeout.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb})
	e.IsTerminal = func(any) bool { return false }
	err := e.Run(args)
	return out.String(), errb.String(), statusOf(t, err)
}

func statusOf(t *testing.T, err error) int {
	t.Helper()
	var exit *core.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exit):
		return exit.Code
	}
	t.Fatalf("Run returned %v, want nil or *core.ExitError", err)
	return 0
}

// self is the helper command: the test binary.
var self = os.Args[0]

func TestParseDuration(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"1", time.Second},
		{"1s", time.Second},
		{"1.5", 1500 * time.Millisecond},
		{".5", 500 * time.Millisecond},
		{"2m", 2 * time.Minute},
		{"1h", time.Hour},
		{"1d", 24 * time.Hour},
		{"0.25m", 15 * time.Second},
		{"0x10", 16 * time.Second},
		{"+3", 3 * time.Second},
		{"0", 0},
		{"0s", 0},
		{"inf", 0},
		{"infinity", 0},
		{"1e-12", 1},
		{"1e300", time.Duration(math.MaxInt64)},
	}
	for _, tt := range tests {
		got, err := timeout.ParseDuration(tt.in)
		if err != nil || got != tt.want {
			t.Errorf("ParseDuration(%q) = %v, %v; want %v", tt.in, got, err, tt.want)
		}
	}
	for _, in := range []string{"", "s", "x", "1x", "-1", "-0", "nan", "1ss", "1 s"} {
		_, err := timeout.ParseDuration(in)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != "invalid time interval '"+in+"'" {
			t.Errorf("ParseDuration(%q) error = %v", in, err)
		}
	}
}

func TestFinishesInTime(t *testing.T) {
	out, errb, status := run(t, timeout.Options{}, "", "10", self, "echo", "hello")
	if status != 0 || out != "hello\n" || errb != "" {
		t.Fatalf("status=%d out=%q stderr=%q", status, out, errb)
	}

	_, _, status = run(t, timeout.Options{}, "", "10", self, "exit", "3")
	if status != 3 {
		t.Fatalf("status = %d, want the command's 3", status)
	}

	out, _, _ = run(t, timeout.Options{}, "piped in", "10", self, "cat")
	if out != "piped in" {
		t.Fatalf("stdin not passed on: %q", out)
	}

	// 0 disables the limit.
	_, _, status = run(t, timeout.Options{}, "", "0", self, "sleep", "0.2")
	if status != 0 {
		t.Fatalf("status = %d with no limit", status)
	}
}

func TestTimesOut(t *testing.T) {
	start := time.Now()
	_, errb, status := run(t, timeout.Options{}, "", "0.2", self, "sleep", "30")
	if status != timeout.ExitTimedOut {
		t.Fatalf("status = %d, want 124 (stderr %q)", status, errb)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Fatalf("took %v", d)
	}
}

func TestVerbose(t *testing.T) {
	_, errb, status := run(t, timeout.Options{Verbose: true}, "", "0.2", self, "sleep", "30")
	want := "timeoutx: sending signal TERM to command '" + self + "'\n"
	if status != timeout.ExitTimedOut || errb != want {
		t.Fatalf("status=%d stderr=%q, want %q", status, errb, want)
	}
}

func TestKillSignal(t *testing.T) {
	_, _, status := run(t, timeout.Options{Signal: "KILL"}, "", "0.2", self, "sleep", "30")
	if status != 137 {
		t.Fatalf("status = %d, want 137 after KILL", status)
	}
	// With --foreground only the command is signaled, so timeoutx survives
	// to report the timeout.
	_, _, status = run(t, timeout.Options{Signal: "9", Foreground: true}, "", "0.2", self, "sleep", "30")
	if status != timeout.ExitTimedOut {
		t.Fatalf("status = %d, want 124 with --foreground", status)
	}
}

func TestFailures(t *testing.T) {
	tests := []struct {
		name   string
		opts   timeout.Options
		args   []string
		status int
		stderr string
	}{
		{"not found", timeout.Options{}, []string{"1", "gnux-no-such-command"}, 127,
			"timeoutx: failed to run command 'gnux-no-such-command': No such file or directory\n"},
		{"invalid signal", timeout.Options{Signal: "BOGUS"}, []string{"1", self}, 125,
			"timeoutx: 'BOGUS': invalid signal\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errb, status := run(t, tt.opts, "", tt.args...)
			if status != tt.status || errb != tt.stderr {
				t.Fatalf("status=%d stderr=%q, want %d %q", status, errb, tt.status, tt.stderr)
			}
		})
	}
}

func TestUsage(t *testing.T) {
	for _, tt := range []struct {
		args []string
		msg  string
	}{
		{nil, "missing operand"},
		{[]string{"5"}, "missing operand after '5'"},
		{[]string{"soon", "cmd"}, "invalid time interval 'soon'"},
	} {
		e := timeout.New(timeout.Options{}, core.Discard())
		err := e.Run(tt.args)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != tt.msg {
			t.Errorf("Run(%q) = %v, want usage error %q", tt.args, err, tt.msg)
		}
	}
}
