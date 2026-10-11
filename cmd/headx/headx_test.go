package headx

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/head"
)

// options registers every flag, parses args and returns the resolved Options.
func options(t *testing.T, args ...string) (head.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(head.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return buildOptions(fs)
}

func TestBuildOptions(t *testing.T) {
	lines := func(n int64) head.Options { return head.Options{Count: n} }
	bytesOpt := func(n int64) head.Options { return head.Options{Count: n, Bytes: true} }

	tests := []struct {
		name string
		args []string
		want head.Options
	}{
		{"defaults", nil, lines(10)},

		{"lines", []string{"-n", "3"}, lines(3)},
		{"lines attached", []string{"-n3"}, lines(3)},
		{"lines long", []string{"--lines=3"}, lines(3)},
		{"lines long, separate", []string{"--lines", "3"}, lines(3)},
		{"lines zero", []string{"-n", "0"}, lines(0)},
		{"lines explicit plus", []string{"-n", "+3"}, lines(3)},
		{"all but last lines", []string{"-n", "-3"}, head.Options{Count: 3, FromEnd: true}},
		{"all but last lines, attached", []string{"-n-3"}, head.Options{Count: 3, FromEnd: true}},
		{"all but last lines, long", []string{"--lines=-3"}, head.Options{Count: 3, FromEnd: true}},

		{"bytes", []string{"-c", "5"}, bytesOpt(5)},
		{"bytes attached", []string{"-c5"}, bytesOpt(5)},
		{"bytes long", []string{"--bytes=5"}, bytesOpt(5)},
		{"all but last bytes", []string{"-c", "-5"}, head.Options{Count: 5, Bytes: true, FromEnd: true}},

		{"suffix b", []string{"-c", "2b"}, bytesOpt(1024)},
		{"suffix K", []string{"-c", "1K"}, bytesOpt(1024)},
		{"suffix KiB", []string{"-c", "1KiB"}, bytesOpt(1024)},
		{"suffix kB", []string{"-c", "1kB"}, bytesOpt(1000)},
		{"suffix M", []string{"-c", "2M"}, bytesOpt(2 << 20)},
		{"suffix MB", []string{"-c", "2MB"}, bytesOpt(2_000_000)},
		{"suffix G", []string{"-c", "1G"}, bytesOpt(1 << 30)},
		{"suffix GB", []string{"-c", "1GB"}, bytesOpt(1_000_000_000)},
		{"suffix on lines", []string{"-n", "1K"}, lines(1024)},
		{"suffix with all-but-last", []string{"-n", "-1K"}, head.Options{Count: 1024, FromEnd: true}},
		{"overflow is clamped", []string{"-c", "1Y"}, bytesOpt(math.MaxInt64)},

		{"quiet", []string{"-q"}, head.Options{Count: 10, Quiet: true}},
		{"quiet long", []string{"--quiet"}, head.Options{Count: 10, Quiet: true}},
		{"silent", []string{"--silent"}, head.Options{Count: 10, Quiet: true}},
		{"verbose", []string{"-v"}, head.Options{Count: 10, Verbose: true}},
		{"verbose long", []string{"--verbose"}, head.Options{Count: 10, Verbose: true}},
		{"zero terminated", []string{"-z"}, head.Options{Count: 10, ZeroTerminated: true}},
		{"zero terminated long", []string{"--zero-terminated"}, head.Options{Count: 10, ZeroTerminated: true}},
		{"combined", []string{"-qzn", "2"}, head.Options{Count: 2, Quiet: true, ZeroTerminated: true}},

		// The last of -n, -c and -NUM wins, as in GNU head.
		{"bytes after lines", []string{"-n", "3", "-c", "5"}, bytesOpt(5)},
		{"lines after bytes", []string{"-c", "5", "-n", "3"}, lines(3)},
		{"later lines replace earlier", []string{"-n", "3", "-n", "-4"}, head.Options{Count: 4, FromEnd: true}},

		{"obsolete -NUM", []string{"-5"}, lines(5)},
		{"obsolete multi-digit", []string{"-120"}, lines(120)},
		{"obsolete zero", []string{"-0"}, lines(0)},
		{"obsolete with other flags", []string{"-5q"}, head.Options{Count: 5, Quiet: true}},
		{"obsolete then -n", []string{"-5", "-n", "3"}, lines(3)},
		{"-n then obsolete", []string{"-n", "3", "-5"}, lines(5)},
		{"-c then obsolete", []string{"-c", "3", "-7"}, lines(7)},
		{"a second obsolete number starts over", []string{"-5", "-n", "3", "-12"}, lines(12)},
		{"a negative value is not the obsolete form", []string{"-n", "-5"}, head.Options{Count: 5, FromEnd: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil {
				t.Fatalf("buildOptions(%v): %v", tt.args, err)
			}
			if got != tt.want {
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
		{[]string{"-n", "x"}, "invalid number of lines: 'x'"},
		{[]string{"-n", ""}, "invalid number of lines: ''"},
		{[]string{"-n", "-"}, "invalid number of lines: '-'"},
		{[]string{"-n", "1.5"}, "invalid number of lines: '1.5'"},
		{[]string{"-n", "--3"}, "invalid number of lines: '--3'"},
		{[]string{"-n", "3x"}, "invalid number of lines: '3x'"},
		{[]string{"-c", "x"}, "invalid number of bytes: 'x'"},
		{[]string{"--bytes=1KiBs"}, "invalid number of bytes: '1KiBs'"},
		{[]string{"-c", "-K"}, "invalid number of bytes: '-K'"},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			_, err := options(t, tt.args...)
			var usage *core.UsageError
			if !errors.As(err, &usage) {
				t.Fatalf("err = %v, want a UsageError", err)
			}
			if usage.Msg != tt.want {
				t.Fatalf("message = %q, want %q", usage.Msg, tt.want)
			}
		})
	}
}

// TestCommand runs the tool the way the binary does, through its cobra
// command, which is what has to accept the obsolete -NUM form.
func TestCommand(t *testing.T) {
	const input = "1\n2\n3\n4\n5\n"
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"default", nil, input},
		{"obsolete form", []string{"-2"}, "1\n2\n"},
		{"obsolete form with an operand", []string{"-3", "-"}, "1\n2\n3\n"},
		{"all but last", []string{"-n", "-2"}, "1\n2\n3\n"},
		{"all but last, long", []string{"--lines=-4"}, "1\n"},
		{"bytes", []string{"-c", "3"}, "1\n2"},
		{"verbose", []string{"-v", "-n1"}, "==> standard input <==\n1\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			c := Spec().Command(core.Streams{In: strings.NewReader(input), Out: &out, Err: &errb})
			c.SetArgs(tt.args)
			if err := c.Execute(); err != nil {
				t.Fatalf("Execute: %v (stderr %q)", err, errb.String())
			}
			if out.String() != tt.want {
				t.Fatalf("stdout = %q, want %q", out.String(), tt.want)
			}
		})
	}

	t.Run("digit shorthands stay out of the help", func(t *testing.T) {
		var out bytes.Buffer
		c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &out})
		c.SetArgs([]string{"--help"})
		if err := c.Execute(); err != nil {
			t.Fatal(err)
		}
		if help := out.String(); strings.Contains(help, "obsolete") || !strings.Contains(help, "--lines [-]NUM") {
			t.Fatalf("unexpected help:\n%s", help)
		}
	})
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != head.Name || !strings.HasPrefix(m.Use, head.Name+" ") {
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
