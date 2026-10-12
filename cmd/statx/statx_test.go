package statx

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/stat"
)

func options(t *testing.T, args ...string) (stat.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(stat.Name, pflag.ContinueOnError)
	Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return buildOptions(fs)
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want stat.Options
	}{
		{"defaults", nil, stat.Options{}},
		{"-L -f -t", []string{"-Lft"}, stat.Options{Dereference: true, FileSystem: true, Terse: true}},
		{"-c", []string{"-c", "%s"}, stat.Options{Format: "%s", FormatSet: true}},
		{"--printf", []string{"--printf", `%s\n`}, stat.Options{Printf: `%s\n`, PrintfSet: true}},
		{"last wins", []string{"-c", "%s", "--printf", "%n"}, stat.Options{Printf: "%n", PrintfSet: true}},
		{"last wins the other way", []string{"--printf", "%n", "-c", "%s"}, stat.Options{Format: "%s", FormatSet: true}},
		{"empty format", []string{"--format="}, stat.Options{FormatSet: true}},
		{"--cached", []string{"--cached=never"}, stat.Options{}},
		{"--cached abbreviated", []string{"--cached=al"}, stat.Options{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil || got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, %v; want %+v", tt.args, got, err, tt.want)
			}
		})
	}
	_, err := options(t, "--cached=sometimes")
	var usage *core.UsageError
	if !errors.As(err, &usage) || !strings.HasPrefix(usage.Msg, "invalid argument 'sometimes' for '--cached'") {
		t.Fatalf("--cached=sometimes: %v", err)
	}
}

func TestCommand(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(f, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	c := Spec().Command(core.Streams{In: strings.NewReader(""), Out: &out, Err: &bytes.Buffer{}})
	c.SetArgs([]string{"--printf", `%s\n`, f})
	if err := c.Execute(); err != nil || out.String() != "3\n" {
		t.Fatalf("Execute: %v, stdout = %q", err, out.String())
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != stat.Name || !strings.HasPrefix(m.Use, stat.Name+" ") {
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
