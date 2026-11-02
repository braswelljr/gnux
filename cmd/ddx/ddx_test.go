package ddx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/dd"
	"github.com/braswelljr/gnux/internal/tool"
)

func TestViewArgs(t *testing.T) {
	view, operands := viewArgs([]string{"if=a", "--pretty", "--theme", "nord", "status=progress", "--theme=dracula"})
	if strings.Join(view, " ") != "--pretty --theme nord --theme=dracula" || strings.Join(operands, " ") != "if=a status=progress" {
		t.Fatalf("view %q, operands %q", view, operands)
	}
}

func TestBuildOptions(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	cases := []struct {
		args     []string
		terminal bool
		env      map[string]string
		want     dd.Options
	}{
		// Standard error redirected: GNU's progress line.
		{nil, false, nil, dd.Options{}},
		{nil, true, nil, dd.Options{Pretty: true, Color: true}},
		{nil, true, map[string]string{"NO_COLOR": "1"}, dd.Options{Pretty: true}},
		{[]string{"--pretty=never"}, true, nil, dd.Options{}},
		{nil, true, map[string]string{"GNUX_PRETTY": "never"}, dd.Options{}},
		{[]string{"--theme=nord"}, true, nil, dd.Options{Pretty: true, Color: true, Theme: "nord"}},
	}
	for _, c := range cases {
		fs := pflag.NewFlagSet(dd.Name, pflag.ContinueOnError)
		tool.RegisterView(fs)
		if err := tool.ParseFlagSet(fs, c.args, false); err != nil {
			t.Fatal(err)
		}
		got, err := buildOptions(fs, c.terminal, env(c.env), &bytes.Buffer{})
		if err != nil || got != c.want {
			t.Errorf("%q terminal=%v env=%v: got %+v (%v), want %+v", c.args, c.terminal, c.env, got, err, c.want)
		}
	}
}
