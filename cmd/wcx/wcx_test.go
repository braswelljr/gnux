package wcx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/wc"
)

// options registers every flag, parses args and returns the resolved Options.
func options(t *testing.T, args ...string) (wc.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(wc.Name, pflag.ContinueOnError)
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
		want wc.Options
	}{
		// The lines/words/bytes default is the engine's, not the adapter's.
		{"defaults", nil, wc.Options{}},
		{"lines", []string{"-l"}, wc.Options{Lines: true}},
		{"lines long", []string{"--lines"}, wc.Options{Lines: true}},
		{"words", []string{"-w"}, wc.Options{Words: true}},
		{"words long", []string{"--words"}, wc.Options{Words: true}},
		{"chars", []string{"-m"}, wc.Options{Chars: true}},
		{"chars long", []string{"--chars"}, wc.Options{Chars: true}},
		{"bytes", []string{"-c"}, wc.Options{Bytes: true}},
		{"bytes long", []string{"--bytes"}, wc.Options{Bytes: true}},
		{"max line length", []string{"-L"}, wc.Options{MaxLineLength: true}},
		{"max line length long", []string{"--max-line-length"}, wc.Options{MaxLineLength: true}},
		{"combined", []string{"-lwmcL"}, wc.Options{Lines: true, Words: true, Chars: true, Bytes: true, MaxLineLength: true}},
		{"files0-from", []string{"--files0-from=list"}, wc.Options{Files0From: "list"}},
		{"files0-from stdin", []string{"--files0-from", "-"}, wc.Options{Files0From: "-"}},
		{"total=auto", []string{"--total=auto"}, wc.Options{Total: wc.TotalAuto}},
		{"total=always", []string{"--total=always"}, wc.Options{Total: wc.TotalAlways}},
		{"total=only", []string{"--total=only"}, wc.Options{Total: wc.TotalOnly}},
		{"total=never", []string{"--total", "never"}, wc.Options{Total: wc.TotalNever}},
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
	for _, when := range []string{"sometimes", "", "ALWAYS"} {
		_, err := options(t, "--total="+when)
		var usage *core.UsageError
		if !errors.As(err, &usage) {
			t.Fatalf("--total=%s: err = %v, want a UsageError", when, err)
		}
		if want := "invalid argument '" + when + "' for '--total'"; usage.Msg != want {
			t.Fatalf("message = %q, want %q", usage.Msg, want)
		}
	}
}

// TestCommand runs the tool the way the binary does, through its cobra command.
func TestCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"default", nil, "      2       3      14\n"},
		{"lines", []string{"-l"}, "2\n"},
		{"order is fixed", []string{"-c", "-l"}, "      2      14\n"},
		{"total only", []string{"--total=only", "-w", "-", "-"}, "3\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			c := Spec().Command(core.Streams{In: strings.NewReader("one two\nthree\n"), Out: &out, Err: &errb})
			c.SetArgs(tt.args)
			if err := c.Execute(); err != nil {
				t.Fatalf("Execute: %v (stderr %q)", err, errb.String())
			}
			if out.String() != tt.want {
				t.Fatalf("stdout = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != wc.Name || !strings.HasPrefix(m.Use, wc.Name+" ") {
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
