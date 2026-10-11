package tool

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/internal/docmeta"
)

func TestUsesFlag(t *testing.T) {
	fs := pflag.NewFlagSet("rmx", pflag.ContinueOnError)
	fs.BoolP("force", "f", false, "")
	fs.BoolP("recursive", "r", false, "")
	fs.String("interactive", "", "")
	spec := Spec{Meta: docmeta.Meta{Name: "rmx"}, GNU: "rm"}

	tests := []struct {
		bash, flag string
		want       bool
	}{
		{"rmx -f a", "force", true},
		{"rmx -rf build/", "force", true},
		{"rm --force a", "force", true},
		{"rmx --interactive=once *.log", "interactive", true},
		{"rmx --interactive once", "interactive", true},
		{"rmx a.txt", "force", false},
		{"lsx -f | rmx a", "force", false},          // -f belongs to lsx
		{"rmx a && headx -r b", "recursive", false}, // after the connector
		{"rmx -- -f", "force", false},               // an operand, not an option
		{"headx -5 | rmx x", "force", false},
	}
	for _, tt := range tests {
		if got := usesFlag(tt.bash, spec, fs.Lookup(tt.flag)); got != tt.want {
			t.Errorf("usesFlag(%q, %s) = %v, want %v", tt.bash, tt.flag, got, tt.want)
		}
	}
}

func TestWrapWords(t *testing.T) {
	text := strings.Repeat("word ", 40)
	for _, line := range wrapWords(text, 30) {
		if len(line) > 30 {
			t.Errorf("line longer than 30 columns: %q", line)
		}
	}
	if got := wrapWords("", 30); len(got) != 1 || got[0] != "" {
		t.Errorf("wrapWords(\"\") = %q", got)
	}
}

func TestFlagLabel(t *testing.T) {
	fs := pflag.NewFlagSet("x", pflag.ContinueOnError)
	fs.BoolP("force", "f", false, "")
	fs.String("interactive", "", "prompt according to `WHEN`")
	fs.Lookup("interactive").NoOptDefVal = "always"
	fs.IntP("lines", "n", 10, "print the first `NUM` lines")

	for name, want := range map[string]string{
		"force":       "-f, --force",
		"interactive": "--interactive[=WHEN]",
		"lines":       "-n, --lines=NUM",
	} {
		if got := flagLabel(fs.Lookup(name)); got != want {
			t.Errorf("flagLabel(%s) = %q, want %q", name, got, want)
		}
	}
}

func TestColorOnlyWhenAsked(t *testing.T) {
	spec := Spec{Meta: docmeta.Meta{Name: "rmx", Use: "rmx [FILE]...", Short: "Remove."}, GNU: "rm"}
	fs := pflag.NewFlagSet("rmx", pflag.ContinueOnError)

	var plain, color bytes.Buffer
	spec.WriteHelp(&plain, fs, Style{})
	spec.WriteHelp(&color, fs, Style{Color: true})
	if strings.Contains(plain.String(), "\x1b[") {
		t.Error("plain style wrote escape sequences")
	}
	if !strings.Contains(color.String(), "\x1b[") {
		t.Error("color style wrote no escape sequences")
	}
	if StyleFor(&plain).Color {
		t.Error("StyleFor a buffer chose color")
	}
}
