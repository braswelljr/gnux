package lsx

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/ls"
)

func TestBuildView(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want ls.Options
	}{
		{"tree", []string{"--tree"}, want(func(o *ls.Options) { o.Tree = true })},
		{"tree with level", []string{"--tree", "--level=2"}, want(func(o *ls.Options) { o.Tree, o.Level = true, 2 })},
		{"-R with level", []string{"-R", "--level", "3"}, want(func(o *ls.Options) { o.Recursive, o.Level = true, 3 })},
		{"bare icons is always", []string{"--icons"}, want(func(o *ls.Options) { o.Icons = ls.WhenAlways })},
		{"icons auto", []string{"--icons=auto"}, want(func(o *ls.Options) { o.Icons = ls.WhenAuto })},
		{"bare color scale is all", []string{"--color-scale"}, want(func(o *ls.Options) { o.ColorScale = ls.ScaleAll })},
		{"color scale age", []string{"--color-scale=age"}, want(func(o *ls.Options) { o.ColorScale = ls.ScaleAge })},
		{"theme", []string{"--theme=nord"}, want(func(o *ls.Options) { o.Theme = "nord" })},
		{"theme abbreviated", []string{"--theme=gru"}, want(func(o *ls.Options) { o.Theme = "gruvbox" })},
		{"header git octal", []string{"--header", "--git", "--octal-permissions"}, want(func(o *ls.Options) {
			o.Header, o.Git, o.OctalPermissions = true, true, true
		})},
		{"git ignore and total size", []string{"--git-ignore", "--total-size"}, want(func(o *ls.Options) {
			o.GitIgnore, o.TotalSize = true, true
		})},
		{"only dirs then files: last wins", []string{"--only-dirs", "--only-files"}, want(func(o *ls.Options) { o.Only = ls.OnlyFiles })},
		{"only files then dirs: last wins", []string{"--only-files", "--only-dirs"}, want(func(o *ls.Options) { o.Only = ls.OnlyDirs })},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil {
				t.Fatalf("buildOptions(%v): %v", tt.args, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildOptions(%v) =\n%+v, want\n%+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestBuildViewErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"level alone", []string{"--level=2"}, "--level needs -R or --tree"},
		{"level zero", []string{"--tree", "--level=0"}, "invalid level: '0'"},
		{"level word", []string{"--tree", "--level=deep"}, "invalid level: 'deep'"},
		{"bad theme", []string{"--theme=neon"}, "invalid argument 'neon' for '--theme'"},
		{"bad icons", []string{"--icons=x"}, "invalid argument 'x' for '--icons'"},
		{"bad color scale", []string{"--color-scale=x"}, "invalid argument 'x' for '--color-scale'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := options(t, tt.args...)
			var usage *core.UsageError
			if !errors.As(err, &usage) || !strings.HasPrefix(usage.Msg, tt.want) {
				t.Fatalf("err = %v, want a usage error starting %q", err, tt.want)
			}
		})
	}
}

func TestApplyView(t *testing.T) {
	prettyOn := func(o *ls.Options, color ls.When) {
		o.Pretty, o.Color, o.HumanReadable, o.GroupDirectories, o.Header = true, color, true, true, true
	}
	tests := []struct {
		name     string
		args     []string
		env      map[string]string
		terminal bool
		want     ls.Options
		warning  string
	}{
		{"pipe: GNU's output", nil, nil, false, want(nil), ""},
		{"terminal: pretty by default", nil, nil, true, want(func(o *ls.Options) { prettyOn(o, ls.WhenAuto) }), ""},
		{"GNUX_PRETTY=never on a terminal", nil, map[string]string{"GNUX_PRETTY": "never"}, true, want(nil), ""},
		{"GNUX_PRETTY=always through a pipe", nil, map[string]string{"GNUX_PRETTY": "always"}, false,
			want(func(o *ls.Options) { prettyOn(o, ls.WhenAlways) }), ""},
		{"--pretty=never beats GNUX_PRETTY", []string{"--pretty=never"}, map[string]string{"GNUX_PRETTY": "always"}, true, want(nil), ""},
		{"bare --pretty through a pipe", []string{"--pretty"}, nil, false, want(func(o *ls.Options) { prettyOn(o, ls.WhenAlways) }), ""},
		{"explicit options win", []string{"--color=never", "--si"}, nil, true, want(func(o *ls.Options) {
			prettyOn(o, ls.WhenNever)
			o.HumanReadable, o.SI = false, true
		}), ""},
		{"--zero keeps color off", []string{"--zero"}, nil, true, want(func(o *ls.Options) {
			prettyOn(o, ls.WhenNever)
			o.Zero, o.Format, o.Quoting, o.ShowControlChars = true, ls.FormatSingle, ls.QuoteLiteral, true
		}), ""},
		{"a block size variable keeps its sizes", nil, map[string]string{"BLOCK_SIZE": "K"}, true, want(func(o *ls.Options) {
			prettyOn(o, ls.WhenAuto)
			o.HumanReadable = false
		}), ""},
		{"NO_COLOR keeps it plain", nil, map[string]string{"NO_COLOR": "1"}, true, want(func(o *ls.Options) { prettyOn(o, ls.WhenNever) }), ""},
		{"GNUX_ICONS", nil, map[string]string{"GNUX_ICONS": "auto"}, false, want(func(o *ls.Options) { o.Icons = ls.WhenAuto }), ""},
		{"--icons beats GNUX_ICONS", []string{"--icons=never"}, map[string]string{"GNUX_ICONS": "always"}, false, want(nil), ""},
		{"an invalid variable is ignored", nil, map[string]string{"GNUX_PRETTY": "sometimes"}, false, want(nil),
			"lsx: ignoring invalid value of environment variable GNUX_PRETTY: 'sometimes'\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet(ls.Name, pflag.ContinueOnError)
			Register(fs)
			if err := fs.Parse(tt.args); err != nil {
				t.Fatal(err)
			}
			got, err := buildOptions(fs)
			if err != nil {
				t.Fatal(err)
			}
			var errb bytes.Buffer
			env := func(k string) string { return tt.env[k] }
			if err := applyView(fs, &got, tt.terminal, env, core.Streams{Err: &errb}); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got\n%+v, want\n%+v", got, tt.want)
			}
			if errb.String() != tt.warning {
				t.Errorf("stderr = %q, want %q", errb.String(), tt.warning)
			}
		})
	}
}
