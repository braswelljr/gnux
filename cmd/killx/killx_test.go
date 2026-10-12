package killx

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/kill"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		opts     kill.Options
		operands []string
	}{
		{"default", []string{"1"}, kill.Options{}, []string{"1"}},
		{"-9", []string{"-9", "1"}, kill.Options{Signal: "9"}, []string{"1"}},
		{"-KILL", []string{"-KILL", "1", "2"}, kill.Options{Signal: "KILL"}, []string{"1", "2"}},
		{"-s", []string{"-s", "HUP", "1"}, kill.Options{Signal: "HUP"}, []string{"1"}},
		{"-sHUP", []string{"-sHUP", "1"}, kill.Options{Signal: "HUP"}, []string{"1"}},
		{"-n", []string{"-n", "15", "1"}, kill.Options{Signal: "15"}, []string{"1"}},
		{"--signal=", []string{"--signal=INT", "1"}, kill.Options{Signal: "INT"}, []string{"1"}},
		{"--signal", []string{"--signal", "INT", "1"}, kill.Options{Signal: "INT"}, []string{"1"}},
		{"signal after the pid", []string{"1", "-9"}, kill.Options{Signal: "9"}, []string{"1"}},
		{"negative pid after --", []string{"-TERM", "--", "-1234"}, kill.Options{Signal: "TERM"}, []string{"-1234"}},
		{"-l", []string{"-l"}, kill.Options{Mode: kill.ModeList}, nil},
		{"-l operands", []string{"-l", "9", "TERM"}, kill.Options{Mode: kill.ModeList}, []string{"9", "TERM"}},
		{"-L", []string{"-L"}, kill.Options{Mode: kill.ModeTable}, nil},
		{"-t", []string{"-t"}, kill.Options{Mode: kill.ModeTable}, nil},
		{"--table", []string{"--table"}, kill.Options{Mode: kill.ModeTable}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseArgs(tt.args)
			if err != nil {
				t.Fatal(err)
			}
			if p.opts != tt.opts || !reflect.DeepEqual(p.operands, tt.operands) {
				t.Fatalf("parseArgs(%q) = %+v %q, want %+v %q", tt.args, p.opts, p.operands, tt.opts, tt.operands)
			}
		})
	}
}

func TestParseArgsInvalid(t *testing.T) {
	tests := []struct {
		args []string
		msg  string
	}{
		{[]string{"-9", "-15", "1"}, "multiple signals specified"},
		{[]string{"-l", "-9"}, "cannot combine signal with -l or -t"},
		{[]string{"-l", "-L"}, "multiple -l or -t options specified"},
		{[]string{"-s"}, "option requires an argument -- 's'"},
		{[]string{"--bogus"}, "unrecognized option '--bogus'"},
	}
	for _, tt := range tests {
		_, err := parseArgs(tt.args)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != tt.msg {
			t.Errorf("parseArgs(%q) = %v, want %q", tt.args, err, tt.msg)
		}
	}
	if p, _ := parseArgs([]string{"--help"}); !p.help {
		t.Fatal("--help not seen")
	}
}

func TestCommand(t *testing.T) {
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs([]string{"-l", "137", "KILL"})
	if err := c.Execute(); err != nil || out.String() != "KILL\n9\n" {
		t.Fatalf("-l: %v, stdout = %q (stderr %q)", err, out.String(), errb.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != kill.Name || !strings.HasPrefix(m.Use, kill.Name+" ") {
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
