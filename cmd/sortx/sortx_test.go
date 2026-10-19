package sortx

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/sort"
)

// options registers every flag, parses args and returns the resolved Options.
func options(t *testing.T, args ...string) (sort.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(sort.Name, pflag.ContinueOnError)
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
		want sort.Options
	}{
		{"defaults", nil, sort.Options{}},
		{"ignore leading blanks", []string{"-b"}, sort.Options{IgnoreLeadingBlanks: true}},
		{"dictionary order", []string{"--dictionary-order"}, sort.Options{DictionaryOrder: true}},
		{"ignore case", []string{"-f"}, sort.Options{IgnoreCase: true}},
		{"ignore nonprinting", []string{"-i"}, sort.Options{IgnoreNonprinting: true}},
		{"general numeric", []string{"-g"}, sort.Options{GeneralNumeric: true}},
		{"human numeric", []string{"--human-numeric-sort"}, sort.Options{HumanNumeric: true}},
		{"month", []string{"-M"}, sort.Options{Month: true}},
		{"numeric", []string{"-n"}, sort.Options{Numeric: true}},
		{"version", []string{"-V"}, sort.Options{Version: true}},
		{"reverse", []string{"-r"}, sort.Options{Reverse: true}},
		{"combined", []string{"-rnu"}, sort.Options{Reverse: true, Numeric: true, Unique: true}},
		{"sort=numeric", []string{"--sort=numeric"}, sort.Options{Numeric: true}},
		{"sort=general-numeric", []string{"--sort=general-numeric"}, sort.Options{GeneralNumeric: true}},
		{"sort=human-numeric", []string{"--sort=human-numeric"}, sort.Options{HumanNumeric: true}},
		{"sort=month", []string{"--sort", "month"}, sort.Options{Month: true}},
		{"sort=version", []string{"--sort=version"}, sort.Options{Version: true}},
		{"random", []string{"-R"}, sort.Options{Random: true}},
		{"sort=random", []string{"--sort=random"}, sort.Options{Random: true}},
		{"random source", []string{"-R", "--random-source=seed"}, sort.Options{Random: true, RandomSource: "seed"}},
		{"keys in order", []string{"-k2,2n", "--key", "1"}, sort.Options{Keys: []string{"2,2n", "1"}}},
		{"separator", []string{"-t", ":"}, sort.Options{FieldSeparator: ":"}},
		{"unique", []string{"--unique"}, sort.Options{Unique: true}},
		{"stable", []string{"-s"}, sort.Options{Stable: true}},
		{"merge", []string{"-m"}, sort.Options{Merge: true}},
		{"output", []string{"-o", "out"}, sort.Options{Output: "out"}},
		{"zero terminated", []string{"-z"}, sort.Options{ZeroTerminated: true}},
		{"files0-from", []string{"--files0-from=list"}, sort.Options{Files0From: "list"}},
		{"check", []string{"-c"}, sort.Options{Check: sort.CheckDiagnose}},
		{"check combined", []string{"-cn"}, sort.Options{Check: sort.CheckDiagnose, Numeric: true}},
		{"check=diagnose-first", []string{"--check=diagnose-first"}, sort.Options{Check: sort.CheckDiagnose}},
		{"check=quiet", []string{"--check=quiet"}, sort.Options{Check: sort.CheckQuiet}},
		{"check=silent", []string{"--check=silent"}, sort.Options{Check: sort.CheckQuiet}},
		{"check quiet", []string{"-C"}, sort.Options{Check: sort.CheckQuiet}},
		// Conflicts resolve toward reporting the disorder, in either order.
		{"-c then -C", []string{"-c", "-C"}, sort.Options{Check: sort.CheckDiagnose}},
		{"-C then -c", []string{"-C", "-c"}, sort.Options{Check: sort.CheckDiagnose}},
		{"quiet check then -C", []string{"--check=quiet", "-C"}, sort.Options{Check: sort.CheckQuiet}},
		{"ignored tuning options", []string{"-S", "1G", "-T", "/tmp", "--parallel=4", "--batch-size=8", "--compress-program=gzip"}, sort.Options{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil {
				t.Fatalf("buildOptions(%v): %v", tt.args, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestBuildOptionsInvalid(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"--sort=shuffle"}, "invalid argument 'shuffle' for '--sort'"},
		{[]string{"--sort=sideways"}, "invalid argument 'sideways' for '--sort'"},
		{[]string{"--check=loudly"}, "invalid argument 'loudly' for '--check'"},
	}
	for _, tt := range tests {
		_, err := options(t, tt.args...)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != tt.want {
			t.Fatalf("%v: err = %v, want a UsageError %q", tt.args, err, tt.want)
		}
	}
}

// TestCommand runs the tool the way the binary does, through its cobra command.
func TestCommand(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		want   string
		stderr string
		status int
	}{
		{"default", nil, "a 2\nb 10\nc 1\n", "", 0},
		{"key numeric", []string{"-k2n"}, "c 1\na 2\nb 10\n", "", 0},
		{"reverse", []string{"-r"}, "c 1\nb 10\na 2\n", "", 0},
		{"check", []string{"-c"}, "", "sortx: -:2: disorder: a 2\n", 1},
		{"incompatible", []string{"-ng"}, "", "sortx: options '-gn' are incompatible\n", 2},
		{"empty tab", []string{"-t", ""}, "", "sortx: empty tab\n", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			c := Spec().Command(core.Streams{In: strings.NewReader("b 10\na 2\nc 1\n"), Out: &out, Err: &errb})
			c.SetArgs(tt.args)
			err := c.Execute()
			status := 0
			var exit *core.ExitError
			if errors.As(err, &exit) {
				status = exit.Code
			} else if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if status != tt.status || out.String() != tt.want || errb.String() != tt.stderr {
				t.Fatalf("status %d, stdout %q, stderr %q; want %d, %q, %q",
					status, out.String(), errb.String(), tt.status, tt.want, tt.stderr)
			}
		})
	}
}

func TestSpec(t *testing.T) {
	if s := Spec(); s.GNU != "sort" || s.UsageExit != 2 {
		t.Fatalf("GNU = %q, UsageExit = %d", s.GNU, s.UsageExit)
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != sort.Name || !strings.HasPrefix(m.Use, sort.Name+" ") {
		t.Fatalf("Name = %q, Use = %q", m.Name, m.Use)
	}
	if m.Short == "" || m.Long == "" || m.Permissions == "" || len(m.UseCases) == 0 || len(m.ExitStatus) == 0 {
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
