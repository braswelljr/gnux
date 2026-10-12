package splitx

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/split"
)

func options(t *testing.T, args ...string) (split.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(split.Name, pflag.ContinueOnError)
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
		want split.Options
	}{
		{"defaults", nil, split.Options{}},
		{"-l", []string{"-l", "5"}, split.Options{Count: 5}},
		{"obsolete", []string{"-1000"}, split.Options{Count: 1000}},
		{"-b", []string{"-b", "2K"}, split.Options{Mode: split.Bytes, Count: 2048}},
		{"-b KB", []string{"-b", "1MB"}, split.Options{Mode: split.Bytes, Count: 1000000}},
		{"-C", []string{"-C", "10"}, split.Options{Mode: split.LineBytes, Count: 10}},
		{"-n", []string{"-n", "4"}, split.Options{Mode: split.ChunkBytes, Count: 4}},
		{"-n K/N", []string{"-n", "2/4"}, split.Options{Mode: split.ChunkBytes, Count: 4, Chunk: 2}},
		{"-n l/N", []string{"-n", "l/3"}, split.Options{Mode: split.ChunkLines, Count: 3}},
		{"-n l/K/N", []string{"-n", " l/1/3"}, split.Options{Mode: split.ChunkLines, Count: 3, Chunk: 1}},
		{"-n r/N", []string{"-n", "r/2"}, split.Options{Mode: split.RoundRobin, Count: 2}},
		{"-a", []string{"-a", "3"}, split.Options{SuffixLength: 3}},
		{"-d", []string{"-d"}, split.Options{Suffixes: split.Numeric}},
		{"-d then obsolete", []string{"-d5"}, split.Options{Suffixes: split.Numeric, Count: 5}},
		{"--numeric-suffixes=FROM", []string{"--numeric-suffixes=7"}, split.Options{Suffixes: split.Numeric, SuffixStart: "7"}},
		{"-x", []string{"-x"}, split.Options{Suffixes: split.Hex}},
		{"-d then -x", []string{"-d", "-x"}, split.Options{Suffixes: split.Hex}},
		{"-x then -d", []string{"--hex-suffixes=a", "-d"}, split.Options{Suffixes: split.Numeric}},
		{"--additional-suffix", []string{"--additional-suffix=.csv"}, split.Options{AdditionalSuffix: ".csv"}},
		{"-e --verbose -u", []string{"-e", "--verbose", "-u"}, split.Options{ElideEmpty: true, Verbose: true}},
		{"-t", []string{"-t", ","}, split.Options{Separator: ',', SeparatorSet: true}},
		{"-t NUL", []string{"-t", `\0`}, split.Options{SeparatorSet: true}},
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
	var usage *core.UsageError
	for _, args := range [][]string{{"-l", "5", "-b", "3"}, {"-5", "-n", "2"}} {
		if _, err := options(t, args...); !errors.As(err, &usage) || usage.Msg != "cannot split in more than one way" {
			t.Errorf("%v: %v", args, err)
		}
	}
	for _, tt := range []struct {
		args []string
		msg  string
	}{
		{[]string{"-l", "0"}, "invalid number of lines: '0'"},
		{[]string{"-l", "x"}, "invalid number of lines: 'x'"},
		{[]string{"-b", "1Q9"}, "invalid number of bytes: '1Q9'"},
		{[]string{"-n", "0"}, "invalid number of chunks: '0'"},
		{[]string{"-n", "5/4"}, "invalid chunk number: '5'"},
		{[]string{"-a", "x"}, "invalid suffix length: 'x'"},
		{[]string{"-t", "ab"}, "multi-character separator: 'ab'"},
		{[]string{"-t", ""}, "empty record separator"},
		{[]string{"-t", ",", "-t", ";"}, "multiple separator characters specified"},
	} {
		_, err := options(t, tt.args...)
		var fatal *fatalError
		if !errors.As(err, &fatal) || fatal.msg != tt.msg {
			t.Errorf("%v: %v, want %q", tt.args, err, tt.msg)
		}
	}
}

func TestCommand(t *testing.T) {
	t.Chdir(t.TempDir())
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader("1\n2\n3\n"), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs([]string{"-2", "-d", "--verbose", "-", "part-"})
	if err := c.Execute(); err != nil || out.String() != "creating file 'part-00'\ncreating file 'part-01'\n" {
		t.Fatalf("Execute: %v, stdout = %q", err, out.String())
	}
	if data, _ := os.ReadFile("part-01"); string(data) != "3\n" {
		t.Fatalf("part-01 = %q", data)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != split.Name || !strings.HasPrefix(m.Use, split.Name+" ") {
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
