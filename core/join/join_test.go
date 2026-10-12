package join_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/join"
)

func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func run(opts join.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = join.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestJoin(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "1 a\n2 b\n3 c\n")
	b := write(t, dir, "b", "1 x\n3 z\n4 w\n")
	tests := []struct {
		name string
		opts join.Options
		want string
	}{
		{"default", join.Options{}, "1 a x\n3 c z\n"},
		{"-a1", join.Options{Unpaired1: true}, "1 a x\n2 b\n3 c z\n"},
		{"-a1 -a2", join.Options{Unpaired1: true, Unpaired2: true}, "1 a x\n2 b\n3 c z\n4 w\n"},
		{"-v2", join.Options{Unpaired2: true, OnlyUnpaired: true}, "4 w\n"},
		{"-v1", join.Options{Unpaired1: true, OnlyUnpaired: true}, "2 b\n"},
		{"-o", join.Options{Format: []join.Field{{File: 2, Field: 2}, {File: 1, Field: 2}}}, "x a\nz c\n"},
		{"-o 0", join.Options{Format: []join.Field{{}, {File: 2, Field: 2}}}, "1 x\n3 z\n"},
		{"outer join", join.Options{Unpaired1: true, Unpaired2: true, Empty: "NULL", EmptySet: true, AutoFormat: true},
			"1 a x\n2 b NULL\n3 c z\n4 NULL w\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, "", a, b)
			if err != nil || out != tt.want {
				t.Fatalf("out = %q, err = %v (stderr %q), want %q", out, err, errb, tt.want)
			}
		})
	}
}

func TestSeparatorFieldsAndCase(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "x,1\ny,2\n")
	b := write(t, dir, "b", "1,one\n2,two\n")
	out, _, err := run(join.Options{Separator: ',', SeparatorSet: true, Field1: 2}, "", a, b)
	if err != nil || out != "1,x,one\n2,y,two\n" {
		t.Fatalf("-t, -1 2: %q, %v", out, err)
	}

	c := write(t, dir, "c", "A 1\n")
	d := write(t, dir, "d", "a 2\n")
	if out, _, _ := run(join.Options{}, "", c, d); out != "" {
		t.Fatalf("case-sensitive: %q", out)
	}
	if out, _, _ := run(join.Options{IgnoreCase: true}, "", c, d); out != "A 1 2\n" {
		t.Fatalf("-i: %q", out)
	}
}

func TestHeaderAndStdin(t *testing.T) {
	dir := t.TempDir()
	b := write(t, dir, "b", "id age\n1 30\n")
	out, _, err := run(join.Options{Header: true}, "id name\n1 ann\n", "-", b)
	if err != nil || out != "id name age\n1 ann 30\n" {
		t.Fatalf("--header: %q, %v", out, err)
	}
}

func TestOrder(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "2 b\n1 a\n")
	b := write(t, dir, "b", "1 x\n2 y\n")
	_, errb, err := run(join.Options{Check: join.CheckOrder}, "", a, b)
	if !errors.Is(err, core.ErrFailed) || !strings.Contains(errb, "joinx: "+a+":2: is not sorted: 1 a\n") {
		t.Fatalf("--check-order: err = %v, stderr = %q", err, errb)
	}
	if _, errb, err := run(join.Options{Check: join.NoCheckOrder}, "", a, b); err != nil || errb != "" {
		t.Fatalf("--nocheck-order: err = %v, stderr = %q", err, errb)
	}
}

func TestErrors(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "1\n")
	for _, tt := range []struct {
		args []string
		msg  string
	}{
		{nil, "missing operand"},
		{[]string{a}, "missing operand after '" + a + "'"},
		{[]string{a, a, a}, "extra operand '" + a + "'"},
	} {
		_, _, err := run(join.Options{}, "", tt.args...)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != tt.msg {
			t.Errorf("Run(%q) = %v, want %q", tt.args, err, tt.msg)
		}
	}

	_, errb, err := run(join.Options{}, "", "-", "-")
	if !errors.Is(err, core.ErrFailed) || errb != "joinx: both files cannot be standard input\n" {
		t.Fatalf("- -: %v %q", err, errb)
	}
	missing := filepath.Join(dir, "missing")
	_, errb, err = run(join.Options{}, "", missing, a)
	if !errors.Is(err, core.ErrFailed) || errb != "joinx: "+missing+": No such file or directory\n" {
		t.Fatalf("missing: %v %q", err, errb)
	}
}
