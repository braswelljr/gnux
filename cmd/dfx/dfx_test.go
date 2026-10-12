package dfx

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/df"
)

func options(t *testing.T, args ...string) df.Options {
	t.Helper()
	fs := pflag.NewFlagSet(df.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	o := buildOptions(fs)
	if len(o.Types) == 0 {
		o.Types = nil
	}
	if len(o.ExcludeTypes) == 0 {
		o.ExcludeTypes = nil
	}
	return o
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want df.Options
	}{
		{"defaults", nil, df.Options{}},
		{"-h", []string{"-h"}, df.Options{BlockSize: "human-readable"}},
		{"-H", []string{"-H"}, df.Options{BlockSize: "si"}},
		{"--si", []string{"--si"}, df.Options{BlockSize: "si"}},
		{"-k", []string{"-k"}, df.Options{BlockSize: "1K"}},
		{"-m", []string{"-m"}, df.Options{BlockSize: "1M"}},
		{"-BM", []string{"-BM"}, df.Options{BlockSize: "M"}},
		{"last unit wins", []string{"-h", "-k"}, df.Options{BlockSize: "1K"}},
		{"last unit wins the other way", []string{"-k", "-h"}, df.Options{BlockSize: "human-readable"}},
		{"flags", []string{"-ailPT", "--total", "-v", "--sync"}, df.Options{All: true, Inodes: true, Local: true, Portability: true, PrintType: true, Total: true}},
		{"-t -F -x", []string{"-t", "ext4", "-F", "xfs", "-x", "tmpfs"}, df.Options{Types: []string{"ext4", "xfs"}, ExcludeTypes: []string{"tmpfs"}}},
		{"--output", []string{"--output"}, df.Options{OutputSet: true}},
		{"--output=list", []string{"--output=source,pcent", "--output=target"}, df.Options{OutputSet: true, Output: []string{"source", "pcent", "target"}}},
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
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs([]string{"--output=pcent", t.TempDir()})
	err := c.Execute()
	if err != nil && strings.Contains(errb.String(), "not supported") {
		t.Skip("no file system statistics here")
	}
	if err != nil || !strings.HasPrefix(out.String(), "Use%\n") {
		t.Fatalf("Execute: %v, stdout = %q, stderr = %q", err, out.String(), errb.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != df.Name || !strings.HasPrefix(m.Use, df.Name+" ") {
		t.Fatalf("Name = %q, Use = %q", m.Name, m.Use)
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
