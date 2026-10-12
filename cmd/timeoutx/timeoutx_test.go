package timeoutx

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/timeout"
)

func options(t *testing.T, args ...string) (timeout.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(timeout.Name, pflag.ContinueOnError)
	Register(fs)
	fs.SetInterspersed(false)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return buildOptions(fs)
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want timeout.Options
	}{
		{"defaults", nil, timeout.Options{}},
		{"-s", []string{"-s", "INT"}, timeout.Options{Signal: "INT"}},
		{"-k", []string{"-k", "10s"}, timeout.Options{KillAfter: 10 * time.Second}},
		{"-k minutes", []string{"--kill-after=1.5m"}, timeout.Options{KillAfter: 90 * time.Second}},
		{"flags", []string{"--preserve-status", "--foreground", "-v"}, timeout.Options{PreserveStatus: true, Foreground: true, Verbose: true}},
		{"options stop at DURATION", []string{"5", "cmd", "-v"}, timeout.Options{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil || got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, %v; want %+v", tt.args, got, err, tt.want)
			}
		})
	}
	_, err := options(t, "-k", "soon")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "invalid time interval 'soon'" {
		t.Fatalf("-k soon: %v", err)
	}
}

func TestCommand(t *testing.T) {
	var errb bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &errb})
	c.SetArgs([]string{"5", "gnux-no-such-command", "-v"})
	err := c.Execute()
	var exit *core.ExitError
	if !errors.As(err, &exit) || exit.Code != timeout.ExitNotFound {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != timeout.Name || !strings.HasPrefix(m.Use, timeout.Name+" ") {
		t.Fatalf("Name = %q, Use = %q", m.Name, m.Use)
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
