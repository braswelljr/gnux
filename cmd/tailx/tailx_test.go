package tailx

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tail"
)

// options registers every flag, parses args and returns the resolved Options.
func options(t *testing.T, args ...string) (tail.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(tail.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return buildOptions(fs)
}

func TestBuildOptions(t *testing.T) {
	// base is what no flags at all resolve to.
	base := tail.Options{Count: 10, SleepInterval: time.Second}
	with := func(edit func(*tail.Options)) tail.Options {
		o := base
		edit(&o)
		return o
	}
	lines := func(n int64) tail.Options { return with(func(o *tail.Options) { o.Count = n }) }
	bytesOpt := func(n int64) tail.Options { return with(func(o *tail.Options) { o.Count, o.Bytes = n, true }) }
	fromLine := func(n int64) tail.Options { return with(func(o *tail.Options) { o.Count, o.FromStart = n, true }) }
	fromByte := func(n int64) tail.Options {
		return with(func(o *tail.Options) { o.Count, o.Bytes, o.FromStart = n, true, true })
	}
	following := func(m tail.FollowMode, retry bool) tail.Options {
		return with(func(o *tail.Options) { o.Follow, o.Retry = m, retry })
	}

	tests := []struct {
		name string
		args []string
		want tail.Options
	}{
		{"defaults", nil, base},

		{"lines", []string{"-n", "3"}, lines(3)},
		{"lines attached", []string{"-n3"}, lines(3)},
		{"lines long", []string{"--lines=3"}, lines(3)},
		{"lines zero", []string{"-n", "0"}, lines(0)},
		{"lines explicit minus", []string{"-n", "-3"}, lines(3)},
		{"from line", []string{"-n", "+3"}, fromLine(3)},
		{"from line, attached", []string{"-n+3"}, fromLine(3)},
		{"from line, long", []string{"--lines=+3"}, fromLine(3)},

		{"bytes", []string{"-c", "5"}, bytesOpt(5)},
		{"bytes long", []string{"--bytes=5"}, bytesOpt(5)},
		{"from byte", []string{"-c", "+5"}, fromByte(5)},

		{"suffix b", []string{"-c", "2b"}, bytesOpt(1024)},
		{"suffix K", []string{"-c", "1K"}, bytesOpt(1024)},
		{"suffix KiB", []string{"-c", "1KiB"}, bytesOpt(1024)},
		{"suffix kB", []string{"-c", "1kB"}, bytesOpt(1000)},
		{"suffix M", []string{"-c", "2M"}, bytesOpt(2 << 20)},
		{"suffix MB", []string{"-c", "2MB"}, bytesOpt(2_000_000)},
		{"suffix G", []string{"-c", "1G"}, bytesOpt(1 << 30)},
		{"suffix on lines", []string{"-n", "1K"}, lines(1024)},
		{"suffix with from-start", []string{"-c", "+1K"}, fromByte(1024)},
		{"overflow is clamped", []string{"-c", "1Y"}, bytesOpt(math.MaxInt64)},

		{"quiet", []string{"-q"}, with(func(o *tail.Options) { o.Quiet = true })},
		{"quiet long", []string{"--quiet"}, with(func(o *tail.Options) { o.Quiet = true })},
		{"silent", []string{"--silent"}, with(func(o *tail.Options) { o.Quiet = true })},
		{"verbose", []string{"-v"}, with(func(o *tail.Options) { o.Verbose = true })},
		{"verbose long", []string{"--verbose"}, with(func(o *tail.Options) { o.Verbose = true })},
		{"zero terminated", []string{"-z"}, with(func(o *tail.Options) { o.ZeroTerminated = true })},
		{"zero terminated long", []string{"--zero-terminated"}, with(func(o *tail.Options) { o.ZeroTerminated = true })},

		{"follow", []string{"-f"}, following(tail.FollowDescriptor, false)},
		{"follow does not swallow an operand", []string{"-f", "name"}, following(tail.FollowDescriptor, false)},
		{"follow long", []string{"--follow"}, following(tail.FollowDescriptor, false)},
		{"follow=descriptor", []string{"--follow=descriptor"}, following(tail.FollowDescriptor, false)},
		{"follow=name", []string{"--follow=name"}, following(tail.FollowName, false)},
		{"follow value may be abbreviated", []string{"--follow=n"}, following(tail.FollowName, false)},
		{"retry", []string{"--retry"}, with(func(o *tail.Options) { o.Retry = true })},
		{"follow with retry", []string{"-f", "--retry"}, following(tail.FollowDescriptor, true)},
		{"-F", []string{"-F"}, following(tail.FollowName, true)},
		{"-F beats -f", []string{"-f", "-F"}, following(tail.FollowName, true)},

		{"sleep interval", []string{"-s", "5"}, with(func(o *tail.Options) { o.SleepInterval = 5 * time.Second })},
		{"sleep interval fraction", []string{"--sleep-interval=0.25"}, with(func(o *tail.Options) { o.SleepInterval = 250 * time.Millisecond })},
		{"sleep interval zero is bounded", []string{"-s", "0"}, with(func(o *tail.Options) { o.SleepInterval = time.Millisecond })},
		{"pid", []string{"--pid=42"}, with(func(o *tail.Options) { o.PID = 42 })},
		{"combined", []string{"-qfn", "2"}, with(func(o *tail.Options) { o.Count, o.Quiet, o.Follow = 2, true, tail.FollowDescriptor })},

		// The last of -n, -c and -NUM wins, as in GNU tail.
		{"bytes after lines", []string{"-n", "3", "-c", "5"}, bytesOpt(5)},
		{"lines after bytes", []string{"-c", "5", "-n", "+3"}, fromLine(3)},

		{"obsolete -NUM", []string{"-5"}, lines(5)},
		{"obsolete multi-digit", []string{"-120"}, lines(120)},
		{"obsolete with follow", []string{"-5f"}, with(func(o *tail.Options) { o.Count, o.Follow = 5, tail.FollowDescriptor })},
		{"obsolete then -n", []string{"-5", "-n", "3"}, lines(3)},
		{"-c then obsolete", []string{"-c", "3", "-7"}, lines(7)},
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
		{[]string{"-n", "+"}, "invalid number of lines: '+'"},
		{[]string{"-n", "++3"}, "invalid number of lines: '++3'"},
		{[]string{"-n", "3x"}, "invalid number of lines: '3x'"},
		{[]string{"-c", "x"}, "invalid number of bytes: 'x'"},
		{[]string{"--bytes=+1KiBs"}, "invalid number of bytes: '+1KiBs'"},
		{[]string{"--follow=x"}, "invalid argument 'x' for '--follow'"},
		{[]string{"--follow="}, "invalid argument '' for '--follow'"},
		{[]string{"-s", "-1"}, "invalid number of seconds: '-1'"},
		{[]string{"--pid=-4"}, "invalid PID: '-4'"},
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
		{"obsolete form", []string{"-2"}, "4\n5\n"},
		{"obsolete form with an operand", []string{"-3", "-"}, "3\n4\n5\n"},
		{"from line", []string{"-n", "+4"}, "4\n5\n"},
		{"from line, long", []string{"--lines=+5"}, "5\n"},
		{"bytes", []string{"-c", "3"}, "\n5\n"},
		{"verbose", []string{"-v", "-n1"}, "==> standard input <==\n5\n"},
		{"max unchanged stats", []string{"--max-unchanged-stats=3", "-n1"}, "5\n"},
		{"max unchanged stats saturates", []string{"--max-unchanged-stats", "+99999999999999999999", "-n1"}, "5\n"},
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
		if help := out.String(); strings.Contains(help, "obsolete") || !strings.Contains(help, "--lines [+]NUM") {
			t.Fatalf("unexpected help:\n%s", help)
		}
	})
}

func TestMaxUnchangedStatsInvalid(t *testing.T) {
	for _, n := range []string{"x", "-1", "", "3k"} {
		var out, errb bytes.Buffer
		c := Spec().Command(core.Streams{In: strings.NewReader("a\n"), Out: &out, Err: &errb})
		c.SetArgs([]string{"--max-unchanged-stats=" + n})
		c.SilenceErrors = true
		if err := c.Execute(); !errors.Is(err, core.ErrFailed) {
			t.Fatalf("%q: err = %v, want ErrFailed", n, err)
		}
		if want := "tailx: invalid maximum number of unchanged stats between opens: '" + n + "'\n"; errb.String() != want || out.Len() != 0 {
			t.Fatalf("%q: stderr = %q, stdout = %q", n, errb.String(), out.String())
		}
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != tail.Name || !strings.HasPrefix(m.Use, tail.Name+" ") {
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
