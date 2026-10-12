package tee_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"testing/iotest"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tee"
)

// run executes the engine against in-memory streams.
func run(opts tee.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = tee.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestCopies(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")

	out, errb, err := run(tee.Options{}, "hello\n", a, b)
	if err != nil || errb != "" {
		t.Fatalf("err = %v, stderr = %q", err, errb)
	}
	if out != "hello\n" || read(t, a) != "hello\n" || read(t, b) != "hello\n" {
		t.Fatalf("stdout = %q, a = %q, b = %q", out, read(t, a), read(t, b))
	}
}

func TestNoOperands(t *testing.T) {
	out, errb, err := run(tee.Options{}, "just stdout")
	if err != nil || errb != "" || out != "just stdout" {
		t.Fatalf("out = %q, stderr = %q, err = %v", out, errb, err)
	}
}

func TestTruncateAndAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := run(tee.Options{Append: true}, "new\n", path); err != nil {
		t.Fatal(err)
	}
	if got := read(t, path); got != "old\nnew\n" {
		t.Fatalf("after -a: %q", got)
	}

	if _, _, err := run(tee.Options{}, "fresh\n", path); err != nil {
		t.Fatal(err)
	}
	if got := read(t, path); got != "fresh\n" {
		t.Fatalf("after truncating: %q", got)
	}
}

// TestDashIsAFile checks GNU 9 behavior: "-" names a file, not standard output.
func TestDashIsAFile(t *testing.T) {
	t.Chdir(t.TempDir())
	out, _, err := run(tee.Options{}, "x", "-")
	if err != nil || out != "x" {
		t.Fatalf("out = %q, err = %v", out, err)
	}
	if got := read(t, "-"); got != "x" {
		t.Fatalf("file '-' = %q", got)
	}
}

func TestUnopenableFile(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good")
	bad := filepath.Join(dir, "missing", "file")

	out, errb, err := run(tee.Options{}, "data", bad, good)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	if want := "teex: " + bad + ": No such file or directory\n"; errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
	if out != "data" || read(t, good) != "data" {
		t.Fatal("the remaining outputs were not written")
	}
}

func TestDirectoryOperand(t *testing.T) {
	dir := t.TempDir()
	out, errb, err := run(tee.Options{}, "data", dir)
	if !errors.Is(err, core.ErrFailed) || out != "data" || !strings.HasPrefix(errb, "teex: "+dir+": ") {
		t.Fatalf("out = %q, stderr = %q, err = %v", out, errb, err)
	}
}

// failWriter fails every write with err, counting the attempts.
type failWriter struct {
	err   error
	calls int
}

func (f *failWriter) Write([]byte) (int, error) {
	f.calls++
	return 0, f.err
}

func TestOutputErrors(t *testing.T) {
	pipe := os.NewSyscallError("write", syscall.EPIPE)
	other := errors.New("no space left on device")
	tests := []struct {
		name   string
		mode   tee.OutputError
		err    error
		stderr string
		failed bool
		file   string // what reaches the file; stdout is written first
	}{
		{"default stops quietly on a closed pipe", tee.OutputErrorDefault, pipe, "", false, ""},
		{"default reports and continues", tee.OutputErrorDefault, other,
			"teex: standard output: No space left on device\n", true, "abc"},
		{"warn reports a pipe", tee.OutputErrorWarn, pipe,
			"teex: standard output: Broken pipe\n", true, "abc"},
		{"warn-nopipe drops a pipe", tee.OutputErrorWarnNoPipe, pipe, "", false, "abc"},
		{"warn-nopipe reports others", tee.OutputErrorWarnNoPipe, other,
			"teex: standard output: No space left on device\n", true, "abc"},
		{"exit stops on a pipe", tee.OutputErrorExit, pipe,
			"teex: standard output: Broken pipe\n", true, ""},
		{"exit-nopipe drops a pipe", tee.OutputErrorExitNoPipe, pipe, "", false, "abc"},
		{"exit-nopipe stops on others", tee.OutputErrorExitNoPipe, other,
			"teex: standard output: No space left on device\n", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "f")
			var errb bytes.Buffer
			w := &failWriter{err: tt.err}
			s := core.Streams{In: iotest.OneByteReader(strings.NewReader("abc")), Out: w, Err: &errb}
			err := tee.New(tee.Options{OutputError: tt.mode}, s).Run([]string{path})
			if errors.Is(err, core.ErrFailed) != tt.failed {
				t.Fatalf("err = %v, failed want %v", err, tt.failed)
			}
			got := errb.String()
			// The text of EPIPE differs by platform; only its shape matters.
			if tt.err == pipe && tt.stderr != "" {
				if !strings.HasPrefix(got, "teex: standard output: ") || strings.Count(got, "\n") != 1 {
					t.Fatalf("stderr = %q", got)
				}
			} else if got != tt.stderr {
				t.Fatalf("stderr = %q, want %q", got, tt.stderr)
			}
			if w.calls != 1 {
				t.Fatalf("stdout written %d times after failing, want 1", w.calls)
			}
			if f := read(t, path); f != tt.file {
				t.Fatalf("file = %q, want %q", f, tt.file)
			}
		})
	}
}

// TestAllOutputsGone checks that tee stops reading once nothing is left to
// write to.
func TestAllOutputsGone(t *testing.T) {
	w := &failWriter{err: os.NewSyscallError("write", syscall.EPIPE)}
	in := &countReader{r: iotest.OneByteReader(strings.NewReader("abcdef"))}
	err := tee.New(tee.Options{OutputError: tee.OutputErrorWarnNoPipe},
		core.Streams{In: in, Out: w, Err: &bytes.Buffer{}}).Run(nil)
	if err != nil || in.reads != 1 {
		t.Fatalf("err = %v, reads = %d", err, in.reads)
	}
}

type countReader struct {
	r     interface{ Read([]byte) (int, error) }
	reads int
}

func (c *countReader) Read(p []byte) (int, error) {
	c.reads++
	return c.r.Read(p)
}

func TestReadError(t *testing.T) {
	var out, errb bytes.Buffer
	in := iotest.TimeoutReader(iotest.OneByteReader(strings.NewReader("abc")))
	err := tee.New(tee.Options{}, core.Streams{In: in, Out: &out, Err: &errb}).Run(nil)
	if !errors.Is(err, core.ErrFailed) || out.String() != "a" || !strings.HasPrefix(errb.String(), "teex: read error: ") {
		t.Fatalf("out = %q, stderr = %q, err = %v", out.String(), errb.String(), err)
	}
}
