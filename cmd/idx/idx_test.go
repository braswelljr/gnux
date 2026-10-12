package idx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/id"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want id.Options
	}{
		{"defaults", nil, id.Options{}},
		{"-a ignored", []string{"-a"}, id.Options{}},
		{"-u", []string{"-u"}, id.Options{User: true}},
		{"-un", []string{"-un"}, id.Options{User: true, Name: true}},
		{"-g", []string{"-g"}, id.Options{Group: true}},
		{"-Gnz", []string{"-Gnz"}, id.Options{Groups: true, Name: true, Zero: true}},
		{"-ru", []string{"-ru"}, id.Options{User: true, Real: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(id.Name, pflag.ContinueOnError)
			Register(fs)
			if err := fs.Parse(tt.args); err != nil {
				t.Fatal(err)
			}
			if got := buildOptions(fs); got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestCommand(t *testing.T) {
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs(nil)
	if err := c.Execute(); err != nil || out.Len() == 0 {
		t.Fatalf("Execute: %v, stdout = %q, stderr = %q", err, out.String(), errb.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != id.Name || !strings.HasPrefix(m.Use, id.Name+" ") {
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

func TestContext(t *testing.T) {
	var errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &errb})
	c.SetArgs([]string{"-Z"})
	if err := c.Execute(); err == nil || errb.String() != "idx: --context (-Z) works only on an SELinux-enabled kernel\n" {
		t.Fatalf("-Z: %v %q", err, errb.String())
	}
}
