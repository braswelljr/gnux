package xargsx

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/xargs"
	"github.com/braswelljr/gnux/internal/tool"
)

// parse reads args as the tool does.
func parse(t *testing.T, args []string) *pflag.FlagSet {
	t.Helper()
	fs, err := parseArgs(args)
	if err != nil {
		t.Fatalf("parse %q: %v", args, err)
	}
	return fs
}

// The warnings GNU prints for an option another one overrides.
const (
	warnReplaceArgs  = "warning: options --max-args and --replace/-I/-i are mutually exclusive, ignoring previous --max-args value"
	warnReplaceLines = "warning: options --max-lines and --replace/-I/-i are mutually exclusive, ignoring previous --max-lines value"
	warnLinesArgs    = "warning: options --max-args and --max-lines/-l/-L are mutually exclusive, ignoring previous --max-args value"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		want     xargs.Options
		warnings []string
		wantArgs []string
	}{
		{"defaults", nil, xargs.Options{}, nil, nil},
		{"-0", []string{"-0"}, xargs.Options{Null: true}, nil, nil},
		{"--null", []string{"--null"}, xargs.Options{Null: true}, nil, nil},
		{"-a", []string{"-a", "items"}, xargs.Options{ArgFile: "items"}, nil, nil},
		{"--arg-file", []string{"--arg-file=items"}, xargs.Options{ArgFile: "items"}, nil, nil},
		{"-d", []string{"-d", `\n`}, xargs.Options{Delimiter: `\n`}, nil, nil},
		{"--delimiter", []string{"--delimiter=,"}, xargs.Options{Delimiter: ","}, nil, nil},
		{"-E", []string{"-E", "END"}, xargs.Options{EOF: "END"}, nil, nil},
		{"-eEND", []string{"-eEND"}, xargs.Options{EOF: "END"}, nil, nil},
		{"--eof=END", []string{"--eof=END"}, xargs.Options{EOF: "END"}, nil, nil},
		{"-e alone is no end string", []string{"-e"}, xargs.Options{}, nil, nil},
		{"--eof alone is no end string", []string{"--eof"}, xargs.Options{}, nil, nil},
		{"-e does not take the next word", []string{"-e", "cmd"}, xargs.Options{}, nil, []string{"cmd"}},
		{"-E wins over -e", []string{"-E", "A", "-eB"}, xargs.Options{EOF: "A"}, nil, nil},
		{"-e loses to -E", []string{"-eB", "-E", "A"}, xargs.Options{EOF: "A"}, nil, nil},
		{"-I", []string{"-I", "{}"}, xargs.Options{Replace: "{}"}, nil, nil},
		{"-i alone is {}", []string{"-i"}, xargs.Options{Replace: "{}"}, nil, nil},
		{"-iX", []string{"-iX"}, xargs.Options{Replace: "X"}, nil, nil},
		{"--replace alone is {}", []string{"--replace"}, xargs.Options{Replace: "{}"}, nil, nil},
		{"--replace=X", []string{"--replace=X"}, xargs.Options{Replace: "X"}, nil, nil},
		{"-i does not take the next word", []string{"-i", "cmd", "{}"}, xargs.Options{Replace: "{}"}, nil, []string{"cmd", "{}"}},
		{"-I wins over -i", []string{"-I", "A", "-iB"}, xargs.Options{Replace: "A"}, nil, nil},
		{"-i loses to -I", []string{"-iB", "-I", "A"}, xargs.Options{Replace: "A"}, nil, nil},
		{"-L", []string{"-L", "2"}, xargs.Options{MaxLines: 2}, nil, nil},
		{"-l alone is 1", []string{"-l"}, xargs.Options{MaxLines: 1}, nil, nil},
		{"-l3", []string{"-l3"}, xargs.Options{MaxLines: 3}, nil, nil},
		{"--max-lines alone is 1", []string{"--max-lines"}, xargs.Options{MaxLines: 1}, nil, nil},
		{"--max-lines=4", []string{"--max-lines=4"}, xargs.Options{MaxLines: 4}, nil, nil},
		{"-l does not take the next word", []string{"-l", "5"}, xargs.Options{MaxLines: 1}, nil, []string{"5"}},
		{"-L wins over -l", []string{"-L", "2", "-l5"}, xargs.Options{MaxLines: 2}, nil, nil},
		{"-l loses to -L", []string{"-l5", "-L", "2"}, xargs.Options{MaxLines: 2}, nil, nil},
		{"-n", []string{"-n", "3"}, xargs.Options{MaxArgs: 3}, nil, nil},
		{"--max-args", []string{"--max-args=3"}, xargs.Options{MaxArgs: 3}, nil, nil},
		{"-s", []string{"-s", "100"}, xargs.Options{MaxChars: 100}, nil, nil},
		{"--max-chars", []string{"--max-chars=100"}, xargs.Options{MaxChars: 100}, nil, nil},
		{"-x", []string{"-x"}, xargs.Options{Exit: true}, nil, nil},
		{"--exit", []string{"--exit"}, xargs.Options{Exit: true}, nil, nil},
		{"-r", []string{"-r"}, xargs.Options{NoRunIfEmpty: true}, nil, nil},
		{"--no-run-if-empty", []string{"--no-run-if-empty"}, xargs.Options{NoRunIfEmpty: true}, nil, nil},
		{"-t", []string{"-t"}, xargs.Options{Verbose: true}, nil, nil},
		{"--verbose", []string{"--verbose"}, xargs.Options{Verbose: true}, nil, nil},
		{"-p", []string{"-p"}, xargs.Options{Interactive: true}, nil, nil},
		{"--interactive", []string{"--interactive"}, xargs.Options{Interactive: true}, nil, nil},
		{"-P", []string{"-P", "4"}, xargs.Options{MaxProcs: 4}, nil, nil},
		{"--max-procs", []string{"--max-procs=4"}, xargs.Options{MaxProcs: 4}, nil, nil},
		{"-P 1", []string{"-P", "1"}, xargs.Options{MaxProcs: 1}, nil, nil},
		{"-P 0 is unlimited", []string{"-P", "0"}, xargs.Options{MaxProcs: xargs.Unlimited}, nil, nil},
		{"huge numbers are capped", []string{"-P", "99999999999999999999999"}, xargs.Options{MaxProcs: 1<<31 - 1}, nil, nil},
		{"--show-limits", []string{"--show-limits"}, xargs.Options{ShowLimits: true}, nil, nil},
		{"-o", []string{"-o"}, xargs.Options{OpenTTY: true}, nil, nil},
		{"--open-tty", []string{"--open-tty"}, xargs.Options{OpenTTY: true}, nil, nil},
		{"--process-slot-var", []string{"--process-slot-var=SLOT"}, xargs.Options{ProcessSlotVar: "SLOT"}, nil, nil},
		{"cluster", []string{"-0rtn2"}, xargs.Options{Null: true, NoRunIfEmpty: true, Verbose: true, MaxArgs: 2}, nil, nil},

		// -0 is the safer separator, in either order.
		{"-0 then -d", []string{"-0", "-d", ","}, xargs.Options{Null: true}, nil, nil},
		{"-d then -0", []string{"-d", ",", "-0"}, xargs.Options{Null: true}, nil, nil},

		// The most cautious of -I, -L and -n wins, in either order.
		{"-I then -n", []string{"-I", "{}", "-n", "2"}, xargs.Options{Replace: "{}"}, []string{warnReplaceArgs}, nil},
		{"-n then -I", []string{"-n", "2", "-I", "{}"}, xargs.Options{Replace: "{}"}, []string{warnReplaceArgs}, nil},
		{"-I then -L", []string{"-I", "{}", "-L", "2"}, xargs.Options{Replace: "{}"}, []string{warnReplaceLines}, nil},
		{"-L then -I", []string{"-L", "2", "-I", "{}"}, xargs.Options{Replace: "{}"}, []string{warnReplaceLines}, nil},
		{"-i then -l", []string{"-i", "-l"}, xargs.Options{Replace: "{}"}, []string{warnReplaceLines}, nil},
		{"-l then -i", []string{"-l", "-i"}, xargs.Options{Replace: "{}"}, []string{warnReplaceLines}, nil},
		{"-L then -n", []string{"-L", "2", "-n", "3"}, xargs.Options{MaxLines: 2}, []string{warnLinesArgs}, nil},
		{"-n then -L", []string{"-n", "3", "-L", "2"}, xargs.Options{MaxLines: 2}, []string{warnLinesArgs}, nil},
		{"-I, -L and -n", []string{"-n", "3", "-L", "2", "-I", "{}"}, xargs.Options{Replace: "{}"},
			[]string{warnReplaceArgs, warnReplaceLines}, nil},
		{"-n, -I and -L", []string{"-I", "{}", "-L", "2", "-n", "3"}, xargs.Options{Replace: "{}"},
			[]string{warnReplaceArgs, warnReplaceLines}, nil},

		// Attached optional arguments, alone and inside a cluster.
		{"-ti{}", []string{"-ti{}"}, xargs.Options{Verbose: true, Replace: "{}"}, nil, nil},
		{"-rl2", []string{"-rl2", "cmd"}, xargs.Options{NoRunIfEmpty: true, MaxLines: 2}, nil, []string{"cmd"}},
		{"-i=X keeps the equals sign", []string{"-i=X"}, xargs.Options{Replace: "=X"}, nil, nil},
		{"-n2 attached", []string{"-n2", "-s100"}, xargs.Options{MaxArgs: 2, MaxChars: 100}, nil, nil},
		{"a value that looks like an option", []string{"-E", "-i", "-d", "-l"}, xargs.Options{EOF: "-i", Delimiter: "-l"}, nil, nil},
		{"a long option's value is not the command", []string{"--max-args", "2", "-iX", "cmd"},
			xargs.Options{Replace: "X"}, []string{warnReplaceArgs}, []string{"cmd"}},
		{"the command's own options are left alone", []string{"-t", "cmd", "-l3", "-iX"},
			xargs.Options{Verbose: true}, nil, []string{"cmd", "-l3", "-iX"}},
		{"a lone dash is the command", []string{"-", "-l3"}, xargs.Options{}, nil, []string{"-", "-l3"}},

		// Options end at the command.
		{"options stop at the command", []string{"-n", "1", "rm", "-r", "-t", "--null"},
			xargs.Options{MaxArgs: 1}, nil, []string{"rm", "-r", "-t", "--null"}},
		{"options stop at --", []string{"-t", "--", "-n", "1"}, xargs.Options{Verbose: true}, nil, []string{"-n", "1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := parse(t, tt.args)
			got, warnings, err := buildOptions(fs)
			if err != nil {
				t.Fatalf("buildOptions(%q): %v", tt.args, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildOptions(%q) = %+v, want %+v", tt.args, got, tt.want)
			}
			if !reflect.DeepEqual(warnings, tt.warnings) {
				t.Errorf("warnings = %q, want %q", warnings, tt.warnings)
			}
			if got := fs.Args(); len(got) != len(tt.wantArgs) || (len(got) > 0 && !reflect.DeepEqual(got, tt.wantArgs)) {
				t.Errorf("operands = %q, want %q", got, tt.wantArgs)
			}
		})
	}
}

func TestBuildOptionsErrors(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"-n", "x"}, `invalid number "x" for -n option`},
		{[]string{"--max-args="}, `invalid number "" for -n option`},
		{[]string{"-n", "2k"}, `invalid number "2k" for -n option`},
		{[]string{"-n", "0"}, "value 0 for -n option should be >= 1"},
		{[]string{"-n", "-3"}, "value -3 for -n option should be >= 1"},
		{[]string{"-L", "x"}, `invalid number "x" for -L option`},
		{[]string{"-L", "0"}, "value 0 for -L option should be >= 1"},
		{[]string{"-lx"}, `invalid number "x" for -l option`},
		{[]string{"-l0"}, "value 0 for -l option should be >= 1"},
		{[]string{"-s", "x"}, `invalid number "x" for -s option`},
		{[]string{"-s", "0"}, "value 0 for -s option should be >= 1"},
		{[]string{"-P", "x"}, `invalid number "x" for -P option`},
		{[]string{"-P", "-1"}, "value -1 for -P option should be >= 0"},
		{[]string{"-P", "-99999999999999999999999"}, "value -99999999999999999999999 for -P option should be >= 0"},
	}
	for _, tt := range tests {
		_, _, err := buildOptions(parse(t, tt.args))
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != tt.want {
			t.Errorf("buildOptions(%q) error = %v, want the usage error %q", tt.args, err, tt.want)
		}
	}
}

func TestRun(t *testing.T) {
	// The registry lives in package cmd, which cannot be imported here; stand
	// in for it the way cmd does, by setting tool.RunTool.
	saved := tool.RunTool
	t.Cleanup(func() { tool.RunTool = saved })
	var ran [][]string
	tool.RunTool = func(name string, args []string, s core.Streams) (int, bool) {
		ran = append(ran, append([]string{name}, args...))
		return 0, true
	}

	var out, errb bytes.Buffer
	s := core.Streams{In: strings.NewReader("a\nb c\n"), Out: &out, Err: &errb}
	if err := run(nil, []string{"-t", "-n", "5", "-I", "{}", "rm", "-f", "{}"}, s); err != nil {
		t.Fatal(err)
	}
	if want := [][]string{{"rm", "-f", "a"}, {"rm", "-f", "b c"}}; !reflect.DeepEqual(ran, want) {
		t.Errorf("ran %q, want %q", ran, want)
	}
	want := "xargsx: " + warnReplaceArgs + "\nrm -f a\nrm -f 'b c'\n"
	if errb.String() != want {
		t.Errorf("stderr = %q, want %q", errb.String(), want)
	}

	var usage *core.UsageError
	for _, args := range [][]string{{"-n", "many"}, {"-Z"}, {"--no-such-option"}, {"-n"}} {
		if err := run(nil, args, s); !errors.As(err, &usage) {
			t.Errorf("run(%q) = %v, want a usage error", args, err)
		}
	}

	out.Reset()
	if err := run(nil, []string{"-t", "--version", "rm"}, s); err != nil || !strings.HasPrefix(out.String(), "xargsx (gnux) ") {
		t.Errorf("--version printed %q, err = %v", out.String(), err)
	}
}
