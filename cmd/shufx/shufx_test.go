package shufx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/shuf"
)

func options(t *testing.T, args ...string) (shuf.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(shuf.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return buildOptions(fs)
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want shuf.Options
	}{
		{"defaults", nil, shuf.Options{}},
		{"echo", []string{"-e"}, shuf.Options{Echo: true}},
		{"echo long", []string{"--echo"}, shuf.Options{Echo: true}},
		{"input range", []string{"-i", "1-6"}, shuf.Options{InputRange: "1-6"}},
		{"input range long", []string{"--input-range=1-6"}, shuf.Options{InputRange: "1-6"}},
		{"head count", []string{"-n", "3"}, shuf.Options{HeadCount: 3, HeadCountSet: true}},
		{"head count zero", []string{"--head-count=0"}, shuf.Options{HeadCountSet: true}},
		{"smallest count wins", []string{"-n", "5", "-n", "2", "-n", "9"}, shuf.Options{HeadCount: 2, HeadCountSet: true}},
		{"output", []string{"-o", "out"}, shuf.Options{Output: "out"}},
		{"random source", []string{"--random-source=seed"}, shuf.Options{RandomSource: "seed"}},
		{"repeat", []string{"-r"}, shuf.Options{Repeat: true}},
		{"zero terminated", []string{"-z"}, shuf.Options{ZeroTerminated: true}},
		{"combined", []string{"-rz", "-n1"}, shuf.Options{Repeat: true, ZeroTerminated: true, HeadCount: 1, HeadCountSet: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestBadCount(t *testing.T) {
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs([]string{"-n", "x"})
	if err := c.Execute(); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if errb.String() != "shufx: invalid line count: 'x'\n" {
		t.Fatalf("stderr = %q", errb.String())
	}
}

func TestCommand(t *testing.T) {
	tests := []struct {
		args []string
		want int
	}{
		{[]string{"-i", "1-5"}, 5},
		{[]string{"-e", "a", "b", "c"}, 3},
		{[]string{"-n", "2"}, 2},
		{[]string{"-r", "-n", "7", "-e", "x"}, 7},
	}
	for _, tt := range tests {
		var out, errb bytes.Buffer
		c := Spec().Command(core.Streams{In: strings.NewReader("1\n2\n3\n"), Out: &out, Err: &errb})
		c.SetArgs(tt.args)
		if err := c.Execute(); err != nil {
			t.Fatalf("%v: %v (%s)", tt.args, err, errb.String())
		}
		if n := strings.Count(out.String(), "\n"); n != tt.want {
			t.Fatalf("%v: %d lines, want %d", tt.args, n, tt.want)
		}
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != shuf.Name || !strings.HasPrefix(m.Use, shuf.Name+" ") {
		t.Fatalf("Name = %q, Use = %q", m.Name, m.Use)
	}
	if m.Short == "" || m.Long == "" || m.Permissions == "" || len(m.UseCases) == 0 {
		t.Fatal("Meta is missing prose")
	}
	if n := len(m.Examples); n < 10 {
		t.Fatalf("%d examples, want at least 10", n)
	}
	for _, e := range m.Examples {
		if e.Description == "" || e.Bash == "" || e.PowerShell == "" {
			t.Fatalf("incomplete example %+v", e)
		}
	}
}
