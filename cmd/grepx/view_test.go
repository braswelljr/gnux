package grepx

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core/grep"
	"github.com/braswelljr/gnux/internal/tool"
)

func TestApplyView(t *testing.T) {
	tests := []struct {
		name string
		args []string
		view tool.View
		want grep.Options
	}{
		{"off", nil, tool.View{}, grep.Options{}},
		{"on", nil, tool.View{Pretty: true, Color: true, Theme: "nord"},
			grep.Options{Pretty: true, Theme: "nord", LineNumber: true, Color: grep.ColorAlways}},
		{"uncolored", nil, tool.View{Pretty: true}, grep.Options{Pretty: true, LineNumber: true}},
		{"--color wins", []string{"--color=never"}, tool.View{Pretty: true, Color: true},
			grep.Options{Pretty: true, LineNumber: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := pflag.NewFlagSet("grepx", pflag.ContinueOnError)
			Register(fs)
			if err := fs.Parse(tt.args); err != nil {
				t.Fatal(err)
			}
			var o grep.Options
			if fs.Changed("color") {
				o.Color = grep.ColorNever
			}
			applyView(fs, &o, tt.view)
			if o.Pretty != tt.want.Pretty || o.Theme != tt.want.Theme || o.LineNumber != tt.want.LineNumber || o.Color != tt.want.Color {
				t.Errorf("got %+v, want %+v", o, tt.want)
			}
		})
	}
}
