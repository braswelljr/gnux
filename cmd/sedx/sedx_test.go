package sedx

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sed"
)

func options(t *testing.T, args ...string) (sed.Options, []string, error) {
	t.Helper()
	fs := pflag.NewFlagSet(sed.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse(rewriteArgs(args)); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	o, err := buildOptions(fs)
	return o, fs.Args(), err
}

func TestBuildOptions(t *testing.T) {
	expr := func(texts ...string) []sed.Script {
		var s []sed.Script
		for _, text := range texts {
			s = append(s, sed.Script{Text: text})
		}
		return s
	}
	tests := []struct {
		name     string
		args     []string
		want     sed.Options
		operands []string
	}{
		{"defaults", nil, sed.Options{}, nil},
		{"script operand", []string{"s/a/b/", "file"}, sed.Options{}, []string{"s/a/b/", "file"}},
		{"-n", []string{"-n"}, sed.Options{Quiet: true}, nil},
		{"--quiet", []string{"--quiet"}, sed.Options{Quiet: true}, nil},
		{"--silent", []string{"--silent"}, sed.Options{Quiet: true}, nil},
		{"-e", []string{"-e", "p"}, sed.Options{Scripts: expr("p")}, nil},
		{"-e attached", []string{"-ep"}, sed.Options{Scripts: expr("p")}, nil},
		{"-e attached with equals", []string{"-e=p"}, sed.Options{Scripts: expr("=p")}, nil},
		{"-e that looks like an option", []string{"-e", "-i.bak"}, sed.Options{Scripts: expr("-i.bak")}, nil},
		{"-e that holds r and i", []string{"-eri.x"}, sed.Options{Scripts: expr("ri.x")}, nil},
		{"--expression", []string{"--expression=p"}, sed.Options{Scripts: expr("p")}, nil},
		{"--expression detached", []string{"--expression", "-ri"}, sed.Options{Scripts: expr("-ri")}, nil},
		{"-e twice", []string{"-e", "p", "-e", "d"}, sed.Options{Scripts: expr("p", "d")}, nil},
		{"-f", []string{"-f", "s.sed"}, sed.Options{Scripts: []sed.Script{{Text: "s.sed", File: true}}}, nil},
		{"--file", []string{"--file=s.sed"}, sed.Options{Scripts: []sed.Script{{Text: "s.sed", File: true}}}, nil},
		{"-e then -f", []string{"-e", "p", "-f", "s.sed"}, sed.Options{Scripts: []sed.Script{{Text: "p"}, {Text: "s.sed", File: true}}}, nil},
		{"-f then -e", []string{"-fs.sed", "-e", "p"}, sed.Options{Scripts: []sed.Script{{Text: "s.sed", File: true}, {Text: "p"}}}, nil},
		{"-e leaves the operands", []string{"-e", "p", "a", "b"}, sed.Options{Scripts: expr("p")}, []string{"a", "b"}},
		{"-E", []string{"-E"}, sed.Options{Extended: true}, nil},
		{"-r", []string{"-r"}, sed.Options{Extended: true}, nil},
		{"--regexp-extended", []string{"--regexp-extended"}, sed.Options{Extended: true}, nil},
		{"-nr", []string{"-nr"}, sed.Options{Quiet: true, Extended: true}, nil},
		{"-i", []string{"-i", "s/a/b/", "file"}, sed.Options{InPlace: true}, []string{"s/a/b/", "file"}},
		{"--in-place", []string{"--in-place"}, sed.Options{InPlace: true}, nil},
		{"-i suffix", []string{"-i.bak"}, sed.Options{InPlace: true, Suffix: ".bak"}, nil},
		{"--in-place suffix", []string{"--in-place=.bak"}, sed.Options{InPlace: true, Suffix: ".bak"}, nil},
		{"--in-place suffix true", []string{"--in-place=true"}, sed.Options{InPlace: true, Suffix: "true"}, nil},
		{"--in-place empty suffix", []string{"--in-place="}, sed.Options{InPlace: true}, nil},
		{"-i suffix of letters", []string{"-in"}, sed.Options{InPlace: true, Suffix: "n"}, nil},
		{"-i suffix with a star", []string{"-ibak/*"}, sed.Options{InPlace: true, Suffix: "bak/*"}, nil},
		{"-ni", []string{"-ni"}, sed.Options{Quiet: true, InPlace: true}, nil},
		{"-ni suffix", []string{"-ni~"}, sed.Options{Quiet: true, InPlace: true, Suffix: "~"}, nil},
		{"-i then suffix: the last wins", []string{"-i", "-i.bak"}, sed.Options{InPlace: true, Suffix: ".bak"}, nil},
		{"suffix then -i: the last wins", []string{"-i.bak", "-i"}, sed.Options{InPlace: true}, nil},
		{"--follow-symlinks", []string{"--follow-symlinks"}, sed.Options{FollowSymlinks: true}, nil},
		{"-s", []string{"-s"}, sed.Options{Separate: true}, nil},
		{"--separate", []string{"--separate"}, sed.Options{Separate: true}, nil},
		{"-z", []string{"-z"}, sed.Options{NullData: true}, nil},
		{"--null-data", []string{"--null-data"}, sed.Options{NullData: true}, nil},
		{"-u", []string{"-u"}, sed.Options{Unbuffered: true}, nil},
		{"--unbuffered", []string{"--unbuffered"}, sed.Options{Unbuffered: true}, nil},
		{"-l", []string{"-l", "40"}, sed.Options{LineLength: 40}, nil},
		{"-l attached", []string{"-l40"}, sed.Options{LineLength: 40}, nil},
		{"--line-length", []string{"--line-length=40"}, sed.Options{LineLength: 40}, nil},
		{"--line-length detached", []string{"--line-length", "40"}, sed.Options{LineLength: 40}, nil},
		{"-l 0 never wraps", []string{"-l", "0"}, sed.Options{LineLength: -1}, nil},
		{"-l 1", []string{"-l", "1"}, sed.Options{LineLength: 1}, nil},
		{"--posix", []string{"--posix"}, sed.Options{Posix: true}, nil},
		{"--sandbox", []string{"--sandbox"}, sed.Options{Sandbox: true}, nil},
		{"-b", []string{"-b"}, sed.Options{}, nil},
		{"--binary", []string{"--binary"}, sed.Options{}, nil},
		{"cluster", []string{"-nEsuz"}, sed.Options{Quiet: true, Extended: true, Separate: true, Unbuffered: true, NullData: true}, nil},
		{"options after operands", []string{"p", "file", "-n"}, sed.Options{Quiet: true}, []string{"p", "file"}},
		{"after --", []string{"-n", "--", "-i.bak", "-r"}, sed.Options{Quiet: true}, []string{"-i.bak", "-r"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, operands, err := options(t, tt.args...)
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildOptions(%v) = %+v, %v; want %+v", tt.args, got, err, tt.want)
			}
			if len(operands)+len(tt.operands) > 0 && !reflect.DeepEqual(operands, tt.operands) {
				t.Fatalf("operands of %v = %q, want %q", tt.args, operands, tt.operands)
			}
		})
	}
}

func TestBuildOptionsInvalid(t *testing.T) {
	var usage *core.UsageError
	for _, value := range []string{"x", "-1", "1.5", ""} {
		if _, _, err := options(t, "--line-length="+value); !errors.As(err, &usage) || usage.Msg != "invalid line length '"+value+"'" {
			t.Errorf("--line-length=%q: %v", value, err)
		}
	}
}

func TestRewriteArgs(t *testing.T) {
	tests := []struct {
		args, want []string
	}{
		{[]string{"-n", "p"}, []string{"-n", "p"}},
		{[]string{"-i.bak", "p", "f"}, []string{"--in-place==.bak", "p", "f"}},
		{[]string{"-ni.bak"}, []string{"-n", "--in-place==.bak"}},
		{[]string{"-ie"}, []string{"--in-place==e"}},
		{[]string{"--in-place=true"}, []string{"--in-place==true"}},
		{[]string{"-rn"}, []string{"-En"}},
		{[]string{"-ne", "p"}, []string{"-ne", "p"}},
		{[]string{"-e", ""}, []string{"-e", ""}},
		{[]string{"-es/r/i/"}, []string{"--expression=s/r/i/"}},
		{[]string{"-nl5"}, []string{"-n", "--line-length=5"}},
		{[]string{"--file", "-r"}, []string{"--file", "-r"}},
		{[]string{"--", "-r"}, []string{"--", "-r"}},
		{[]string{"-", "-r"}, []string{"-", "-E"}},
		{[]string{"-e"}, []string{"-e"}},
	}
	for _, tt := range tests {
		if got := rewriteArgs(tt.args); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("rewriteArgs(%q) = %q, want %q", tt.args, got, tt.want)
		}
	}
}

// command runs sedx the way the binary does and returns what it wrote.
func command(t *testing.T, stdin string, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errOut})
	c.SetArgs(args)
	err := c.Execute()
	return out.String(), errOut.String(), err
}

func TestCommand(t *testing.T) {
	out, errOut, err := command(t, "a\nb\n", "-n", "-e", "1p", "-e", "s/b/c/p")
	if err != nil || out != "a\nc\n" || errOut != "" {
		t.Fatalf("got %q, stderr %q, error %v", out, errOut, err)
	}

	dir := t.TempDir()
	file := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(file, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := command(t, "", "-i.orig", "s/x/y/", file); err != nil {
		t.Fatal(err)
	}
	edited, _ := os.ReadFile(file)
	backup, _ := os.ReadFile(file + ".orig")
	if string(edited) != "y\n" || string(backup) != "x\n" {
		t.Fatalf("file %q, backup %q", edited, backup)
	}

	var usage *core.UsageError
	for _, args := range [][]string{{"--debug", "p"}, {"-k", "p"}, {"-h"}, {"-l", "x", "p"}, {}} {
		if _, _, err := command(t, "", args...); !errors.As(err, &usage) {
			t.Errorf("%v: error %v, want a usage error", args, err)
		}
	}

	if out, _, err := command(t, "", "-n", "--version"); err != nil || !strings.HasPrefix(out, "sedx (gnux) ") {
		t.Errorf("--version: %q, %v", out, err)
	}
	if out, _, err := command(t, "", "--help"); err != nil || !strings.Contains(out, "-i, --in-place ") {
		t.Errorf("--help: %v\n%s", err, out)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != "sedx" || !strings.HasPrefix(m.Use, "sedx ") || !strings.Contains(m.Long, "Differences from GNU sed:") {
		t.Fatalf("meta: %q, %q", m.Name, m.Use)
	}
	if len(m.Examples) < 8 {
		t.Fatalf("%d examples", len(m.Examples))
	}
	for _, e := range m.Examples {
		if e.Description == "" || e.Bash == "" || e.PowerShell == "" {
			t.Errorf("incomplete example: %+v", e)
		}
	}
}
