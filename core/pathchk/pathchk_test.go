package pathchk_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/pathchk"
)

func run(t *testing.T, opts pathchk.Options, args ...string) (string, error) {
	t.Helper()
	var errb bytes.Buffer
	p := pathchk.New(opts, core.Streams{In: strings.NewReader(""), Out: &errb, Err: &errb})
	p.NameMax = func(string) (int, error) { return 255, nil }
	p.PathMax = func(string) (int, error) { return 4096, nil }
	err := p.Run(args)
	return errb.String(), err
}

func TestPortable(t *testing.T) {
	long := strings.Repeat("0", 300)
	tests := []struct {
		name string
		opts pathchk.Options
		file string
		want string
	}{
		{"portable name", pathchk.Options{Portable: true}, "a/b-c_d.e", ""},
		{"space", pathchk.Options{Portable: true}, "a b/c", "pathchkx: non-portable character ' ' in file name 'a b/c'\n"},
		{"empty", pathchk.Options{Portable: true}, "", "pathchkx: empty file name\n"},
		{"empty with -P", pathchk.Options{Extra: true}, "", "pathchkx: empty file name\n"},
		{"long component", pathchk.Options{Portable: true}, "abcdefghijklmnop",
			"pathchkx: limit 14 exceeded by length 16 of file name component 'abcdefghijklmnop'\n"},
		{"long path", pathchk.Options{Portable: true}, long,
			"pathchkx: limit 255 exceeded by length 300 of file name '" + long + "'\n"},
		{"leading hyphen", pathchk.Options{Extra: true}, "x/-y", "pathchkx: leading '-' in a component of file name 'x/-y'\n"},
		{"hyphen inside", pathchk.Options{Extra: true}, "x/y-", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := run(t, tt.opts, tt.file)
			if got != tt.want || (err == nil) != (tt.want == "") {
				t.Fatalf("stderr = %q, err = %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestAgainstTheSystem(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := run(t, pathchk.Options{}, file, filepath.Join(dir, "new", "name")); err != nil {
		t.Fatalf("existing and creatable names: %v %q", err, got)
	}
	got, err := run(t, pathchk.Options{}, "")
	if !errors.Is(err, core.ErrFailed) || got != "pathchkx: '': No such file or directory\n" {
		t.Fatalf("empty name: %v %q", err, got)
	}
	got, _ = run(t, pathchk.Options{}, file+"/x")
	if want := "pathchkx: " + file + "/x: Not a directory\n"; got != want {
		t.Fatalf("under a file: %q, want %q", got, want)
	}
}

func TestComponentLimitFromTheSystem(t *testing.T) {
	var errb bytes.Buffer
	p := pathchk.New(pathchk.Options{}, core.Streams{In: strings.NewReader(""), Out: &errb, Err: &errb})
	p.NameMax = func(string) (int, error) { return 20, nil }
	name := strings.Repeat("n", 21)
	if err := p.Run([]string{"missing/" + name, "ok"}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if want := "pathchkx: limit 20 exceeded by length 21 of file name component '" + name + "'\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
}

func TestMissingOperand(t *testing.T) {
	_, err := run(t, pathchk.Options{})
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "missing operand" {
		t.Fatalf("err = %v", err)
	}
}
