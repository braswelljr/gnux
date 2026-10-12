package mktemp_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/mktemp"
)

// zeros is a random source that always yields the same name ("aaa...").
type zeros struct{}

func (zeros) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func run(t *testing.T, opts mktemp.Options, tmp, env string, setup func(*mktemp.Mktemp), args ...string) (string, string, error) {
	t.Helper()
	var out, errb bytes.Buffer
	m := mktemp.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	m.Getenv = func(k string) string {
		if k == "TMPDIR" {
			return env
		}
		return ""
	}
	m.TempDir = func() string { return tmp }
	if setup != nil {
		setup(m)
	}
	err := m.Run(args)
	return out.String(), errb.String(), err
}

func TestDefaultTemplate(t *testing.T) {
	tmp := t.TempDir()
	out, _, err := run(t, mktemp.Options{}, tmp, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	name := strings.TrimSuffix(out, "\n")
	if want := regexp.MustCompile(`^` + regexp.QuoteMeta(filepath.Join(tmp, "tmp.")) + `[A-Za-z0-9]{10}$`); !want.MatchString(name) {
		t.Fatalf("name = %q", name)
	}
	info, err := os.Stat(name)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() != 0 {
		t.Fatal("file not empty")
	}
}

func TestTMPDIRWinsOverSystemDefault(t *testing.T) {
	sys, env := t.TempDir(), t.TempDir()
	out, _, err := run(t, mktemp.Options{}, sys, env, nil)
	if err != nil || !strings.HasPrefix(out, filepath.Join(env, "tmp.")) {
		t.Fatalf("out=%q err=%v", out, err)
	}
	// An explicit -p DIR wins over TMPDIR.
	dir := t.TempDir()
	out, _, err = run(t, mktemp.Options{UseTmpDir: true, TmpDir: dir}, sys, env, nil, "xXXX")
	if err != nil || !strings.HasPrefix(out, filepath.Join(dir, "x")) {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestTemplateInCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	tmpl := filepath.Join(dir, "build.XXXX")
	out, _, err := run(t, mktemp.Options{}, t.TempDir(), "", nil, tmpl)
	if err != nil || !strings.HasPrefix(out, filepath.Join(dir, "build.")) || len(strings.TrimSpace(out)) != len(tmpl) {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestDirectory(t *testing.T) {
	tmp := t.TempDir()
	out, _, err := run(t, mktemp.Options{Directory: true}, tmp, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(strings.TrimSpace(out))
	if err != nil || !info.IsDir() {
		t.Fatalf("not a directory: %v", err)
	}
}

func TestSuffix(t *testing.T) {
	dir := t.TempDir()
	out, _, err := run(t, mktemp.Options{SuffixSet: true, Suffix: ".txt"}, dir, "", nil, filepath.Join(dir, "aXXX"))
	if err != nil || !strings.HasSuffix(strings.TrimSpace(out), ".txt") {
		t.Fatalf("out=%q err=%v", out, err)
	}
	// Without --suffix, the text after the last X is the suffix.
	out, _, err = run(t, mktemp.Options{}, dir, "", nil, filepath.Join(dir, "bXXX.log"))
	if err != nil || !strings.HasSuffix(strings.TrimSpace(out), ".log") {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestDryRunCreatesNothing(t *testing.T) {
	dir := t.TempDir()
	out, _, err := run(t, mktemp.Options{DryRun: true}, dir, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(strings.TrimSpace(out)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("-u created %s", out)
	}
}

func TestRetriesAndGivesUp(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "faaa"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	_, errb, err := run(t, mktemp.Options{}, dir, "", func(m *mktemp.Mktemp) { m.Random = zeros{} }, filepath.Join(dir, "fXXX"))
	if !errors.Is(err, core.ErrFailed) || !strings.Contains(errb, "mktempx: failed to create file via template '") || !strings.Contains(errb, "File exists") {
		t.Fatalf("err=%v stderr=%q", err, errb)
	}
	_, errb, err = run(t, mktemp.Options{Quiet: true}, dir, "", func(m *mktemp.Mktemp) { m.Random = zeros{} }, filepath.Join(dir, "fXXX"))
	if !errors.Is(err, core.ErrFailed) || errb != "" {
		t.Fatalf("-q: err=%v stderr=%q", err, errb)
	}
}

func TestMissingDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nope")
	_, errb, err := run(t, mktemp.Options{Directory: true}, dir, "", nil)
	if !errors.Is(err, core.ErrFailed) || !strings.Contains(errb, "failed to create directory via template") || !strings.Contains(errb, "No such file or directory") {
		t.Fatalf("err=%v stderr=%q", err, errb)
	}
}

func TestLegacyT(t *testing.T) {
	tmp := t.TempDir()
	out, _, err := run(t, mktemp.Options{Legacy: true}, tmp, "", nil, "fooXXXX")
	if err != nil || !strings.HasPrefix(out, filepath.Join(tmp, "foo")) {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		opts mktemp.Options
		args []string
		want string
	}{
		{"too few", mktemp.Options{}, []string{"aXX"}, "too few X's in template 'aXX'"},
		{"none", mktemp.Options{}, []string{"plain"}, "too few X's in template 'plain'"},
		{"too many", mktemp.Options{}, []string{"aXXX", "bXXX"}, "too many templates"},
		{"suffix needs X", mktemp.Options{SuffixSet: true, Suffix: ".x"}, []string{"aXXXb"}, "with --suffix, template 'aXXXb' must end in X"},
		{"suffix with slash", mktemp.Options{SuffixSet: true, Suffix: "a/b"}, []string{"aXXX"}, "invalid suffix 'a/b', contains directory separator"},
		{"t with slash", mktemp.Options{Legacy: true}, []string{"a/bXXX"}, "invalid template, 'a/bXXX', contains directory separator"},
		{"tmpdir absolute", mktemp.Options{UseTmpDir: true}, []string{"/abs/XXX"}, "invalid template, '/abs/XXX'; with --tmpdir, it may not be absolute"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := run(t, tt.opts, t.TempDir(), "", nil, tt.args...)
			var u *core.UsageError
			if !errors.As(err, &u) || u.Msg != tt.want {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
		})
	}
}
