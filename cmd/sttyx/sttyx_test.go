package sttyx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/stty"
)

// TestSettingsAreNotOptions: sttyx reads its own command line, so a setting
// such as -echo reaches the engine instead of being refused as an option.
func TestSettingsAreNotOptions(t *testing.T) {
	if !Spec().RawArgs {
		t.Fatal("sttyx must read its arguments itself")
	}
	var out, errb bytes.Buffer
	fs := pflag.NewFlagSet("sttyx", pflag.ContinueOnError)
	Register(fs)
	err := run(fs, []string{"-echo"}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	if err == nil || !strings.Contains(errb.String(), "standard input") {
		t.Fatalf("err=%v stderr=%q", err, errb.String())
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		args     []string
		opts     stty.Options
		settings []string
	}{
		{nil, stty.Options{}, nil},
		{[]string{"-a"}, stty.Options{All: true}, nil},
		{[]string{"--save"}, stty.Options{Save: true}, nil},
		{[]string{"--sa"}, stty.Options{Save: true}, nil},
		{[]string{"-F", "/dev/ttyS0", "9600"}, stty.Options{File: "/dev/ttyS0"}, []string{"9600"}},
		{[]string{"-F/dev/ttyS0"}, stty.Options{File: "/dev/ttyS0"}, nil},
		{[]string{"--file=/dev/x", "raw"}, stty.Options{File: "/dev/x"}, []string{"raw"}},
		{[]string{"-echo", "-icanon", "min", "1"}, stty.Options{}, []string{"-echo", "-icanon", "min", "1"}},
		{[]string{"-ag"}, stty.Options{All: true, Save: true}, nil},
		{[]string{"--", "-a"}, stty.Options{}, []string{"-a"}},
	}
	for _, tt := range tests {
		opts, settings, err := parseArgs(tt.args)
		if err != nil || opts != tt.opts || strings.Join(settings, " ") != strings.Join(tt.settings, " ") {
			t.Errorf("parseArgs(%v) = %+v %v %v", tt.args, opts, settings, err)
		}
	}
	if _, _, err := parseArgs([]string{"-F", "a", "-F", "b"}); err == nil || err.Error() != "only one device may be specified" {
		t.Errorf("two devices: %v", err)
	}
}

func TestViewArgs(t *testing.T) {
	fs := pflag.NewFlagSet("sttyx", pflag.ContinueOnError)
	Register(fs)
	s := core.Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}
	view, rest, err := viewArgs(fs, []string{"-a", "--pretty", "--theme", "nord", "--pr", "--", "--pretty"}, s)
	// Only the full names are gnux's; after "--" everything is a setting.
	if err != nil || !view.Pretty || view.Theme != "nord" || strings.Join(rest, " ") != "-a --pr -- --pretty" {
		t.Fatalf("view %+v, rest %q, err %v", view, rest, err)
	}
	fs = pflag.NewFlagSet("sttyx", pflag.ContinueOnError)
	Register(fs)
	if view, _, _ = viewArgs(fs, []string{"-a"}, s); view.Pretty {
		t.Fatal("the view is on off a terminal")
	}
}
