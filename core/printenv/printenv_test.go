package printenv_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/printenv"
)

var env = []string{"HOME=/home/u", "EMPTY=", "EQ=a=b", "PATH=/bin:/usr/bin"}

func run(opts printenv.Options, args ...string) (string, string, error) {
	var out, errb bytes.Buffer
	p := printenv.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	p.Environ = func() []string { return env }
	err := p.Run(args)
	return out.String(), errb.String(), err
}

func TestAll(t *testing.T) {
	out, _, err := run(printenv.Options{})
	if err != nil || out != strings.Join(env, "\n")+"\n" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	out, _, _ = run(printenv.Options{Null: true})
	if out != strings.Join(env, "\x00")+"\x00" {
		t.Fatalf("-0: out = %q", out)
	}
}

func TestNamed(t *testing.T) {
	tests := []struct {
		args []string
		want string
		fail bool
	}{
		{[]string{"HOME"}, "/home/u\n", false},
		{[]string{"PATH", "HOME"}, "/bin:/usr/bin\n/home/u\n", false},
		{[]string{"EMPTY"}, "\n", false},
		{[]string{"EQ"}, "a=b\n", false},
		{[]string{"NOPE"}, "", true},
		{[]string{"HOME", "NOPE", "PATH"}, "/home/u\n/bin:/usr/bin\n", true},
		{[]string{"HOME=/home/u"}, "", true},
		{[]string{"HOM"}, "", true},
		{[]string{"home"}, "", true},
	}
	for _, tt := range tests {
		out, errb, err := run(printenv.Options{}, tt.args...)
		if out != tt.want || errb != "" {
			t.Fatalf("%v: out = %q, stderr = %q, want %q", tt.args, out, errb, tt.want)
		}
		if failed := errors.Is(err, core.ErrFailed); failed != tt.fail {
			t.Fatalf("%v: err = %v, want failure %v", tt.args, err, tt.fail)
		}
	}
}

func TestNamedNull(t *testing.T) {
	out, _, _ := run(printenv.Options{Null: true}, "HOME", "EMPTY")
	if out != "/home/u\x00\x00" {
		t.Fatalf("out = %q", out)
	}
}
