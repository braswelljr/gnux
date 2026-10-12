package chmod_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/chmod"
)

// run executes the engine with a fixed umask against in-memory streams.
func run(t *testing.T, opts chmod.Options, umask uint32, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	var out, errb bytes.Buffer
	c := chmod.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.Umask = func() uint32 { return umask }
	err = c.Run(args)
	return out.String(), errb.String(), err
}

func file(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestUsageErrors(t *testing.T) {
	f := file(t, t.TempDir(), "f")
	tests := []struct {
		name string
		opts chmod.Options
		args []string
		want string
	}{
		{"no operands", chmod.Options{}, nil, "missing operand"},
		{"mode only", chmod.Options{}, []string{"755"}, "missing operand after '755'"},
		{"invalid symbolic", chmod.Options{}, []string{"u+q", f}, "invalid mode: 'u+q'"},
		{"invalid octal", chmod.Options{}, []string{"18", f}, "invalid mode: '18'"},
		{"too large", chmod.Options{}, []string{"17777", f}, "invalid mode: '17777'"},
		{"no operator", chmod.Options{}, []string{"u", f}, "invalid mode: 'u'"},
		{"empty", chmod.Options{}, []string{"", f}, "invalid mode: ''"},
		{"reference with no file", chmod.Options{Reference: f}, nil, "missing operand"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := run(t, tt.opts, 0, tt.args...)
			var usage *core.UsageError
			if !errors.As(err, &usage) || usage.Msg != tt.want {
				t.Fatalf("err = %v, want usage error %q", err, tt.want)
			}
		})
	}
}

func TestMissingFileDoesNotStopTheRest(t *testing.T) {
	dir := t.TempDir()
	f := file(t, dir, "f")
	missing := filepath.Join(dir, "missing")
	var seen []string
	var out, errb bytes.Buffer
	c := chmod.New(chmod.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	c.ChmodFn = func(p string, _ os.FileMode) error { seen = append(seen, p); return nil }
	c.Umask = func() uint32 { return 0 }
	err := c.Run([]string{"000", missing, f})
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "chmodx: cannot access '" + missing + "': No such file or directory\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
	if len(seen) != 1 || seen[0] != f {
		t.Fatalf("changed %v, want only %s", seen, f)
	}
}

func TestSilent(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	_, errb, err := run(t, chmod.Options{Silent: true}, 0, "644", missing)
	if !errors.Is(err, core.ErrFailed) || errb != "" {
		t.Fatalf("err = %v, stderr = %q: -f must fail silently", err, errb)
	}
}

func TestReferenceMissing(t *testing.T) {
	dir := t.TempDir()
	f := file(t, dir, "f")
	_, errb, err := run(t, chmod.Options{Reference: filepath.Join(dir, "nope")}, 0, f)
	if !errors.Is(err, core.ErrFailed) || !strings.Contains(errb, "failed to get attributes of") {
		t.Fatalf("err = %v, stderr = %q", err, errb)
	}
}
