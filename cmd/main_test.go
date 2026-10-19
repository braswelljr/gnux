package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/braswelljr/gnux/core"
)

// run executes gnux with argv and returns the exit status and both streams.
func run(t *testing.T, argv ...string) (status int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	status = Main(argv, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	return status, out.String(), errb.String()
}

func TestProgName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"rmx", "rmx"},
		{"/usr/local/bin/rmx", "rmx"},
		{`C:\tools\rmx.exe`, "rmx"},
		{"./bin/gnux", "gnux"},
		{"RMX.EXE", "rmx"},
	}
	for _, tt := range tests {
		if got := progName(tt.in); got != tt.want {
			t.Errorf("progName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRegistryIsConsistent(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range Tools() {
		name := s.Name()
		if name == "" || !strings.HasSuffix(name, "x") {
			t.Errorf("tool name %q must be the GNU name with an x suffix", name)
		}
		if s.GNU+"x" != name {
			t.Errorf("tool %q mirrors GNU %q; want name %q", name, s.GNU, s.GNU+"x")
		}
		if !strings.HasPrefix(s.Meta.Use, name) {
			t.Errorf("tool %q: Use %q must start with the tool name", name, s.Meta.Use)
		}
		if s.Meta.Short == "" || s.Meta.Long == "" || len(s.Meta.Examples) == 0 {
			t.Errorf("tool %q: Short, Long and at least one example are required", name)
		}
		if s.Run == nil {
			t.Errorf("tool %q has no Run", name)
		}
		for _, n := range []string{name, s.GNU} {
			if seen[n] {
				t.Errorf("name %q is registered twice", n)
			}
			seen[n] = true
		}
	}
}

func TestEveryToolAnswersHelpAndVersion(t *testing.T) {
	for _, s := range Tools() {
		name := s.Name()
		t.Run(name, func(t *testing.T) {
			status, out, errb := run(t, name, "--version")
			if status != 0 || !strings.Contains(out, name) {
				t.Errorf("--version: status=%d stdout=%q stderr=%q", status, out, errb)
			}
			status, out, errb = run(t, name, "--help")
			if status != 0 || !strings.Contains(out, name) {
				t.Errorf("--help: status=%d stdout=%q stderr=%q", status, out, errb)
			}
		})
	}
}

func TestDispatch(t *testing.T) {
	newFile := func(t *testing.T) string {
		t.Helper()
		f := filepath.Join(t.TempDir(), "f")
		if err := os.WriteFile(f, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		return f
	}
	gone := func(t *testing.T, f string) {
		t.Helper()
		if _, err := os.Lstat(f); !os.IsNotExist(err) {
			t.Fatalf("%s should have been removed (err=%v)", f, err)
		}
	}

	t.Run("subcommand", func(t *testing.T) {
		f := newFile(t)
		if status, _, errb := run(t, "gnux", "rmx", f); status != 0 {
			t.Fatalf("status = %d, stderr = %q", status, errb)
		}
		gone(t, f)
	})

	t.Run("subcommand by GNU name", func(t *testing.T) {
		f := newFile(t)
		if status, _, errb := run(t, "gnux", "rm", f); status != 0 {
			t.Fatalf("status = %d, stderr = %q", status, errb)
		}
		gone(t, f)
	})

	t.Run("argv0", func(t *testing.T) {
		f := newFile(t)
		if status, _, errb := run(t, "/usr/local/bin/rmx", f); status != 0 {
			t.Fatalf("status = %d, stderr = %q", status, errb)
		}
		gone(t, f)
	})

	t.Run("argv0 by GNU name", func(t *testing.T) {
		f := newFile(t)
		if status, _, errb := run(t, "rm", f); status != 0 {
			t.Fatalf("status = %d, stderr = %q", status, errb)
		}
		gone(t, f)
	})
}

// TestBracket checks that [ is test with a closing ], by every route: as a
// subcommand, as the program name, and from the shell, but not as a tool of
// its own.
func TestBracket(t *testing.T) {
	cases := []struct {
		argv       []string
		status     int
		wantStderr string
	}{
		{[]string{"gnux", "[", "1", "-eq", "1", "]"}, 0, ""},
		{[]string{"gnux", "[", "a", "=", "b", "]"}, 1, ""},
		{[]string{"/usr/local/bin/[", "-n", "x", "]"}, 0, ""},
		{[]string{"gnux", "[", "1", "-eq", "1"}, 2, "[: missing ']'\n"},
		{[]string{"[", "-l", "abc", "-eq", "3", "]"}, 0, ""},
	}
	for _, c := range cases {
		status, _, errb := run(t, c.argv...)
		if status != c.status || errb != c.wantStderr {
			t.Errorf("%q: status = %d, stderr = %q; want %d, %q", c.argv, status, errb, c.status, c.wantStderr)
		}
	}
	if status, out, _ := run(t, "gnux", "[", "--help"); status != 0 || !strings.HasPrefix(out, "Usage: [ EXPRESSION ]") {
		t.Errorf("[ --help: status = %d, stdout = %q", status, out)
	}
	for _, s := range Tools() {
		if s.Name() == "[" || s.GNU == "[" {
			t.Errorf("[ is registered as a tool: %+v", s.Meta.Name)
		}
	}
}

// A tool named by the first argument runs without the root command being
// built, so it must answer exactly as it does under its own name.
func TestSubcommandMatchesArgv0(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"--version"}, {}, {"--no-such-flag"}, {"--", "-missing"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			wantStatus, wantOut, wantErr := run(t, append([]string{"rmx"}, args...)...)
			for _, name := range []string{"rmx", "rm"} {
				status, out, errb := run(t, append([]string{"gnux", name}, args...)...)
				if status != wantStatus || out != wantOut || errb != wantErr {
					t.Errorf("gnux %s %q = (%d, %q, %q), want (%d, %q, %q)",
						name, args, status, out, errb, wantStatus, wantOut, wantErr)
				}
			}
		})
	}
}

// BenchmarkDispatch measures what gnux spends before a tool starts, for each
// way of naming one. None of them builds the root command, so they cost about
// the same.
func BenchmarkDispatch(b *testing.B) {
	for _, argv := range [][]string{{"truex"}, {"gnux", "truex"}, {"gnux", "true"}} {
		b.Run(strings.Join(argv, " "), func(b *testing.B) {
			s := core.Discard()
			b.ReportAllocs()
			for b.Loop() {
				Main(argv, s)
			}
		})
	}
}

// A tool that panics is reported as a bug and gives a status; it does not take
// its host (a shell pipeline, the playground, an embedding program) down.
func TestPanicIsContained(t *testing.T) {
	boom := &cobra.Command{
		Use:           "boomx",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          func(*cobra.Command, []string) error { panic("kaboom") },
	}
	var errb bytes.Buffer
	status := runCommand(boom, nil, core.Streams{In: strings.NewReader(""), Out: io.Discard, Err: &errb})
	if status != ExitInternal {
		t.Errorf("status = %d, want %d", status, ExitInternal)
	}
	for _, want := range []string{"boomx: internal error: kaboom\n", "goroutine ", "boomx: this is a bug in gnux; please report it\n"} {
		if !strings.Contains(errb.String(), want) {
			t.Errorf("stderr lacks %q:\n%s", want, errb.String())
		}
	}
}

// A panic an engine brought back from one of its own goroutines is reported
// with the stack of where it happened.
func TestCarriedPanicShowsItsOwnStack(t *testing.T) {
	boom := &cobra.Command{
		Use:           "boomx",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(*cobra.Command, []string) error {
			panic(&core.Panic{Value: "worker kaboom", Stack: []byte("goroutine 7 [running]:\nthe worker's stack\n")})
		},
	}
	var errb bytes.Buffer
	status := runCommand(boom, nil, core.Streams{In: strings.NewReader(""), Out: io.Discard, Err: &errb})
	want := "boomx: internal error: worker kaboom\ngoroutine 7 [running]:\nthe worker's stack\n" +
		"boomx: this is a bug in gnux; please report it\n"
	if status != ExitInternal || errb.String() != want {
		t.Errorf("status = %d, stderr = %q, want %d, %q", status, errb.String(), ExitInternal, want)
	}
}

func TestDiagnostics(t *testing.T) {
	t.Run("engine failure exits 1 without a usage hint", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "nope")
		status, _, errb := run(t, "rmx", missing)
		if status != 1 {
			t.Fatalf("status = %d, want 1", status)
		}
		if !strings.HasPrefix(errb, "rmx: cannot remove") || strings.Contains(errb, "--help") {
			t.Fatalf("stderr = %q", errb)
		}
	})

	t.Run("usage error names the tool", func(t *testing.T) {
		for _, argv := range [][]string{{"rmx"}, {"gnux", "rmx"}, {"rmx", "--no-such-flag"}} {
			status, _, errb := run(t, argv...)
			if status != 1 {
				t.Errorf("%v: status = %d, want 1", argv, status)
			}
			if !strings.HasPrefix(errb, "rmx: ") || !strings.Contains(errb, "Try 'rmx --help' for more information.") {
				t.Errorf("%v: stderr = %q", argv, errb)
			}
		}
	})

	t.Run("unknown tool", func(t *testing.T) {
		status, _, errb := run(t, "gnux", "nosuchtool")
		if status != 1 || !strings.Contains(errb, "nosuchtool") {
			t.Fatalf("status = %d, stderr = %q", status, errb)
		}
	})
}

func TestRootListAndVersion(t *testing.T) {
	status, out, _ := run(t, "gnux", "--list")
	if status != 0 {
		t.Fatalf("--list status = %d", status)
	}
	if got, want := len(strings.Fields(out)), len(Tools()); got != want {
		t.Fatalf("--list printed %d names, want %d", got, want)
	}

	status, out, _ = run(t, "gnux", "--version")
	if status != 0 || !strings.HasPrefix(out, "gnux ") {
		t.Fatalf("--version: status=%d out=%q", status, out)
	}
}
