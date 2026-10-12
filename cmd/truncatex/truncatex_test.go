package truncatex

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/truncate"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want truncate.Options
	}{
		{"defaults", nil, truncate.Options{}},
		{"no-create", []string{"-c"}, truncate.Options{NoCreate: true}},
		{"no-create long", []string{"--no-create"}, truncate.Options{NoCreate: true}},
		{"io-blocks", []string{"-o"}, truncate.Options{IOBlocks: true}},
		{"io-blocks long", []string{"--io-blocks"}, truncate.Options{IOBlocks: true}},
		{"reference", []string{"-r", "ref"}, truncate.Options{Reference: "ref"}},
		{"reference long", []string{"--reference=ref"}, truncate.Options{Reference: "ref"}},
		{"size", []string{"-s", "10K"}, truncate.Options{Size: "10K"}},
		{"size attached", []string{"-s+5"}, truncate.Options{Size: "+5"}},
		{"size long", []string{"--size=%4K"}, truncate.Options{Size: "%4K"}},
		{"combined", []string{"-co", "-s", "2"}, truncate.Options{NoCreate: true, IOBlocks: true, Size: "2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(truncate.Name, pflag.ContinueOnError)
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
	p := filepath.Join(t.TempDir(), "f")
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs([]string{"-s", "1K", p})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute: %v (%s)", err, errb.String())
	}
	info, err := os.Stat(p)
	if err != nil || info.Size() != 1024 {
		t.Fatalf("stat: %v, size %v", err, info)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != truncate.Name || !strings.HasPrefix(m.Use, truncate.Name+" ") {
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
