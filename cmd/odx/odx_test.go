package odx

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/od"
)

func options(t *testing.T, args ...string) (od.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(od.Name, pflag.ContinueOnError)
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
		want od.Options
	}{
		{"defaults", nil, od.Options{}},
		{"radix decimal", []string{"-A", "d"}, od.Options{Radix: od.RadixDecimal}},
		{"radix hex", []string{"--address-radix=x"}, od.Options{Radix: od.RadixHex}},
		{"radix none", []string{"-An"}, od.Options{Radix: od.RadixNone}},
		{"radix octal", []string{"-Ao"}, od.Options{}},
		{"endian big", []string{"--endian=big"}, od.Options{Endian: od.EndianBig}},
		{"endian little", []string{"--endian", "little"}, od.Options{Endian: od.EndianLittle}},
		{"skip", []string{"-j", "0x10"}, od.Options{Skip: 16}},
		{"limit", []string{"-N", "1k"}, od.Options{Limit: 1024, LimitSet: true}},
		{"limit zero", []string{"--read-bytes=0"}, od.Options{LimitSet: true}},
		{"strings", []string{"-S", "5"}, od.Options{Strings: 5}},
		{"strings attached", []string{"-S8"}, od.Options{Strings: 8}},
		{"verbose", []string{"-v"}, od.Options{Verbose: true}},
		{"width", []string{"-w8"}, od.Options{Width: 8}},
		{"type", []string{"-t", "x1"}, od.Options{Types: []string{"x1"}}},
		{"types keep order", []string{"-t", "c", "-x", "-t", "d4", "-b"}, od.Options{Types: []string{"c", "x2", "d4", "o1"}}},
		{"traditional letters combine", []string{"-cb"}, od.Options{Types: []string{"c", "o1"}}},
		{"every traditional letter", []string{"-a", "-d", "-f", "-i", "-l", "-o", "-s"}, od.Options{Types: []string{"a", "u2", "fF", "dI", "dL", "o2", "d2"}}},
		{"long traditional name", []string{"--hex-2-bytes"}, od.Options{Types: []string{"x2"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestBuildOptionsInvalid(t *testing.T) {
	tests := []struct {
		args  []string
		msg   string
		usage bool
	}{
		{[]string{"-A", "q"}, "invalid output address radix 'q'; it must be one character from [doxn]", false},
		{[]string{"-j", "x"}, "invalid -j argument 'x'", false},
		{[]string{"-N", "-1"}, "invalid -N argument '-1'", false},
		{[]string{"-w", "0"}, "invalid -w argument '0'", false},
		{[]string{"-S", "0"}, "invalid -S argument '0'", false},
		{[]string{"--endian=middle"}, "invalid argument 'middle' for '--endian'\nValid arguments are:\n  - 'big'\n  - 'little'", true},
		{[]string{"-t", "q"}, "invalid character 'q' in type string 'q'", true},
	}
	for _, tt := range tests {
		_, err := options(t, tt.args...)
		var u *core.UsageError
		var f fatal
		switch {
		case tt.usage && errors.As(err, &u) && u.Msg == tt.msg:
		case !tt.usage && errors.As(err, &f) && f.msg == tt.msg:
		default:
			t.Fatalf("%v: err = %v, want %q", tt.args, err, tt.msg)
		}
	}
}

func TestCommand(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"-c"}, "0000000   h   i  \\n\n0000003\n"},
		{[]string{"-An", "-tx1"}, " 68 69 0a\n"},
		{[]string{"-Ax", "-b"}, "000000 150 151 012\n000003\n"},
		{[]string{"-j1", "-N1", "-c"}, "0000001   i\n0000002\n"},
	}
	for _, tt := range tests {
		var out, errb bytes.Buffer
		c := Spec().Command(core.Streams{In: strings.NewReader("hi\n"), Out: &out, Err: &errb})
		c.SetArgs(tt.args)
		if err := c.Execute(); err != nil {
			t.Fatalf("%v: %v (%s)", tt.args, err, errb.String())
		}
		if out.String() != tt.want {
			t.Fatalf("%v: stdout = %q, want %q", tt.args, out.String(), tt.want)
		}
	}

	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs([]string{"-j", "nope"})
	if err := c.Execute(); !errors.Is(err, core.ErrFailed) || errb.String() != "odx: invalid -j argument 'nope'\n" {
		t.Fatalf("err=%v stderr=%q", err, errb.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != od.Name || !strings.HasPrefix(m.Use, od.Name+" ") {
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
