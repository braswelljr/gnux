package fmtx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fmt"
)

func options(t *testing.T, args ...string) (fmt.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(fmt.Name, pflag.ContinueOnError)
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
		want fmt.Options
	}{
		{"defaults", nil, fmt.Options{}},
		{"-c", []string{"-c"}, fmt.Options{Crown: true}},
		{"-t", []string{"-t"}, fmt.Options{Tagged: true}},
		{"-s", []string{"-s"}, fmt.Options{Split: true}},
		{"-u", []string{"-u"}, fmt.Options{Uniform: true}},
		{"-p", []string{"-p", "# "}, fmt.Options{Prefix: "# "}},
		{"-w", []string{"-w", "60"}, fmt.Options{Width: 60, WidthSet: true}},
		{"-w 0", []string{"-w", "0"}, fmt.Options{WidthSet: true}},
		{"obsolete", []string{"-60"}, fmt.Options{Width: 60, WidthSet: true}},
		{"-g", []string{"-g", "50"}, fmt.Options{Goal: 50, GoalSet: true}},
		{"-g above default without -w", []string{"-g", "75"}, fmt.Options{Goal: 75, GoalSet: true}},
		{"-w and -g", []string{"-w", "100", "-g", "90"}, fmt.Options{Width: 100, WidthSet: true, Goal: 90, GoalSet: true}},
		{"-w then obsolete", []string{"-w", "60", "-40"}, fmt.Options{Width: 40, WidthSet: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil || got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, %v; want %+v", tt.args, got, err, tt.want)
			}
		})
	}
}

func TestBuildOptionsInvalid(t *testing.T) {
	for _, tt := range []struct {
		args []string
		bad  string
	}{
		{[]string{"-w", "x"}, "x"},
		{[]string{"-w", "2501"}, "2501"},
		{[]string{"-g", "76"}, "76"},
		{[]string{"-w", "50", "-g", "60"}, "60"},
	} {
		_, err := options(t, tt.args...)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != "invalid width: '"+tt.bad+"'" {
			t.Errorf("%v: %v", tt.args, err)
		}
	}
}

func TestCommand(t *testing.T) {
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader("aaa bbb\nccc ddd\n"), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs([]string{"-10"})
	if err := c.Execute(); err != nil || out.String() != "aaa bbb\nccc ddd\n" {
		t.Fatalf("Execute: %v, stdout = %q", err, out.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != fmt.Name || !strings.HasPrefix(m.Use, fmt.Name+" ") {
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
