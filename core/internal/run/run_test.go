package run_test

import (
	"bytes"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/run"
)

func TestCommandPrefersTheTool(t *testing.T) {
	var out bytes.Buffer
	s := core.Streams{In: strings.NewReader(""), Out: &out, Err: &out}
	tool := func(name string, args []string, s core.Streams) (int, bool) {
		if name != "echo" {
			return 0, false
		}
		_, _ = s.Out.Write([]byte(strings.Join(args, " ") + "\n"))
		return 3, true
	}

	res, err := run.Command(tool, []string{"echo", "a", "b"}, "", s)
	if err != nil || res.Status != 3 || res.Signaled || out.String() != "a b\n" {
		t.Fatalf("res = %+v, err = %v, out = %q", res, err, out.String())
	}
}

func TestCommandNotFound(t *testing.T) {
	s := core.Discard()
	none := func(string, []string, core.Streams) (int, bool) { return 0, false }
	for _, argv := range [][]string{nil, {"gnux-no-such-program-anywhere"}} {
		if _, err := run.Command(none, argv, "", s); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Command(%q) error = %v, want fs.ErrNotExist", argv, err)
		}
	}
}
