package unexpandx

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/unexpand"
)

func options(t *testing.T, args ...string) unexpand.Options {
	t.Helper()
	fs := pflag.NewFlagSet(unexpand.Name, pflag.ContinueOnError)
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
		want unexpand.Options
	}{
		{"defaults", nil, unexpand.Options{}},
		{"-a", []string{"-a"}, unexpand.Options{All: true}},
		{"first only", []string{"--first-only"}, unexpand.Options{FirstOnly: true}},
		{"-t implies -a", []string{"-t", "4"}, unexpand.Options{All: true, Tabs: []string{"4"}}},
		{"-t and first only", []string{"-t", "4", "--first-only"}, unexpand.Options{All: true, FirstOnly: true, Tabs: []string{"4"}}},
		{"obsolete does not imply -a", []string{"-4"}, unexpand.Options{Tabs: []string{"4"}}},
		{"obsolete and -t", []string{"-4", "-t", "8"}, unexpand.Options{All: true, Tabs: []string{"4", "8"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := options(t, tt.args...); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestCommand(t *testing.T) {
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader("    a   b\n"), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs([]string{"-t", "4"})
	if err := c.Execute(); err != nil || out.String() != "\ta\tb\n" {
		t.Fatalf("Execute: %v, stdout = %q", err, out.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != unexpand.Name || !strings.HasPrefix(m.Use, unexpand.Name+" ") {
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
