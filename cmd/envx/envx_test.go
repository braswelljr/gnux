package envx

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/env"
)

// parse runs args through a flag set configured the way the tool's is.
func parse(t *testing.T, args []string) *pflag.FlagSet {
	t.Helper()
	fs := pflag.NewFlagSet(env.Name, pflag.ContinueOnError)
	fs.SetInterspersed(false)
	Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %q: %v", args, err)
	}
	return fs
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		want     env.Options
		wantArgs []string
	}{
		{"defaults", nil, env.Options{}, nil},
		{"-i", []string{"-i"}, env.Options{IgnoreEnvironment: true}, nil},
		{"--ignore-environment", []string{"--ignore-environment"}, env.Options{IgnoreEnvironment: true}, nil},
		{"-u repeats", []string{"-u", "A", "-uB", "--unset=C"}, env.Options{Unset: []string{"A", "B", "C"}}, nil},
		{"-u keeps commas", []string{"-u", "A,B"}, env.Options{Unset: []string{"A,B"}}, nil},
		{"-0", []string{"-0"}, env.Options{Null: true}, nil},
		{"--null", []string{"--null"}, env.Options{Null: true}, nil},
		{"-C", []string{"-C", "/tmp"}, env.Options{Chdir: "/tmp"}, nil},
		{"--chdir", []string{"--chdir=/tmp"}, env.Options{Chdir: "/tmp"}, nil},
		{"-a", []string{"-a", "name"}, env.Options{Argv0: "name"}, nil},
		{"--argv0", []string{"--argv0=name"}, env.Options{Argv0: "name"}, nil},
		{"-v", []string{"-v"}, env.Options{Debug: true}, nil},
		{"--debug", []string{"--debug"}, env.Options{Debug: true}, nil},
		{"cluster", []string{"-i0v"}, env.Options{IgnoreEnvironment: true, Null: true, Debug: true}, nil},
		{"options stop at an assignment", []string{"-i", "A=1", "-u", "B"},
			env.Options{IgnoreEnvironment: true}, []string{"A=1", "-u", "B"}},
		{"options stop at the command", []string{"-u", "A", "cmd", "-i", "--null"},
			env.Options{Unset: []string{"A"}}, []string{"cmd", "-i", "--null"}},
		{"options stop at a lone dash", []string{"-", "-i"}, env.Options{}, []string{"-", "-i"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := parse(t, tt.args)
			if got := buildOptions(fs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildOptions(%q) = %+v, want %+v", tt.args, got, tt.want)
			}
			if got := fs.Args(); len(got) != len(tt.wantArgs) || (len(got) > 0 && !reflect.DeepEqual(got, tt.wantArgs)) {
				t.Errorf("operands = %q, want %q", got, tt.wantArgs)
			}
		})
	}
}

func TestExpandSplit(t *testing.T) {
	lookup := func(name string) (string, bool) {
		if name == "DIR" {
			return "/opt", true
		}
		return "", false
	}
	tests := []struct {
		name     string
		args     []string
		want     env.Options
		wantArgs []string
	}{
		{"no -S", []string{"-i", "cmd"}, env.Options{IgnoreEnvironment: true}, []string{"cmd"}},
		{"shebang line", []string{"-S perl -w", "script", "arg"}, env.Options{}, []string{"perl", "-w", "script", "arg"}},
		{"options inside the string", []string{"-S", "-i -u A B=1 cmd -x", "script"},
			env.Options{IgnoreEnvironment: true, Unset: []string{"A"}}, []string{"B=1", "cmd", "-x", "script"}},
		{"options before and inside merge", []string{"-u", "A", "-C", "/a", "-S", "-u B -C /b -v cmd"},
			env.Options{Unset: []string{"A", "B"}, Chdir: "/b", Debug: true}, []string{"cmd"}},
		{"combined with -v", []string{"-vS cmd 'a b'"}, env.Options{Debug: true}, []string{"cmd", "a b"}},
		{"long form", []string{"--split-string=A=1 cmd"}, env.Options{}, []string{"A=1", "cmd"}},
		{"variable expansion", []string{"-S", "${DIR}/bin/tool ${NOPE}"}, env.Options{}, []string{"/opt/bin/tool"}},
		{"nested -S", []string{"-S", `-i -S "-0 A=1"`}, env.Options{IgnoreEnvironment: true, Null: true}, []string{"A=1"}},
		{"empty string", []string{"-S", "", "cmd"}, env.Options{}, []string{"cmd"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := parse(t, tt.args)
			opts, args, err := expandSplit(fs, fs.Args(), lookup)
			if err != nil {
				t.Fatalf("expandSplit(%q): %v", tt.args, err)
			}
			if !reflect.DeepEqual(opts, tt.want) {
				t.Errorf("options = %+v, want %+v", opts, tt.want)
			}
			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("operands = %q, want %q", args, tt.wantArgs)
			}
		})
	}
}

// execute runs the tool's command end to end against buffers.
func execute(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	// cobra falls back to os.Args for a nil slice.
	c.SetArgs(append([]string{}, args...))
	err = c.Execute()
	return out.String(), errb.String(), err
}

func TestCommandPrints(t *testing.T) {
	t.Setenv("GNUX_ENVX_TEST", "inherited")
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"-i with assignments", []string{"-i", "A=1", "B=2"}, "A=1\nB=2\n"},
		{"lone dash", []string{"-", "A=1"}, "A=1\n"},
		{"-0", []string{"-i", "-0", "A=1", "B=2"}, "A=1\x00B=2\x00"},
		{"-S supplies options and assignments", []string{"-S", "-i A=1 'B=two words'"}, "A=1\nB=two words\n"},
		{"-S expands variables", []string{"-i", "-S", "A=${GNUX_ENVX_TEST}"}, "A=inherited\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := execute(t, tt.args...)
			if err != nil {
				t.Fatalf("Execute(%q): %v", tt.args, err)
			}
			if stdout != tt.want {
				t.Errorf("stdout = %q, want %q", stdout, tt.want)
			}
			if stderr != "" {
				t.Errorf("stderr = %q, want empty", stderr)
			}
		})
	}

	t.Run("inherits and unsets the real environment", func(t *testing.T) {
		stdout, _, err := execute(t)
		if err != nil || !strings.Contains(stdout, "GNUX_ENVX_TEST=inherited\n") {
			t.Fatalf("stdout lacks the inherited variable (err %v)", err)
		}
		stdout, _, err = execute(t, "-u", "GNUX_ENVX_TEST")
		if err != nil || strings.Contains(stdout, "GNUX_ENVX_TEST=") {
			t.Fatalf("-u did not remove the variable (err %v)", err)
		}
	})
}

func TestCommandErrors(t *testing.T) {
	t.Run("usage errors exit 125", func(t *testing.T) {
		if got := Spec().UsageExit; got != 125 {
			t.Fatalf("UsageExit = %d, want 125", got)
		}
		for _, args := range [][]string{
			{"-0", "cmd"},
			{"-C", "."},
			{"--no-such-option"},
			{"-S", "--no-such-option"},
		} {
			_, _, err := execute(t, args...)
			var exit *core.ExitError
			if err == nil || errors.Is(err, core.ErrFailed) || errors.As(err, &exit) {
				t.Errorf("Execute(%q) = %v, want a usage error", args, err)
			}
		}
	})

	t.Run("null with a command", func(t *testing.T) {
		_, _, err := execute(t, "-0", "cmd")
		var usage *core.UsageError
		if want := "cannot specify --null (-0) with command"; !errors.As(err, &usage) || usage.Msg != want {
			t.Fatalf("err = %v, want usage error %q", err, want)
		}
	})

	t.Run("malformed -S", func(t *testing.T) {
		stdout, stderr, err := execute(t, "-S", "a 'b")
		var exit *core.ExitError
		if !errors.As(err, &exit) || exit.Code != 125 {
			t.Fatalf("err = %v, want exit status 125", err)
		}
		if want := "envx: no terminating quote in -S string\n"; stderr != want || stdout != "" {
			t.Fatalf("stdout = %q, stderr = %q, want stderr %q", stdout, stderr, want)
		}
	})

	t.Run("command not found", func(t *testing.T) {
		stdout, stderr, err := execute(t, "-i", "gnux-no-such-command", "--flag")
		var exit *core.ExitError
		if !errors.As(err, &exit) || exit.Code != 127 {
			t.Fatalf("err = %v, want exit status 127", err)
		}
		if want := "envx: 'gnux-no-such-command': No such file or directory\n"; stderr != want || stdout != "" {
			t.Fatalf("stdout = %q, stderr = %q, want stderr %q", stdout, stderr, want)
		}
	})
}
