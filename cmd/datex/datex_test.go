package datex

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/date"
)

func TestRewrite(t *testing.T) {
	tests := []struct {
		in, want []string
	}{
		{[]string{"-I"}, []string{"--iso-8601"}},
		{[]string{"-Iseconds"}, []string{"--iso-8601=seconds"}},
		{[]string{"-uIns"}, []string{"-u", "--iso-8601=ns"}},
		{[]string{"-d", "-Ix"}, []string{"-d", "-Ix"}},
		{[]string{"-dIx"}, []string{"-dIx"}},
		{[]string{"--date", "-Ix"}, []string{"--date", "-Ix"}},
		{[]string{"--", "-Ix"}, []string{"--", "-Ix"}},
		{[]string{"+%F", "-u"}, []string{"+%F", "-u"}},
	}
	for _, tt := range tests {
		if got := rewrite(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("rewrite(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func parse(t *testing.T, args ...string) (parsed, error) {
	t.Helper()
	fs := pflag.NewFlagSet(date.Name, pflag.ContinueOnError)
	fs.Bool("help", false, "")
	fs.Bool("version", false, "")
	Register(fs)
	return parseArgs(fs, args)
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		want     date.Options
		operands []string
	}{
		{"defaults", nil, date.Options{}, nil},
		{"-d", []string{"-d", "now"}, date.Options{Date: "now", HasDate: true}, nil},
		{"-d empty", []string{"--date="}, date.Options{HasDate: true}, nil},
		{"-f", []string{"-f", "dates"}, date.Options{File: "dates"}, nil},
		{"-r", []string{"-r", "file"}, date.Options{Reference: "file"}, nil},
		{"-s", []string{"-s", "x"}, date.Options{Set: "x", HasSet: true}, nil},
		{"-u", []string{"-u"}, date.Options{UTC: true}, nil},
		{"--universal", []string{"--universal"}, date.Options{UTC: true}, nil},
		{"-I", []string{"-I"}, date.Options{ISO8601: "date"}, nil},
		{"-Iseconds", []string{"-Iseconds"}, date.Options{ISO8601: "seconds"}, nil},
		{"-I abbreviation", []string{"-Isec"}, date.Options{ISO8601: "seconds"}, nil},
		{"--iso-8601=ns", []string{"--iso-8601=ns"}, date.Options{ISO8601: "ns"}, nil},
		{"-I unknown left to the engine", []string{"-Ifoo"}, date.Options{ISO8601: "foo"}, nil},
		{"--rfc-3339", []string{"--rfc-3339=s"}, date.Options{RFC3339: "seconds"}, nil},
		{"-R", []string{"-R"}, date.Options{RFC5322: true}, nil},
		{"format", []string{"-u", "+%F"}, date.Options{UTC: true}, []string{"+%F"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parse(t, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			same := reflect.DeepEqual(p.operands, tt.operands) || len(p.operands)+len(tt.operands) == 0
			if p.opts != tt.want || !same {
				t.Fatalf("parseArgs(%q) = %+v %q, want %+v %q", tt.args, p.opts, p.operands, tt.want, tt.operands)
			}
		})
	}
	var usage *core.UsageError
	if _, err := parse(t, "-Z"); !errors.As(err, &usage) {
		t.Fatalf("unknown option: %v", err)
	}
}

func TestCommand(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"-u", "-d", "@0", "+%s"}, "0\n"},
		{[]string{"-uIseconds", "-d", "@0"}, "1970-01-01T00:00:00+00:00\n"},
		{[]string{"-u", "--date=@86400", "-I"}, "1970-01-02\n"},
		{[]string{"-u", "-R", "-d", "@0"}, "Thu, 01 Jan 1970 00:00:00 +0000\n"},
	}
	for _, tt := range tests {
		var out, errb bytes.Buffer
		c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
		c.SetArgs(tt.args)
		if err := c.Execute(); err != nil || out.String() != tt.want {
			t.Errorf("%q: %v, stdout = %q (stderr %q), want %q", tt.args, err, out.String(), errb.String(), tt.want)
		}
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != date.Name || !strings.HasPrefix(m.Use, date.Name+" ") {
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
