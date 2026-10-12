package cksumx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/cksum"
)

func options(t *testing.T, args ...string) (cksum.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(cksum.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return buildOptions(fs)
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want cksum.Options
	}{
		{"defaults", nil, cksum.Options{}},
		{"-a", []string{"-a", "sha256"}, cksum.Options{Algorithm: "sha256"}},
		{"abbreviation", []string{"-a", "bl"}, cksum.Options{Algorithm: "blake2b"}},
		{"unknown left to the engine", []string{"-a", "foo"}, cksum.Options{Algorithm: "foo"}},
		{"case kept", []string{"-a", "SHA256"}, cksum.Options{Algorithm: "SHA256"}},
		{"untagged", []string{"--untagged"}, cksum.Options{Untagged: true}},
		{"tag wins", []string{"--untagged", "--tag"}, cksum.Options{}},
		{"tag wins before", []string{"--tag", "--untagged"}, cksum.Options{}},
		{"base64 raw zero", []string{"--base64", "--raw", "-z"}, cksum.Options{Base64: true, Raw: true, Zero: true}},
		{"-l", []string{"-a", "blake2b", "-l", "256"}, cksum.Options{Algorithm: "blake2b", Length: 256}},
		{"check flags", []string{"-c", "--ignore-missing", "--quiet", "--status", "--strict", "-w"},
			cksum.Options{Check: true, IgnoreMissing: true, Quiet: true, Status: true, Strict: true, Warn: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil || got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, %v; want %+v", tt.args, got, err, tt.want)
			}
		})
	}
}

func TestBuildOptionsInvalid(t *testing.T) {
	_, err := options(t, "-a", "sha")
	var usage *core.UsageError
	if !errors.As(err, &usage) || !strings.HasPrefix(usage.Msg, "ambiguous argument 'sha' for '--algorithm'") {
		t.Fatalf("-a sha: %v", err)
	}
	_, err = options(t, "-l", "x")
	var fatal *fatalError
	if !errors.As(err, &fatal) || fatal.msg != "invalid length: 'x'" {
		t.Fatalf("-l x: %v", err)
	}
}

func TestCommand(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{nil, "1219131554 3\n"},
		{[]string{"-a", "md5"}, "MD5 (-) = 900150983cd24fb0d6963f7d28e17f72\n"},
		{[]string{"-a", "md5", "--untagged"}, "900150983cd24fb0d6963f7d28e17f72  -\n"},
	}
	for _, tt := range tests {
		var out, errb bytes.Buffer
		c := Spec().Command(core.Streams{In: strings.NewReader("abc"), Out: &out, Err: &errb})
		c.SetArgs(tt.args)
		if err := c.Execute(); err != nil || out.String() != tt.want {
			t.Errorf("%v: %v, stdout = %q (stderr %q), want %q", tt.args, err, out.String(), errb.String(), tt.want)
		}
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != cksum.Name || !strings.HasPrefix(m.Use, cksum.Name+" ") {
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
