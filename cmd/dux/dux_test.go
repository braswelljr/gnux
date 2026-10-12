package dux

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
	"github.com/braswelljr/gnux/core/du"
)

func options(t *testing.T, args ...string) (du.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(du.Name, pflag.ContinueOnError)
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
		want du.Options
	}{
		{"defaults", nil, du.Options{}},
		{"-sh", []string{"-sh"}, du.Options{Summarize: true, BlockSize: "human-readable"}},
		{"--si", []string{"--si"}, du.Options{BlockSize: "si"}},
		{"-k", []string{"-k"}, du.Options{BlockSize: "1K"}},
		{"-m", []string{"-m"}, du.Options{BlockSize: "1M"}},
		{"-b", []string{"-b"}, du.Options{Apparent: true, BlockSize: "1"}},
		{"-b then -h keeps apparent", []string{"-b", "-h"}, du.Options{Apparent: true, BlockSize: "human-readable"}},
		{"-h then -k", []string{"-h", "-k"}, du.Options{BlockSize: "1K"}},
		{"-BM", []string{"-BM"}, du.Options{BlockSize: "M"}},
		{"-A", []string{"-A"}, du.Options{Apparent: true}},
		{"flags", []string{"-a0cSlx", "--inodes"}, du.Options{All: true, Null: true, Total: true, SeparateDirs: true, CountLinks: true, OneFileSystem: true, Inodes: true}},
		{"-d", []string{"-d", "2"}, du.Options{MaxDepth: 2, MaxDepthSet: true}},
		{"-L", []string{"-L"}, du.Options{Deref: du.DerefAll}},
		{"-H", []string{"-H"}, du.Options{Deref: du.DerefArgs}},
		{"-D", []string{"-D"}, du.Options{Deref: du.DerefArgs}},
		{"-P beats -L", []string{"-L", "-P"}, du.Options{}},
		{"-H beats -L", []string{"-H", "-L"}, du.Options{Deref: du.DerefArgs}},
		{"-t", []string{"-t", "100M"}, du.Options{Threshold: 100 << 20}},
		{"-t KB", []string{"-t", "2KB"}, du.Options{Threshold: 2000}},
		{"-t negative", []string{"-t", "-1k"}, du.Options{Threshold: -1024}},
		{"--time", []string{"--time"}, du.Options{Time: "mtime"}},
		{"--time=access", []string{"--time=access"}, du.Options{Time: "atime"}},
		{"--time=st", []string{"--time=st"}, du.Options{Time: "ctime"}},
		{"--time-style", []string{"--time-style=iso"}, du.Options{TimeStyle: "iso"}},
		{"excludes", []string{"--exclude=*.o", "-X", "list"}, du.Options{Exclude: []string{"*.o"}, ExcludeFrom: []string{"list"}}},
		{"--files0-from", []string{"--files0-from=-"}, du.Options{Files0From: "-"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Exclude) == 0 {
				got.Exclude = nil
			}
			if len(got.ExcludeFrom) == 0 {
				got.ExcludeFrom = nil
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestBuildOptionsInvalid(t *testing.T) {
	var usage *core.UsageError
	if _, err := options(t, "-d", "x"); !errors.As(err, &usage) || usage.Msg != "invalid maximum depth 'x'" {
		t.Errorf("-d x: %v", err)
	}
	if _, err := options(t, "--time=birth"); !errors.As(err, &usage) {
		t.Errorf("--time=birth: %v", err)
	}
	for _, tt := range []struct{ arg, msg string }{
		{"x", "invalid --threshold argument 'x'"},
		{"-0", "invalid --threshold argument '-0'"},
		{"5X", "invalid --threshold argument '5X'"},
	} {
		_, err := options(t, "-t", tt.arg)
		var fatal *fatalError
		if !errors.As(err, &fatal) || fatal.msg != tt.msg {
			t.Errorf("-t %s: %v", tt.arg, err)
		}
	}
}

func TestCommand(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f"), bytes.Repeat([]byte("x"), 2048), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs([]string{"-sb", dir})
	if err := c.Execute(); err != nil || out.String() != "2048\t"+dir+"\n" {
		t.Fatalf("Execute: %v, stdout = %q", err, out.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != du.Name || !strings.HasPrefix(m.Use, du.Name+" ") {
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
