package whox

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/who"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		args []string
		want who.Options
	}{
		{nil, who.Options{}},
		{[]string{"-abdHlmpqrstu", "--lookup"}, who.Options{All: true, Boot: true, Dead: true, Heading: true, Login: true,
			MyLine: true, Process: true, Count: true, RunLevel: true, Short: true, Time: true, Users: true, Lookup: true}},
		{[]string{"-T"}, who.Options{Mesg: true}},
		{[]string{"-w"}, who.Options{Mesg: true}},
		{[]string{"--message"}, who.Options{Mesg: true}},
		{[]string{"--writable"}, who.Options{Mesg: true}},
	}
	for _, tt := range tests {
		fs := pflag.NewFlagSet(who.Name, pflag.ContinueOnError)
		Register(fs)
		if err := fs.Parse(tt.args); err != nil {
			t.Fatal(err)
		}
		if got := buildOptions(fs); got != tt.want {
			t.Errorf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
		}
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != who.Name || !strings.HasPrefix(m.Use, who.Name+" ") {
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
