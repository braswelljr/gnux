package comm_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/comm"
)

func run(opts comm.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = comm.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestColumns(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "apple\nbanana\ncherry\n")
	b := write(t, dir, "b", "banana\ncherry\ndate\n")

	tests := []struct {
		name string
		opts comm.Options
		want string
	}{
		{"three columns", comm.Options{}, "apple\n\t\tbanana\n\t\tcherry\n\tdate\n"},
		{"-1", comm.Options{Suppress1: true}, "\tbanana\n\tcherry\ndate\n"},
		{"-2", comm.Options{Suppress2: true}, "apple\n\tbanana\n\tcherry\n"},
		{"-3", comm.Options{Suppress3: true}, "apple\n\tdate\n"},
		{"-12 common lines", comm.Options{Suppress1: true, Suppress2: true}, "banana\ncherry\n"},
		{"-23 only in file 1", comm.Options{Suppress2: true, Suppress3: true}, "apple\n"},
		{"-13 only in file 2", comm.Options{Suppress1: true, Suppress3: true}, "date\n"},
		{"-123", comm.Options{Suppress1: true, Suppress2: true, Suppress3: true}, ""},
		{"output delimiter", comm.Options{OutputDelimiter: "|", OutputDelimiterSet: true}, "apple\n||banana\n||cherry\n|date\n"},
		{"empty output delimiter is NUL", comm.Options{OutputDelimiterSet: true, Suppress1: true}, "\x00banana\n\x00cherry\ndate\n"},
		{"total", comm.Options{Total: true}, "apple\n\t\tbanana\n\t\tcherry\n\tdate\n1\t1\t2\ttotal\n"},
		{"total counts suppressed columns", comm.Options{Total: true, Suppress1: true, Suppress2: true, Suppress3: true}, "1\t1\t2\ttotal\n"},
		{"total with a delimiter", comm.Options{Total: true, Suppress3: true, OutputDelimiter: ",", OutputDelimiterSet: true}, "apple\n,date\n1,1,2,total\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, "", a, b)
			if err != nil || errb != "" {
				t.Fatalf("err = %v, stderr = %q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestInputs(t *testing.T) {
	dir := t.TempDir()
	noeol := write(t, dir, "noeol", "a\nb")
	empty := write(t, dir, "empty", "")
	b := write(t, dir, "b", "b\nc\n")

	out, _, err := run(comm.Options{}, "", noeol, b)
	if err != nil || out != "a\n\t\tb\n\tc\n" {
		t.Fatalf("missing newline: err = %v, stdout = %q", err, out)
	}
	out, _, err = run(comm.Options{}, "", empty, b)
	if err != nil || out != "\tb\n\tc\n" {
		t.Fatalf("empty file: err = %v, stdout = %q", err, out)
	}
	out, _, err = run(comm.Options{}, "a\nc\n", "-", b)
	if err != nil || out != "a\n\tb\n\t\tc\n" {
		t.Fatalf("stdin: err = %v, stdout = %q", err, out)
	}
	z1 := write(t, dir, "z1", "a\x00b\x00")
	z2 := write(t, dir, "z2", "b\x00")
	out, _, err = run(comm.Options{ZeroTerminated: true}, "", z1, z2)
	if err != nil || out != "a\x00\t\tb\x00" {
		t.Fatalf("-z: err = %v, stdout = %q", err, out)
	}
	// Comparison is by bytes (the C locale): uppercase sorts first.
	u1 := write(t, dir, "u1", "B\na\n")
	u2 := write(t, dir, "u2", "a\n")
	out, errb, err := run(comm.Options{}, "", u1, u2)
	if err != nil || errb != "" || out != "B\n\t\ta\n" {
		t.Fatalf("byte order: err = %v, stdout = %q, stderr = %q", err, out, errb)
	}
}

func TestOrder(t *testing.T) {
	dir := t.TempDir()
	unsorted := write(t, dir, "unsorted", "b\na\n")
	sorted := write(t, dir, "sorted", "a\nb\n")
	other := write(t, dir, "other", "c\n")

	// The same lines in both files: no line is unpairable, so the default
	// check never looks at the order, as in GNU.
	out, errb, err := run(comm.Options{}, "", unsorted, unsorted)
	if err != nil || errb != "" || out != "\t\tb\n\t\ta\n" {
		t.Fatalf("pairable: err = %v, stdout = %q, stderr = %q", err, out, errb)
	}

	// Once a line is unpairable, disorder is a warning, then a failure.
	out, errb, err = run(comm.Options{}, "", unsorted, other)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("default: err = %v", err)
	}
	if out != "b\na\n\tc\n" {
		t.Fatalf("default: stdout = %q", out)
	}
	if want := "commx: file 1 is not in sorted order\ncommx: input is not in sorted order\n"; errb != want {
		t.Fatalf("default: stderr = %q, want %q", errb, want)
	}

	// --check-order stops at the first disorder, after the lines before it.
	out, errb, err = run(comm.Options{Check: comm.CheckOrder}, "", unsorted, unsorted)
	if !errors.Is(err, core.ErrFailed) || errb != "commx: file 1 is not in sorted order\n" || out != "\t\tb\n" {
		t.Fatalf("--check-order: err = %v, stdout = %q, stderr = %q", err, out, errb)
	}

	// --nocheck-order never complains.
	_, errb, err = run(comm.Options{Check: comm.NoCheckOrder}, "", unsorted, other)
	if err != nil || errb != "" {
		t.Fatalf("--nocheck-order: err = %v, stderr = %q", err, errb)
	}

	_, errb, err = run(comm.Options{}, "", sorted, other)
	if err != nil || errb != "" {
		t.Fatalf("sorted: err = %v, stderr = %q", err, errb)
	}
}

func TestOperands(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{nil, "missing operand"},
		{[]string{"a"}, "missing operand after 'a'"},
		{[]string{"a", "b", "c"}, "extra operand 'c'"},
	}
	for _, tt := range tests {
		_, _, err := run(comm.Options{}, "", tt.args...)
		var usage *core.UsageError
		if !errors.As(err, &usage) || usage.Msg != tt.want {
			t.Fatalf("%v: err = %v, want usage error %q", tt.args, err, tt.want)
		}
	}

	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	_, errb, err := run(comm.Options{}, "", missing, missing)
	if !errors.Is(err, core.ErrFailed) || errb != "commx: "+missing+": No such file or directory\n" {
		t.Fatalf("missing: err = %v, stderr = %q", err, errb)
	}
}
