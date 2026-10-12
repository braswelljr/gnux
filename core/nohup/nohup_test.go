package nohup_test

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/nohup"
)

// self is the helper command: the test binary.
var self = os.Args[0]

// run executes the engine; terminal reports which streams count as terminals.
func run(t *testing.T, terminal func(any) bool, stdin string, args ...string) (stdout, stderr string, status int) {
	t.Helper()
	var out, errb bytes.Buffer
	in := strings.NewReader(stdin)
	n := nohup.New(nohup.Options{}, core.Streams{In: in, Out: &out, Err: &errb})
	n.IsTerminal = terminal
	n.Getenv = func(string) string { return "" }
	err := n.Run(args)
	var exit *core.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exit):
		status = exit.Code
	default:
		t.Fatalf("Run returned %v", err)
	}
	return out.String(), errb.String(), status
}

func none(any) bool { return false }

func TestNoTerminal(t *testing.T) {
	out, errb, status := run(t, none, "piped", self, "cat")
	if status != 0 || out != "piped" || errb != "" {
		t.Fatalf("status=%d out=%q stderr=%q", status, out, errb)
	}
	_, _, status = run(t, none, "", self, "exit", "3")
	if status != 3 {
		t.Fatalf("status = %d, want the command's 3", status)
	}
}

func TestStdoutTerminal(t *testing.T) {
	t.Chdir(t.TempDir())

	// Standard output alone is a terminal: it goes to nohup.out.
	var out, errb bytes.Buffer
	n := nohup.New(nohup.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	n.IsTerminal = func(s any) bool { return s == any(&out) }
	n.Getenv = func(string) string { return "" }
	if err := n.Run([]string{self, "echo", "saved"}); err != nil {
		t.Fatal(err)
	}
	if errb.String() != "nohupx: appending output to 'nohup.out'\n" || out.Len() != 0 {
		t.Fatalf("stderr = %q, stdout = %q", errb.String(), out.String())
	}
	data, err := os.ReadFile(nohup.OutFile)
	if err != nil || string(data) != "saved\n" {
		t.Fatalf("nohup.out = %q, %v", data, err)
	}
}

func TestStdinTerminal(t *testing.T) {
	var errb bytes.Buffer
	in := strings.NewReader("never read")
	n := nohup.New(nohup.Options{}, core.Streams{In: in, Out: &bytes.Buffer{}, Err: &errb})
	n.IsTerminal = func(s any) bool { return s == any(in) }
	n.Getenv = func(string) string { return "" }
	if err := n.Run([]string{self, "exit", "0"}); err != nil {
		t.Fatal(err)
	}
	if errb.String() != "nohupx: ignoring input\n" {
		t.Fatalf("stderr = %q", errb.String())
	}
}

func TestFailures(t *testing.T) {
	_, errb, status := run(t, none, "", "gnux-no-such-command")
	if status != nohup.ExitNotFound || errb != "nohupx: failed to run command 'gnux-no-such-command': No such file or directory\n" {
		t.Fatalf("status=%d stderr=%q", status, errb)
	}
	err := nohup.New(nohup.Options{}, core.Discard()).Run(nil)
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "missing operand" {
		t.Fatalf("no operand: %v", err)
	}
}
