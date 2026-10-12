package readlinkx

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/readlink"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want readlink.Options
	}{
		{"defaults", nil, readlink.Options{}},
		{"canonicalize", []string{"-f"}, readlink.Options{Mode: readlink.ModeFollow}},
		{"canonicalize long", []string{"--canonicalize"}, readlink.Options{Mode: readlink.ModeFollow}},
		{"existing", []string{"-e"}, readlink.Options{Mode: readlink.ModeExisting}},
		{"existing long", []string{"--canonicalize-existing"}, readlink.Options{Mode: readlink.ModeExisting}},
		{"missing", []string{"-m"}, readlink.Options{Mode: readlink.ModeMissing}},
		{"missing long", []string{"--canonicalize-missing"}, readlink.Options{Mode: readlink.ModeMissing}},
		{"existing beats follow", []string{"-f", "-e"}, readlink.Options{Mode: readlink.ModeExisting}},
		{"existing beats follow, other order", []string{"-e", "-f"}, readlink.Options{Mode: readlink.ModeExisting}},
		{"follow beats missing", []string{"-m", "-f"}, readlink.Options{Mode: readlink.ModeFollow}},
		{"follow beats missing, other order", []string{"-f", "-m"}, readlink.Options{Mode: readlink.ModeFollow}},
		{"no newline", []string{"-n"}, readlink.Options{NoNewline: true}},
		{"no newline long", []string{"--no-newline"}, readlink.Options{NoNewline: true}},
		{"quiet is the default", []string{"-q"}, readlink.Options{}},
		{"silent is the default", []string{"--silent"}, readlink.Options{}},
		{"verbose", []string{"-v"}, readlink.Options{Verbose: true}},
		{"verbose beats quiet", []string{"-v", "-q"}, readlink.Options{Verbose: true}},
		{"verbose beats silent, other order", []string{"-s", "--verbose"}, readlink.Options{Verbose: true}},
		{"zero", []string{"-z"}, readlink.Options{Zero: true}},
		{"zero long", []string{"--zero"}, readlink.Options{Zero: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(readlink.Name, pflag.ContinueOnError)
			Register(fs)
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("parse %v: %v", tt.args, err)
			}
			if got := buildOptions(fs); got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestCommand(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "link")
	if err := os.Symlink("target", link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs([]string{"-n", link})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute: %v (stderr %q)", err, errb.String())
	}
	if out.String() != "target" {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != readlink.Name || !strings.HasPrefix(m.Use, readlink.Name+" ") {
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
