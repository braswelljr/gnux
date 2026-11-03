package tool

import (
	"bytes"
	"testing"

	"github.com/spf13/pflag"
)

func TestViewOf(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		env      map[string]string
		terminal bool
		want     View
		warning  string
	}{
		{"pipe", nil, nil, false, View{}, ""},
		{"terminal", nil, nil, true, View{Pretty: true, Color: true}, ""},
		{"dumb terminal", nil, map[string]string{"TERM": "dumb"}, true, View{Pretty: true}, ""},
		{"NO_COLOR", nil, map[string]string{"NO_COLOR": "1"}, true, View{Pretty: true}, ""},
		{"GNUX_PRETTY=never", nil, map[string]string{"GNUX_PRETTY": "never"}, true, View{}, ""},
		{"GNUX_PRETTY=always through a pipe", nil, map[string]string{"GNUX_PRETTY": "always"}, false, View{Pretty: true, Color: true}, ""},
		{"--pretty=auto beats GNUX_PRETTY", []string{"--pretty=auto"}, map[string]string{"GNUX_PRETTY": "always"}, false, View{}, ""},
		{"bare --pretty", []string{"--pretty"}, nil, false, View{Pretty: true, Color: true}, ""},
		{"theme", []string{"--theme=nord"}, nil, true, View{Pretty: true, Color: true, Theme: "nord"}, ""},
		{"invalid variable", nil, map[string]string{"GNUX_PRETTY": "x"}, true, View{Pretty: true, Color: true},
			"t: ignoring invalid value of environment variable GNUX_PRETTY: 'x'\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet("t", pflag.ContinueOnError)
			RegisterView(fs)
			if err := fs.Parse(tt.args); err != nil {
				t.Fatal(err)
			}
			var warn bytes.Buffer
			got, err := ViewOf(fs, tt.terminal, "t", func(k string) string { return tt.env[k] }, &warn)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
			if warn.String() != tt.warning {
				t.Errorf("warning %q, want %q", warn.String(), tt.warning)
			}
		})
	}

	fs := pflag.NewFlagSet("t", pflag.ContinueOnError)
	RegisterView(fs)
	_ = fs.Parse([]string{"--theme=neon"})
	if _, err := ViewOf(fs, true, "t", func(string) string { return "" }, &bytes.Buffer{}); err == nil {
		t.Error("an unknown theme should be a usage error")
	}
	_ = fs.Parse([]string{"--pretty=sometimes"})
	if _, err := ViewOf(fs, true, "t", func(string) string { return "" }, &bytes.Buffer{}); err == nil {
		t.Error("an invalid --pretty should be a usage error")
	}
}
