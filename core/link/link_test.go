package link_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/link"
)

func run(args ...string) (string, string, error) {
	var out, errb bytes.Buffer
	err := link.New(link.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestLink(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	if err := os.WriteFile(a, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errb, err := run(a, b)
	if err != nil || out != "" || errb != "" {
		t.Fatalf("err = %v, stdout %q, stderr %q", err, out, errb)
	}
	ia, _ := os.Stat(a)
	ib, err := os.Stat(b)
	if err != nil || !os.SameFile(ia, ib) {
		t.Fatalf("b is not a hard link to a (err %v)", err)
	}
}

func TestLinkFailure(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "missing"), filepath.Join(dir, "b")
	_, errb, err := run(a, b)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "linkx: cannot create link '" + b + "' to '" + a + "': No such file or directory\n"; errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
}

func TestLinkExisting(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, errb, err := run(a, b)
	if !errors.Is(err, core.ErrFailed) || !strings.HasSuffix(errb, ": File exists\n") {
		t.Fatalf("err = %v, stderr %q", err, errb)
	}
}

func TestInjectedLink(t *testing.T) {
	var got [2]string
	l := link.New(link.Options{}, core.Discard())
	l.LinkFn = func(o, n string) error { got = [2]string{o, n}; return nil }
	if err := l.Run([]string{"x", "y"}); err != nil || got != [2]string{"x", "y"} {
		t.Fatalf("err = %v, LinkFn got %v", err, got)
	}
}

func TestOperands(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{nil, "missing operand"},
		{[]string{"a"}, "missing operand after 'a'"},
		{[]string{"a", "b", "c"}, "extra operand 'c'"},
	}
	for _, tt := range tests {
		_, _, err := run(tt.args...)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != tt.want {
			t.Fatalf("%v: err = %v, want usage error %q", tt.args, err, tt.want)
		}
	}
}
