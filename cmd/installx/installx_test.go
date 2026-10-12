package installx

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/install"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want install.Options
	}{
		{"defaults", nil, install.Options{}},
		{"copy is ignored", []string{"-c"}, install.Options{}},
		{"compare", []string{"-C"}, install.Options{Compare: true}},
		{"directory", []string{"-d"}, install.Options{Directory: true}},
		{"create leading", []string{"-D"}, install.Options{CreateLeading: true}},
		{"create leading long", []string{"--create-leading"}, install.Options{CreateLeading: true}},
		{"group", []string{"-g", "staff"}, install.Options{Group: "staff"}},
		{"mode", []string{"-m", "644"}, install.Options{Mode: "644"}},
		{"mode long", []string{"--mode=u=rw"}, install.Options{Mode: "u=rw"}},
		{"owner", []string{"-o", "root"}, install.Options{Owner: "root"}},
		{"preserve", []string{"-p"}, install.Options{PreserveTimestamps: true}},
		{"strip", []string{"-s"}, install.Options{Strip: true}},
		{"target", []string{"-t", "bin"}, install.Options{TargetDirectory: "bin"}},
		{"target long", []string{"--target-directory=bin"}, install.Options{TargetDirectory: "bin"}},
		{"no target", []string{"-T"}, install.Options{NoTargetDirectory: true}},
		{"verbose", []string{"-v"}, install.Options{Verbose: true}},
		{"combined", []string{"-Dvm", "600"}, install.Options{CreateLeading: true, Verbose: true, Mode: "600"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(install.Name, pflag.ContinueOnError)
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
	src := filepath.Join(dir, "a")
	if err := os.WriteFile(src, []byte("A"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "x", "y", "a")
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs([]string{"-D", src, dst})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute: %v (%s)", err, errb.String())
	}
	if b, err := os.ReadFile(dst); err != nil || string(b) != "A" {
		t.Fatalf("read: %v %q", err, b)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != install.Name || !strings.HasPrefix(m.Use, install.Name+" ") {
		t.Fatalf("Name = %q, Use = %q", m.Name, m.Use)
	}
	if m.Short == "" || m.Long == "" || m.Permissions == "" || len(m.UseCases) == 0 {
		t.Fatal("Meta is missing prose")
	}
	if len(m.Examples) < 10 {
		t.Fatalf("%d examples, want at least 10", len(m.Examples))
	}
	for _, e := range m.Examples {
		if e.Description == "" || e.Bash == "" || e.PowerShell == "" {
			t.Fatalf("incomplete example %+v", e)
		}
	}
}
