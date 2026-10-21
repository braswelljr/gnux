package stdbuf_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/stdbuf"
)

func newEngine(opts stdbuf.Options) (*stdbuf.Stdbuf, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	b := stdbuf.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	b.Executable = func() (string, error) { return "/nonexistent/gnux", nil }
	b.LibDirs = nil
	return b, &out, &errb
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

func TestParseMode(t *testing.T) {
	for in, want := range map[string]int64{"L": -1, "0": 0, "4096": 4096, "4K": 4096, "4k": 4096, "1KB": 1000, "1MiB": 1 << 20, " 8": 8} {
		if got, err := stdbuf.ParseMode(in); err != nil || got != want {
			t.Errorf("ParseMode(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "x", "1X", "-1", "1Kx"} {
		if _, err := stdbuf.ParseMode(in); err == nil {
			t.Errorf("ParseMode(%q) accepted", in)
		}
	}
}

func TestMistakes(t *testing.T) {
	tests := []struct {
		opts   stdbuf.Options
		args   []string
		usage  string
		stderr string
	}{
		{stdbuf.Options{Input: "L"}, []string{"x"}, "line buffering stdin is meaningless", ""},
		{stdbuf.Options{Output: "L"}, nil, "missing operand", ""},
		{stdbuf.Options{}, []string{"x"}, "you must specify a buffering mode option", ""},
		{stdbuf.Options{Output: "x"}, []string{"x"}, "", "stdbufx: invalid mode 'x'\n"},
		{stdbuf.Options{Output: "99999999999999999999"}, []string{"x"}, "", "stdbufx: invalid mode '99999999999999999999': Value too large for defined data type\n"},
	}
	for _, tt := range tests {
		b, _, errb := newEngine(tt.opts)
		err := b.Run(tt.args)
		var usage *core.UsageError
		if tt.usage != "" {
			if !errors.As(err, &usage) || usage.Msg != tt.usage {
				t.Errorf("%+v: err = %v, want usage %q", tt.opts, err, tt.usage)
			}
			continue
		}
		if status(err) != stdbuf.ExitCanceled || errb.String() != tt.stderr {
			t.Errorf("%+v: err=%v stderr=%q", tt.opts, err, errb)
		}
	}
}

func TestToolRunsInProcess(t *testing.T) {
	b, out, _ := newEngine(stdbuf.Options{Output: "L"})
	b.Tool = func(name string, args []string, s core.Streams) (int, bool) {
		if name != "echox" {
			return 0, false
		}
		_, _ = s.Out.Write([]byte(strings.Join(args, " ") + "\n"))
		return 3, true
	}
	if err := b.Run([]string{"echox", "hi"}); status(err) != 3 || out.String() != "hi\n" {
		t.Fatalf("err=%v stdout=%q", err, out)
	}
}

func TestProgramNeedsTheLibrary(t *testing.T) {
	b, _, errb := newEngine(stdbuf.Options{Output: "L"})
	if err := b.Run([]string{"some-program"}); status(err) != stdbuf.ExitCanceled || errb.String() != "stdbufx: failed to find 'libstdbuf.so'\n" {
		t.Fatalf("err=%v stderr=%q", err, errb)
	}
}

func TestProgramNotFound(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, stdbuf.LibName), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	b, _, errb := newEngine(stdbuf.Options{Output: "L"})
	b.LibDirs = []string{dir}
	err := b.Run([]string{"no-such-program-gnux-test"})
	if status(err) != stdbuf.ExitNotFound && status(err) != stdbuf.ExitCanceled {
		t.Fatalf("err=%v stderr=%q", err, errb)
	}
}
