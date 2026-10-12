package nicex

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/nice"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		opts    nice.Options
		command []string
	}{
		{"no args", nil, nice.Options{}, nil},
		{"command", []string{"make", "-j8"}, nice.Options{}, []string{"make", "-j8"}},
		{"-n", []string{"-n", "5", "make"}, nice.Options{Adjustment: 5, Adjust: true}, []string{"make"}},
		{"-n attached", []string{"-n-5", "make"}, nice.Options{Adjustment: -5, Adjust: true}, []string{"make"}},
		{"-n negative", []string{"-n", "-5", "make"}, nice.Options{Adjustment: -5, Adjust: true}, []string{"make"}},
		{"--adjustment=", []string{"--adjustment=7", "make"}, nice.Options{Adjustment: 7, Adjust: true}, []string{"make"}},
		{"--adjustment", []string{"--adjustment", "7", "make"}, nice.Options{Adjustment: 7, Adjust: true}, []string{"make"}},
		{"obsolete", []string{"-5", "make"}, nice.Options{Adjustment: 5, Adjust: true}, []string{"make"}},
		{"obsolete plus", []string{"-+5", "make"}, nice.Options{Adjustment: 5, Adjust: true}, []string{"make"}},
		{"obsolete negative", []string{"--5", "make"}, nice.Options{Adjustment: -5, Adjust: true}, []string{"make"}},
		{"-n 0", []string{"-n", "0", "make"}, nice.Options{Adjust: true}, []string{"make"}},
		{"last wins", []string{"-n", "3", "-7", "make"}, nice.Options{Adjustment: 7, Adjust: true}, []string{"make"}},
		{"huge clamps", []string{"-n", "99999999999", "make"}, nice.Options{Adjustment: 2*nice.MaxNiceness + 1, Adjust: true}, []string{"make"}},
		{"options stop at the command", []string{"make", "-n", "5"}, nice.Options{}, []string{"make", "-n", "5"}},
		{"--", []string{"--", "-x"}, nice.Options{}, []string{"-x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseArgs(tt.args)
			if err != nil {
				t.Fatal(err)
			}
			if p.opts != tt.opts || !reflect.DeepEqual(p.command, tt.command) {
				t.Fatalf("parseArgs(%q) = %+v %q, want %+v %q", tt.args, p.opts, p.command, tt.opts, tt.command)
			}
		})
	}
}

func TestParseArgsInvalid(t *testing.T) {
	var usage *core.UsageError
	for _, args := range [][]string{{"-x"}, {"--bogus"}, {"-n"}} {
		if _, err := parseArgs(args); !errors.As(err, &usage) {
			t.Errorf("%q: %v", args, err)
		}
	}
	_, err := parseArgs([]string{"-n", "x", "make"})
	var fatal *fatalError
	if !errors.As(err, &fatal) || fatal.msg != "invalid adjustment 'x'" {
		t.Fatalf("-n x: %v", err)
	}
}

func TestCommand(t *testing.T) {
	var out, errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.SetArgs([]string{"-n", "x", "true"})
	err := c.Execute()
	var exit *core.ExitError
	if !errors.As(err, &exit) || exit.Code != 125 || errb.String() != "nicex: invalid adjustment 'x'\n" {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != nice.Name || !strings.HasPrefix(m.Use, nice.Name+" ") {
		t.Fatalf("Name = %q, Use = %q", m.Name, m.Use)
	}
	if n := len(m.Examples); n < 10 {
		t.Fatalf("%d examples, want at least 10", n)
	}
}
