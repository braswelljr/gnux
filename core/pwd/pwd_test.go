package pwd_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/pwd"
)

func newEngine(opts pwd.Options) (*pwd.Pwd, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	p := pwd.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	return p, &out, &errb
}

func TestPhysicalResolvesSymlinks(t *testing.T) {
	real, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(real, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	p, out, _ := newEngine(pwd.Options{})
	p.Getwd = func() (string, error) { return link, nil }
	if err := p.Run(nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != real {
		t.Fatalf("pwd -P = %q, want %q", got, real)
	}
}

func TestLogical(t *testing.T) {
	real, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(real, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	tests := []struct {
		name, env, want string
	}{
		{"trusts a matching PWD", link, link},
		{"ignores an unset PWD", "", real},
		{"ignores a relative PWD", "link", real},
		{"ignores a PWD with dot components", link + string(filepath.Separator) + ".", real},
		{"ignores a PWD naming another directory", filepath.Dir(real), real},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, out, _ := newEngine(pwd.Options{Logical: true})
			p.Getwd = func() (string, error) { return real, nil }
			p.Getenv = func(string) string { return tt.env }
			if err := p.Run(nil); err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got := strings.TrimSpace(out.String()); got != tt.want {
				t.Fatalf("pwd -L = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetwdFailure(t *testing.T) {
	p, out, errb := newEngine(pwd.Options{})
	p.Getwd = func() (string, error) { return "", os.ErrNotExist }
	if err := p.Run(nil); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
	if want := "pwdx: cannot get current directory: No such file or directory\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
}
