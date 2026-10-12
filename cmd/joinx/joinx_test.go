package joinx

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/join"
)

func options(t *testing.T, args ...string) (join.Options, error) {
	t.Helper()
	fs := pflag.NewFlagSet(join.Name, pflag.ContinueOnError)
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
		want join.Options
	}{
		{"defaults", nil, join.Options{}},
		{"-a 1", []string{"-a", "1"}, join.Options{Unpaired1: true}},
		{"-a 1 -a 2", []string{"-a1", "-a2"}, join.Options{Unpaired1: true, Unpaired2: true}},
		{"-v 2", []string{"-v", "2"}, join.Options{Unpaired2: true, OnlyUnpaired: true}},
		{"-e", []string{"-e", "NULL"}, join.Options{Empty: "NULL", EmptySet: true}},
		{"-e empty", []string{"-e", ""}, join.Options{EmptySet: true}},
		{"-i", []string{"-i"}, join.Options{IgnoreCase: true}},
		{"-j", []string{"-j", "2"}, join.Options{Field1: 2, Field2: 2}},
		{"-1 -2", []string{"-1", "3", "-2", "1"}, join.Options{Field1: 3, Field2: 1}},
		{"-j and agreeing -1", []string{"-j", "2", "-1", "2"}, join.Options{Field1: 2, Field2: 2}},
		{"-t,", []string{"-t,"}, join.Options{Separator: ',', SeparatorSet: true}},
		{"-t empty", []string{"-t", ""}, join.Options{Separator: '\n', SeparatorSet: true}},
		{"-t NUL", []string{"-t", `\0`}, join.Options{Separator: 0, SeparatorSet: true}},
		{"-o", []string{"-o", "1.2,2.3 0"}, join.Options{Format: []join.Field{{File: 1, Field: 2}, {File: 2, Field: 3}, {}}}},
		{"-o twice", []string{"-o", "1.1", "-o", "2.1"}, join.Options{Format: []join.Field{{File: 1, Field: 1}, {File: 2, Field: 1}}}},
		{"-o auto", []string{"-o", "auto"}, join.Options{AutoFormat: true}},
		{"check order", []string{"--check-order"}, join.Options{Check: join.CheckOrder}},
		{"nocheck order", []string{"--nocheck-order"}, join.Options{Check: join.NoCheckOrder}},
		{"check wins after", []string{"--nocheck-order", "--check-order"}, join.Options{Check: join.CheckOrder}},
		{"check wins before", []string{"--check-order", "--nocheck-order"}, join.Options{Check: join.CheckOrder}},
		{"header", []string{"--header"}, join.Options{Header: true}},
		{"-z", []string{"-z"}, join.Options{ZeroTerminated: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := options(t, tt.args...)
			if err != nil {
				t.Fatalf("buildOptions(%v): %v", tt.args, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildOptions(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestBuildOptionsInvalid(t *testing.T) {
	tests := []struct {
		args []string
		msg  string
	}{
		{[]string{"-a", "3"}, "invalid file number: '3'"},
		{[]string{"-v", "x"}, "invalid file number: 'x'"},
		{[]string{"-1", "0"}, "invalid field number: '0'"},
		{[]string{"-2", "x"}, "invalid field number: 'x'"},
		{[]string{"-j", "1", "-1", "2"}, "incompatible join fields 1, 2"},
		{[]string{"-t", "ab"}, "multi-character tab 'ab'"},
		{[]string{"-t", ",", "-t", ";"}, "incompatible tabs"},
		{[]string{"-e", "a", "-e", "b"}, "conflicting empty-field replacement strings"},
		{[]string{"-o", "1"}, "invalid field specifier: '1'"},
		{[]string{"-o", "3.1"}, "invalid file number in field spec: '3.1'"},
		{[]string{"-o", "1.0"}, "invalid field number: '0'"},
	}
	for _, tt := range tests {
		_, err := options(t, tt.args...)
		var fatal *fatalError
		if !errors.As(err, &fatal) || fatal.msg != tt.msg {
			t.Errorf("buildOptions(%q) = %v, want %q", tt.args, err, tt.msg)
		}
	}
}

func TestCommand(t *testing.T) {
	b := filepath.Join(t.TempDir(), "b")
	if err := os.WriteFile(b, []byte("1 x\n2 y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr string
	}{
		{"default", []string{"-", b}, "1 a x\n", ""},
		{"-v 2", []string{"-v", "2", "-", b}, "2 y\n", ""},
		{"bad file number", []string{"-a", "9", "-", b}, "", "joinx: invalid file number: '9'\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			c := Spec().Command(core.Streams{In: strings.NewReader("1 a\n"), Out: &out, Err: &errb})
			c.SetArgs(tt.args)
			err := c.Execute()
			if tt.wantErr != "" {
				if !errors.Is(err, core.ErrFailed) || errb.String() != tt.wantErr {
					t.Fatalf("err = %v, stderr = %q", err, errb.String())
				}
				return
			}
			if err != nil || out.String() != tt.want {
				t.Fatalf("Execute: %v, stdout = %q, want %q (stderr %q)", err, out.String(), tt.want, errb.String())
			}
		})
	}
}

func TestMeta(t *testing.T) {
	m := Meta()
	if m.Name != join.Name || !strings.HasPrefix(m.Use, join.Name+" ") {
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
