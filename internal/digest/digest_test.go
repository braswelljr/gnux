package digest

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/b2sum"
	"github.com/braswelljr/gnux/core/hashsum"
	"github.com/braswelljr/gnux/core/md5sum"
)

var (
	md5Tool = Tool{Name: md5sum.Name, GNU: "md5sum", Algorithm: "MD5", Bits: 128, Weak: true, New: md5sum.New}
	b2Tool  = Tool{Name: b2sum.Name, GNU: "b2sum", Algorithm: "BLAKE2b", Bits: 512, Length: true, New: b2sum.New}
)

func options(t *testing.T, tl Tool, args ...string) (hashsum.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(tl.Name, pflag.ContinueOnError)
	tl.Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return tl.BuildOptions(fs)
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want hashsum.Options
	}{
		{"defaults", nil, hashsum.Options{}},
		{"-b", []string{"-b"}, hashsum.Options{Binary: true}},
		{"-t", []string{"-t"}, hashsum.Options{Text: true}},
		{"-b -t", []string{"-b", "-t"}, hashsum.Options{Text: true}},
		{"-t -b", []string{"-t", "-b"}, hashsum.Options{Text: true}},
		{"--tag", []string{"--tag"}, hashsum.Options{Tag: true}},
		{"-z", []string{"-z"}, hashsum.Options{Zero: true}},
		{"check flags", []string{"-c", "--ignore-missing", "--quiet", "--status", "--strict", "-w"},
			hashsum.Options{Check: true, IgnoreMissing: true, Quiet: true, Status: true, Strict: true, Warn: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, md5Tool, tt.args...)
			if err != nil || got != tt.want {
				t.Fatalf("BuildOptions(%v) = %+v, %v; want %+v", tt.args, got, err, tt.want)
			}
		})
	}
}

func TestLength(t *testing.T) {
	fs := pflag.NewFlagSet("x", pflag.ContinueOnError)
	md5Tool.Register(fs)
	if fs.Lookup("length") != nil {
		t.Fatal("md5sumx has -l")
	}
	got, err := options(t, b2Tool, "-l", "256")
	if err != nil || got != (hashsum.Options{Length: 256}) {
		t.Fatalf("-l 256: %+v, %v", got, err)
	}
	_, err = options(t, b2Tool, "-l", "x")
	var fatal *fatalError
	if !errors.As(err, &fatal) || fatal.Error() != "invalid length: 'x'" {
		t.Fatalf("-l x: %v", err)
	}
}

func execute(tl Tool, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	c := tl.Spec().Command(core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb})
	c.SetArgs(args)
	err = c.Execute()
	return out.String(), errb.String(), err
}

func TestCommand(t *testing.T) {
	out, _, err := execute(md5Tool, "abc")
	if err != nil || out != "900150983cd24fb0d6963f7d28e17f72  -\n" {
		t.Fatalf("md5: %q, %v", out, err)
	}
	out, _, err = execute(md5Tool, "abc", "--tag")
	if err != nil || out != "MD5 (-) = 900150983cd24fb0d6963f7d28e17f72\n" {
		t.Fatalf("--tag: %q, %v", out, err)
	}
	out, _, err = execute(b2Tool, "abc", "-l", "8", "--tag")
	if err != nil || !strings.HasPrefix(out, "BLAKE2b-8 (-) = ") {
		t.Fatalf("-l 8 --tag: %q, %v", out, err)
	}
	_, errb, err := execute(b2Tool, "abc", "-l", "x")
	if !errors.Is(err, core.ErrFailed) || errb != "b2sumx: invalid length: 'x'\n" {
		t.Fatalf("-l x: %v %q", err, errb)
	}
	_, _, err = execute(md5Tool, "", "--quiet")
	var usage *core.UsageError
	if !errors.As(err, &usage) {
		t.Fatalf("--quiet without -c: %v", err)
	}
}

func TestMeta(t *testing.T) {
	for _, tl := range []Tool{md5Tool, b2Tool} {
		m := tl.Meta()
		if m.Name != tl.Name || !strings.HasPrefix(m.Use, tl.Name+" ") {
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
		for _, s := range m.SeeAlso {
			if s == tl.Name {
				t.Fatalf("%s lists itself in SeeAlso", tl.Name)
			}
		}
	}
	if !strings.Contains(md5Tool.Meta().Long, "broken") || strings.Contains(b2Tool.Meta().Long, "broken") {
		t.Fatal("Weak note misplaced")
	}
	if !strings.Contains(b2Tool.Meta().Long, "-l BITS") {
		t.Fatal("b2sumx does not document -l")
	}
}
