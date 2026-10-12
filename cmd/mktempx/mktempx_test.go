package mktempx

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/mktemp"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want mktemp.Options
	}{
		{"defaults", nil, mktemp.Options{}},
		{"directory", []string{"-d"}, mktemp.Options{Directory: true}},
		{"directory long", []string{"--directory"}, mktemp.Options{Directory: true}},
		{"dry-run", []string{"-u"}, mktemp.Options{DryRun: true}},
		{"quiet", []string{"-q"}, mktemp.Options{Quiet: true}},
		{"suffix", []string{"--suffix=.txt"}, mktemp.Options{Suffix: ".txt", SuffixSet: true}},
		{"empty suffix", []string{"--suffix="}, mktemp.Options{SuffixSet: true}},
		{"tmpdir bare", []string{"--tmpdir"}, mktemp.Options{UseTmpDir: true}},
		{"tmpdir value", []string{"--tmpdir=out"}, mktemp.Options{UseTmpDir: true, TmpDir: "out"}},
		{"p", []string{"-p", "out"}, mktemp.Options{UseTmpDir: true, TmpDir: "out"}},
		{"p attached", []string{"-pout"}, mktemp.Options{UseTmpDir: true, TmpDir: "out"}},
		{"p wins over bare tmpdir", []string{"--tmpdir", "-p", "out"}, mktemp.Options{UseTmpDir: true, TmpDir: "out"}},
		{"t", []string{"-t"}, mktemp.Options{Legacy: true}},
		{"t with p is only a fallback", []string{"-t", "-p", "out"}, mktemp.Options{Legacy: true, TmpDir: "out"}},
		{"combined", []string{"-dqu"}, mktemp.Options{Directory: true, Quiet: true, DryRun: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(mktemp.Name, pflag.ContinueOnError)
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
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs([]string{"-d", "-p", dir, "w.XXXX"})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute: %v (%s)", err, errb.String())
	}
	name := strings.TrimSpace(out.String())
	if !strings.HasPrefix(name, filepath.Join(dir, "w.")) {
		t.Fatalf("name = %q", name)
	}
	if info, err := os.Stat(name); err != nil || !info.IsDir() {
		t.Fatalf("not a directory: %v", err)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != mktemp.Name || !strings.HasPrefix(m.Use, mktemp.Name+" ") {
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
