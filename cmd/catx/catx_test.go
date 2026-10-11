package catx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/cat"
)

func TestBuildOptions(t *testing.T) {
	visible := cat.Options{ShowEnds: true, ShowTabs: true, ShowNonprinting: true}
	tests := []struct {
		name string
		args []string
		want cat.Options
	}{
		{"defaults", nil, cat.Options{}},
		{"number", []string{"-n"}, cat.Options{Number: true}},
		{"number long", []string{"--number"}, cat.Options{Number: true}},
		{"number nonblank", []string{"-b"}, cat.Options{NumberNonblank: true}},
		{"number nonblank long", []string{"--number-nonblank"}, cat.Options{NumberNonblank: true}},
		{"squeeze", []string{"-s"}, cat.Options{SqueezeBlank: true}},
		{"squeeze long", []string{"--squeeze-blank"}, cat.Options{SqueezeBlank: true}},
		{"show ends", []string{"-E"}, cat.Options{ShowEnds: true}},
		{"show ends long", []string{"--show-ends"}, cat.Options{ShowEnds: true}},
		{"show tabs", []string{"-T"}, cat.Options{ShowTabs: true}},
		{"show tabs long", []string{"--show-tabs"}, cat.Options{ShowTabs: true}},
		{"show nonprinting", []string{"-v"}, cat.Options{ShowNonprinting: true}},
		{"show nonprinting long", []string{"--show-nonprinting"}, cat.Options{ShowNonprinting: true}},
		{"show all", []string{"-A"}, visible},
		{"show all long", []string{"--show-all"}, visible},
		{"-e is -vE", []string{"-e"}, cat.Options{ShowEnds: true, ShowNonprinting: true}},
		{"-t is -vT", []string{"-t"}, cat.Options{ShowTabs: true, ShowNonprinting: true}},
		{"-et is -A", []string{"-et"}, visible},
		{"-vET is -A", []string{"-vET"}, visible},
		{"-u is ignored", []string{"-u"}, cat.Options{}},
		{"combined", []string{"-nsA"}, cat.Options{Number: true, SqueezeBlank: true, ShowEnds: true, ShowTabs: true, ShowNonprinting: true}},
		{"both numberings are passed on", []string{"-bn"}, cat.Options{Number: true, NumberNonblank: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(cat.Name, pflag.ContinueOnError)
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

// TestCommand runs the tool the way the binary does, through its cobra command.
func TestCommand(t *testing.T) {
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader("a\tb\n\n"), Out: &out, Err: &errb})
	c.SetArgs([]string{"-An", "-"})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute: %v (stderr %q)", err, errb.String())
	}
	if want := "     1\ta^Ib$\n     2\t$\n"; out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != cat.Name || !strings.HasPrefix(m.Use, cat.Name+" ") {
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
