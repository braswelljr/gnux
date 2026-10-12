package tty_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tty"
)

func run(opts tty.Options, name string, ok bool, args ...string) (string, error) {
	var out bytes.Buffer
	t := tty.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}})
	t.Terminal = func(any) (string, bool) { return name, ok }
	err := t.Run(args)
	return out.String(), err
}

func status(err error) int {
	var exit *core.ExitError
	if errors.As(err, &exit) {
		return exit.Code
	}
	if err != nil {
		return -1
	}
	return 0
}

func TestTty(t *testing.T) {
	tests := []struct {
		name   string
		opts   tty.Options
		tname  string
		ok     bool
		out    string
		status int
	}{
		{"terminal", tty.Options{}, "/dev/pts/3", true, "/dev/pts/3\n", 0},
		{"not a terminal", tty.Options{}, "", false, "not a tty\n", tty.ExitNotTTY},
		{"silent terminal", tty.Options{Silent: true}, "/dev/pts/3", true, "", 0},
		{"silent not a terminal", tty.Options{Silent: true}, "", false, "", tty.ExitNotTTY},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := run(tt.opts, tt.tname, tt.ok)
			if out != tt.out || status(err) != tt.status {
				t.Fatalf("out = %q, err = %v; want %q, status %d", out, err, tt.out, tt.status)
			}
		})
	}
}

func TestRealStreams(t *testing.T) {
	// A string reader is never a terminal.
	var out bytes.Buffer
	err := tty.New(tty.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}}).Run(nil)
	if status(err) != tty.ExitNotTTY || out.String() != "not a tty\n" {
		t.Fatalf("out = %q, err = %v", out.String(), err)
	}
}

func TestExtraOperand(t *testing.T) {
	_, err := run(tty.Options{}, "", false, "x")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "extra operand 'x'" {
		t.Fatalf("err = %v", err)
	}
}
