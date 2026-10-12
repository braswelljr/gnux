package commx

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/comm"
)

func options(t *testing.T, args ...string) (comm.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(comm.Name, pflag.ContinueOnError)
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
		want comm.Options
	}{
		{"defaults", nil, comm.Options{}},
		{"-1", []string{"-1"}, comm.Options{Suppress1: true}},
		{"-2", []string{"-2"}, comm.Options{Suppress2: true}},
		{"-3", []string{"-3"}, comm.Options{Suppress3: true}},
		{"-12", []string{"-12"}, comm.Options{Suppress1: true, Suppress2: true}},
		{"-123", []string{"-123"}, comm.Options{Suppress1: true, Suppress2: true, Suppress3: true}},
		{"long names", []string{"--suppress-file1", "--suppress-file2", "--suppress-common"}, comm.Options{Suppress1: true, Suppress2: true, Suppress3: true}},
		{"check order", []string{"--check-order"}, comm.Options{Check: comm.CheckOrder}},
		{"nocheck order", []string{"--nocheck-order"}, comm.Options{Check: comm.NoCheckOrder}},
		{"check wins after", []string{"--nocheck-order", "--check-order"}, comm.Options{Check: comm.CheckOrder}},
		{"check wins before", []string{"--check-order", "--nocheck-order"}, comm.Options{Check: comm.CheckOrder}},
		{"output delimiter", []string{"--output-delimiter=,"}, comm.Options{OutputDelimiter: ",", OutputDelimiterSet: true}},
		{"empty output delimiter", []string{"--output-delimiter="}, comm.Options{OutputDelimiterSet: true}},
		{"same output delimiter twice", []string{"--output-delimiter=,", "--output-delimiter=,"}, comm.Options{OutputDelimiter: ",", OutputDelimiterSet: true}},
		{"total", []string{"--total"}, comm.Options{Total: true}},
		{"zero terminated", []string{"-z"}, comm.Options{ZeroTerminated: true}},
		{"zero terminated long", []string{"--zero-terminated"}, comm.Options{ZeroTerminated: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil {
				t.Fatalf("buildOptions(%v): %v", tt.args, err)
			}
			if got != tt.want {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestBuildOptionsInvalid(t *testing.T) {
	_, err := options(t, "--output-delimiter=,", "--output-delimiter=;")
	var fatal *fatalError
	if !errors.As(err, &fatal) || fatal.msg != "multiple output delimiters specified" {
		t.Fatalf("err = %v", err)
	}
}

func TestCommand(t *testing.T) {
	dir := t.TempDir()
	b := filepath.Join(dir, "b")
	if err := os.WriteFile(b, []byte("b\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr string
	}{
		{"default", []string{"-", b}, "a\n\t\tb\n\tc\n", ""},
		{"common", []string{"-12", "-", b}, "b\n", ""},
		{"two delimiters", []string{"--output-delimiter=x", "--output-delimiter=y", "-", b}, "", "commx: multiple output delimiters specified\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			c := Spec().Command(core.Streams{In: strings.NewReader("a\nb\n"), Out: &out, Err: &errb})
			c.SetArgs(tt.args)
			err := c.Execute()
			if tt.wantErr != "" {
				if !errors.Is(err, core.ErrFailed) || errb.String() != tt.wantErr {
					t.Fatalf("err = %v, stderr = %q", err, errb.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute: %v (stderr %q)", err, errb.String())
			}
			if out.String() != tt.want {
				t.Fatalf("stdout = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != comm.Name || !strings.HasPrefix(m.Use, comm.Name+" ") {
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
