package uniqx

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/uniq"
)

// options registers every flag, parses args and returns the resolved Options.
func options(t *testing.T, args ...string) (uniq.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(uniq.Name, pflag.ContinueOnError)
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
		want uniq.Options
	}{
		{"defaults", nil, uniq.Options{}},
		{"count", []string{"-c"}, uniq.Options{Count: true}},
		{"count long", []string{"--count"}, uniq.Options{Count: true}},
		{"repeated", []string{"-d"}, uniq.Options{Repeated: true}},
		{"repeated long", []string{"--repeated"}, uniq.Options{Repeated: true}},
		{"unique", []string{"-u"}, uniq.Options{Unique: true}},
		{"unique long", []string{"--unique"}, uniq.Options{Unique: true}},
		{"ignore case", []string{"-i"}, uniq.Options{IgnoreCase: true}},
		{"ignore case long", []string{"--ignore-case"}, uniq.Options{IgnoreCase: true}},
		{"zero terminated", []string{"-z"}, uniq.Options{ZeroTerminated: true}},
		{"zero terminated long", []string{"--zero-terminated"}, uniq.Options{ZeroTerminated: true}},
		{"all repeated short", []string{"-D"}, uniq.Options{AllRepeated: true}},
		{"all repeated short combined", []string{"-Di"}, uniq.Options{AllRepeated: true, IgnoreCase: true}},
		{"all repeated long", []string{"--all-repeated"}, uniq.Options{AllRepeated: true}},
		{"all repeated none", []string{"--all-repeated=none"}, uniq.Options{AllRepeated: true}},
		{"all repeated prepend", []string{"--all-repeated=prepend"}, uniq.Options{AllRepeated: true, Delimit: uniq.DelimitPrepend}},
		{"all repeated separate", []string{"--all-repeated=separate"}, uniq.Options{AllRepeated: true, Delimit: uniq.DelimitSeparate}},
		{"all repeated abbreviated", []string{"--all-repeated=sep"}, uniq.Options{AllRepeated: true, Delimit: uniq.DelimitSeparate}},
		{"group", []string{"--group"}, uniq.Options{Group: uniq.GroupSeparate}},
		{"group separate", []string{"--group=separate"}, uniq.Options{Group: uniq.GroupSeparate}},
		{"group prepend", []string{"--group=prepend"}, uniq.Options{Group: uniq.GroupPrepend}},
		{"group append", []string{"--group=append"}, uniq.Options{Group: uniq.GroupAppend}},
		{"group both", []string{"--group=both"}, uniq.Options{Group: uniq.GroupBoth}},
		{"group abbreviated", []string{"--group=b"}, uniq.Options{Group: uniq.GroupBoth}},
		{"group abbreviated to one letter", []string{"--group=p"}, uniq.Options{Group: uniq.GroupPrepend}},
		{"skip fields", []string{"-f", "2"}, uniq.Options{SkipFields: 2}},
		{"skip fields attached", []string{"-f2"}, uniq.Options{SkipFields: 2}},
		{"skip fields long", []string{"--skip-fields=3"}, uniq.Options{SkipFields: 3}},
		{"skip chars", []string{"-s", "4"}, uniq.Options{SkipChars: 4}},
		{"skip chars long", []string{"--skip-chars=5"}, uniq.Options{SkipChars: 5}},
		{"check chars", []string{"-w", "6"}, uniq.Options{CheckChars: 6, CheckCharsSet: true}},
		{"check chars long", []string{"--check-chars=7"}, uniq.Options{CheckChars: 7, CheckCharsSet: true}},
		{"check zero chars", []string{"-w0"}, uniq.Options{CheckChars: 0, CheckCharsSet: true}},
		{"huge count is clamped", []string{"-s", "99999999999999999999999"}, uniq.Options{SkipChars: math.MaxInt64}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil {
				t.Fatalf("buildOptions(%v): %v", tt.args, err)
			}
			if got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestBuildOptionsInvalid(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"--all-repeated=sometimes"}, "invalid argument 'sometimes' for '--all-repeated'"},
		{[]string{"--all-repeated="}, "invalid argument '' for '--all-repeated'"},
		{[]string{"--group=never"}, "invalid argument 'never' for '--group'"},
		{[]string{"--group=x"}, "invalid argument 'x' for '--group'"},
		{[]string{"-f", "x"}, "invalid number of fields to skip: 'x'"},
		{[]string{"-f", "-1"}, "invalid number of fields to skip: '-1'"},
		{[]string{"-s", "1.5"}, "invalid number of bytes to skip: '1.5'"},
		{[]string{"-w", ""}, "invalid number of bytes to compare: ''"},
	}
	for _, tt := range tests {
		_, err := options(t, tt.args...)
		var usage *core.UsageError
		if !errors.As(err, &usage) {
			t.Fatalf("%v: err = %v, want a UsageError", tt.args, err)
		}
		if usage.Msg != tt.want {
			t.Fatalf("%v: message = %q, want %q", tt.args, usage.Msg, tt.want)
		}
	}
}

// TestAmbiguous checks that a prefix matching two methods is refused.
func TestAmbiguous(t *testing.T) {
	_, err := match("x", "a", []choice[int]{{"alpha", 1}, {"apple", 2}})
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "ambiguous argument 'a' for '--x'" {
		t.Fatalf("err = %v", err)
	}
}

// TestCommand runs the tool the way the binary does, through its cobra command.
func TestCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"default", nil, "a\nb\nc\n"},
		{"count", []string{"-c"}, "      2 a\n      1 b\n      2 c\n"},
		{"repeated", []string{"-d"}, "a\nc\n"},
		{"all repeated separate", []string{"--all-repeated=separate"}, "a\na\n\nc\nc\n"},
		{"group append", []string{"--group=append"}, "a\na\n\nb\n\nc\nc\n\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			c := Spec().Command(core.Streams{In: strings.NewReader("a\na\nb\nc\nc\n"), Out: &out, Err: &errb})
			c.SetArgs(tt.args)
			if err := c.Execute(); err != nil {
				t.Fatalf("Execute: %v (stderr %q)", err, errb.String())
			}
			if out.String() != tt.want {
				t.Fatalf("stdout = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != uniq.Name || !strings.HasPrefix(m.Use, uniq.Name+" ") {
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
