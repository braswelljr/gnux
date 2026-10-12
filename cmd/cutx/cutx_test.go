package cutx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/cut"
)

// options registers every flag, parses args and returns the resolved Options.
func options(t *testing.T, args ...string) (cut.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(cut.Name, pflag.ContinueOnError)
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
		want cut.Options
	}{
		{"defaults", nil, cut.Options{}},
		{"bytes", []string{"-b", "1-3"}, cut.Options{Mode: cut.ModeBytes, List: "1-3"}},
		{"bytes long", []string{"--bytes=2"}, cut.Options{Mode: cut.ModeBytes, List: "2"}},
		{"characters", []string{"-c1"}, cut.Options{Mode: cut.ModeChars, List: "1"}},
		{"characters long", []string{"--characters", "4-"}, cut.Options{Mode: cut.ModeChars, List: "4-"}},
		{"fields", []string{"-f", "1,3"}, cut.Options{Mode: cut.ModeFields, List: "1,3"}},
		{"fields long", []string{"--fields=-2"}, cut.Options{Mode: cut.ModeFields, List: "-2"}},
		{"empty list is passed on", []string{"-f", ""}, cut.Options{Mode: cut.ModeFields}},
		{"delimiter", []string{"-f1", "-d", ":"}, cut.Options{Mode: cut.ModeFields, List: "1", Delimiter: ":"}},
		{"delimiter long", []string{"-f1", "--delimiter=,"}, cut.Options{Mode: cut.ModeFields, List: "1", Delimiter: ","}},
		{"empty delimiter is NUL", []string{"-f1", "-d", ""}, cut.Options{Mode: cut.ModeFields, List: "1", Delimiter: "\x00"}},
		{"only delimited", []string{"-f1", "-s"}, cut.Options{Mode: cut.ModeFields, List: "1", OnlyDelimited: true}},
		{"only delimited long", []string{"-f1", "--only-delimited"}, cut.Options{Mode: cut.ModeFields, List: "1", OnlyDelimited: true}},
		{"complement", []string{"-f1", "--complement"}, cut.Options{Mode: cut.ModeFields, List: "1", Complement: true}},
		{"output delimiter", []string{"-f1", "--output-delimiter=|"}, cut.Options{Mode: cut.ModeFields, List: "1", OutputDelimiter: "|", OutputDelimiterSet: true}},
		{"empty output delimiter", []string{"-f1", "--output-delimiter="}, cut.Options{Mode: cut.ModeFields, List: "1", OutputDelimiterSet: true}},
		{"zero terminated", []string{"-b1", "-z"}, cut.Options{Mode: cut.ModeBytes, List: "1", ZeroTerminated: true}},
		{"zero terminated long", []string{"-b1", "--zero-terminated"}, cut.Options{Mode: cut.ModeBytes, List: "1", ZeroTerminated: true}},
		{"n is ignored", []string{"-n", "-b1"}, cut.Options{Mode: cut.ModeBytes, List: "1"}},
		{"n combined", []string{"-nb1"}, cut.Options{Mode: cut.ModeBytes, List: "1"}},
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
	for _, args := range [][]string{
		{"-b1", "-f1"},
		{"-f1", "-c1"},
		{"-b1", "-c1", "-f1"},
	} {
		_, err := options(t, args...)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != "only one type of list may be specified" {
			t.Fatalf("%v: err = %v", args, err)
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
		{"fields", []string{"-d", ",", "-f", "2"}, "b\ne\n"},
		{"complement", []string{"-d", ",", "-f", "2", "--complement"}, "a,c\nd,f\n"},
		{"bytes", []string{"-b", "1,3"}, "ab\nde\n"},
		{"output delimiter", []string{"-d,", "-f1,3", "--output-delimiter=:"}, "a:c\nd:f\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			c := Spec().Command(core.Streams{In: strings.NewReader("a,b,c\nd,e,f\n"), Out: &out, Err: &errb})
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
	if m.Name != cut.Name || !strings.HasPrefix(m.Use, cut.Name+" ") {
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
