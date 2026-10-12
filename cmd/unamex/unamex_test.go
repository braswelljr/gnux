package unamex

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/uname"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want uname.Options
	}{
		{"defaults", nil, uname.Options{}},
		{"-a", []string{"-a"}, uname.Options{All: true}},
		{"-snrvmpio", []string{"-snrvmpio"}, uname.Options{KernelName: true, Nodename: true, KernelRelease: true, KernelVersion: true, Machine: true, Processor: true, HardwarePlatform: true, OperatingSystem: true}},
		{"long names", []string{"--machine", "--operating-system"}, uname.Options{Machine: true, OperatingSystem: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(uname.Name, pflag.ContinueOnError)
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
	if m.Name != uname.Name || !strings.HasPrefix(m.Use, uname.Name+" ") {
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
