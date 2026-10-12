package expandx

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/expand"
)

func options(t *testing.T, args ...string) expand.Options {
	t.Helper()
	fs := pflag.NewFlagSet(expand.Name, pflag.ContinueOnError)
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
		want expand.Options
	}{
		{"defaults", nil, expand.Options{}},
		{"-i", []string{"-i"}, expand.Options{Initial: true}},
		{"-t", []string{"-t", "4"}, expand.Options{Tabs: []string{"4"}}},
		{"-t list", []string{"--tabs=2,5"}, expand.Options{Tabs: []string{"2,5"}}},
		{"-t twice", []string{"-t", "2", "-t", "5"}, expand.Options{Tabs: []string{"2", "5"}}},
		{"obsolete", []string{"-4"}, expand.Options{Tabs: []string{"4"}}},
		{"obsolete multi-digit", []string{"-12"}, expand.Options{Tabs: []string{"12"}}},
		{"consecutive obsolete forms run together", []string{"-4", "-8"}, expand.Options{Tabs: []string{"48"}}},
		{"obsolete and -t in order", []string{"-4", "-t", "8", "-16"}, expand.Options{Tabs: []string{"4", "8", "16"}}},
		{"obsolete with -i", []string{"-i4"}, expand.Options{Initial: true, Tabs: []string{"4"}}},
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
	c := Spec().Command(core.Streams{In: strings.NewReader("a\tb\n"), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs([]string{"-4"})
	if err := c.Execute(); err != nil || out.String() != "a   b\n" {
		t.Fatalf("Execute: %v, stdout = %q", err, out.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != expand.Name || !strings.HasPrefix(m.Use, expand.Name+" ") {
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
