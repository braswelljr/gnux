package kill_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/kill"
)

// sent records one injected signal delivery.
type sent struct{ pid, sig int }

// run executes the engine with an injected sender that records deliveries
// and fails for pid 99999.
func run(opts kill.Options, args ...string) (stdout, stderr string, calls []sent, err error) {
	var out, errb bytes.Buffer
	k := kill.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	k.SendFn = func(pid, sig int) error {
		calls = append(calls, sent{pid, sig})
		if pid == 99999 {
			return syscall.ESRCH
		}
		return nil
	}
	err = k.Run(args)
	return out.String(), errb.String(), calls, err
}

func TestParseSignal(t *testing.T) {
	tests := []struct {
		spec string
		want int
		ok   bool
	}{
		{"TERM", kill.TermSignal, true},
		{"SIGTERM", kill.TermSignal, true},
		{"term", kill.TermSignal, true},
		{"SigKill", kill.KillSignal, true},
		{"KILL", kill.KillSignal, true},
		{"9", 9, true},
		{"15", 15, true},
		{"0", 0, true},
		{"", 0, false},
		{"NOPE", 0, false},
		{"SIG", 0, false},
		{"999", 0, false},
		{"-1", 0, false},
		{"1x", 0, false},
	}
	for _, tt := range tests {
		got, ok := kill.ParseSignal(tt.spec)
		if got != tt.want || ok != tt.ok {
			t.Errorf("ParseSignal(%q) = %d, %v; want %d, %v", tt.spec, got, ok, tt.want, tt.ok)
		}
	}
}

func TestSignalsTable(t *testing.T) {
	sigs := kill.Signals()
	if len(sigs) < 20 {
		t.Fatalf("only %d signals", len(sigs))
	}
	for i, s := range sigs {
		if i > 0 && s.Number <= sigs[i-1].Number {
			t.Fatalf("table not ordered at %+v", s)
		}
		if s.Name == "" || strings.HasPrefix(s.Name, "SIG") || s.Description == "" {
			t.Fatalf("bad entry %+v", s)
		}
		if name, ok := kill.SignalName(s.Number); !ok || name != s.Name {
			t.Fatalf("SignalName(%d) = %q", s.Number, name)
		}
	}
	if name, _ := kill.SignalName(kill.KillSignal); name != "KILL" || kill.KillSignal != 9 || kill.TermSignal != 15 {
		t.Fatalf("KILL=%d TERM=%d", kill.KillSignal, kill.TermSignal)
	}
}

func TestSend(t *testing.T) {
	tests := []struct {
		name  string
		opts  kill.Options
		args  []string
		calls []sent
	}{
		{"default is TERM", kill.Options{}, []string{"10"}, []sent{{10, kill.TermSignal}}},
		{"by name", kill.Options{Signal: "KILL"}, []string{"10", "11"}, []sent{{10, 9}, {11, 9}}},
		{"by number", kill.Options{Signal: "9"}, []string{"10"}, []sent{{10, 9}}},
		{"SIG prefix and lower case", kill.Options{Signal: "sigint"}, []string{"10"}, []sent{{10, 2}}},
		{"signal 0", kill.Options{Signal: "0"}, []string{"10"}, []sent{{10, 0}}},
		{"process group", kill.Options{}, []string{"-10"}, []sent{{-10, kill.TermSignal}}},
		{"plus sign", kill.Options{}, []string{"+10"}, []sent{{10, kill.TermSignal}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, calls, err := run(tt.opts, tt.args...)
			if err != nil || out != "" || errb != "" {
				t.Fatalf("err=%v out=%q stderr=%q", err, out, errb)
			}
			if fmt.Sprint(calls) != fmt.Sprint(tt.calls) {
				t.Fatalf("calls = %v, want %v", calls, tt.calls)
			}
		})
	}
}

func TestSendErrorsContinue(t *testing.T) {
	_, errb, calls, err := run(kill.Options{}, "abc", "99999", "12")
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	want := "killx: 'abc': invalid process id\nkillx: 99999: No such process\n"
	if errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
	if len(calls) != 2 || calls[1].pid != 12 {
		t.Fatalf("calls = %v: a failing PID must not stop the rest", calls)
	}
}

func TestSendUsage(t *testing.T) {
	_, _, _, err := run(kill.Options{})
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "no process ID specified" {
		t.Fatalf("err = %v, want 'no process ID specified'", err)
	}

	_, errb, calls, err := run(kill.Options{Signal: "BOGUS"}, "1")
	if !errors.Is(err, core.ErrFailed) || errb != "killx: 'BOGUS': invalid signal\n" || len(calls) != 0 {
		t.Fatalf("err=%v stderr=%q calls=%v", err, errb, calls)
	}
}

func TestList(t *testing.T) {
	out, _, _, err := run(kill.Options{Mode: kill.ModeList})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != len(kill.Signals()) || lines[0] != "HUP" {
		t.Fatalf("list = %q", out)
	}

	tests := []struct {
		args []string
		want string
	}{
		{[]string{"9"}, "KILL\n"},
		{[]string{"KILL"}, "9\n"},
		{[]string{"sigterm"}, "15\n"},
		{[]string{"143"}, "TERM\n"}, // an exit status: 128 + 15
		{[]string{"2", "HUP"}, "INT\n1\n"},
	}
	for _, tt := range tests {
		out, errb, _, err := run(kill.Options{Mode: kill.ModeList}, tt.args...)
		if err != nil || out != tt.want || errb != "" {
			t.Errorf("-l %v: out=%q stderr=%q err=%v, want %q", tt.args, out, errb, err, tt.want)
		}
	}

	out, errb, _, err := run(kill.Options{Mode: kill.ModeList}, "NOPE", "9")
	if !errors.Is(err, core.ErrFailed) || errb != "killx: 'NOPE': invalid signal\n" || out != "KILL\n" {
		t.Fatalf("out=%q stderr=%q err=%v", out, errb, err)
	}
}

func TestTable(t *testing.T) {
	out, _, _, err := run(kill.Options{Mode: kill.ModeTable}, "TERM", "9")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("table = %q", out)
	}
	if f := strings.Fields(lines[0]); len(f) < 3 || f[0] != "15" || f[1] != "TERM" || f[2] != "Terminated" {
		t.Fatalf("TERM row = %q", lines[0])
	}
	if f := strings.Fields(lines[1]); len(f) < 3 || f[0] != "9" || f[1] != "KILL" || f[2] != "Killed" {
		t.Fatalf("KILL row = %q", lines[1])
	}
	// Columns line up: the name always starts at the same offset.
	if strings.Index(lines[0], "TERM") != strings.Index(lines[1], "KILL") {
		t.Fatalf("columns do not line up: %q", out)
	}

	all, _, _, _ := run(kill.Options{Mode: kill.ModeTable})
	if n := strings.Count(all, "\n"); n != len(kill.Signals()) {
		t.Fatalf("%d table rows, want %d", n, len(kill.Signals()))
	}
}

// TestRealSignal terminates a real child process with the default sender.
func TestRealSignal(t *testing.T) {
	cmd := exec.Command(os.Args[0], "sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	var out, errb bytes.Buffer
	k := kill.New(kill.Options{Signal: "KILL"}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	if err := k.Run([]string{fmt.Sprint(cmd.Process.Pid)}); err != nil {
		t.Fatalf("Run: %v (%s)", err, errb.String())
	}
	select {
	case err := <-done:
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("child ended with %v, want it killed", err)
		}
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("child still running after killx -s KILL")
	}
}

func TestExitStatus(t *testing.T) {
	cmd := exec.Command(os.Args[0], "exit", "3")
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || kill.ExitStatus(exitErr.ProcessState) != 3 {
		t.Fatalf("ExitStatus of 'exit 3' = %v", err)
	}
}
