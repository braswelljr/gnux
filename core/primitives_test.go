package core_test

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/braswelljr/gnux/core"
)

func TestErrorTypes(t *testing.T) {
	err := core.Usagef("missing operand after '%s'", "x")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "missing operand after 'x'" || err.Error() != usage.Msg {
		t.Fatalf("Usagef = %#v", err)
	}
	if got := (&core.ExitError{Code: 125}).Error(); got != "exit status 125" {
		t.Fatalf("ExitError = %q", got)
	}
}

func TestDiag(t *testing.T) {
	var out, errb bytes.Buffer
	d := core.NewDiag("toolx", core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	if d.Prog() != "toolx" || d.Failed() || d.Result() != nil {
		t.Fatal("a new Diag has failed")
	}
	d.Warnf("skipping '%s'", "a")
	if d.Failed() {
		t.Fatal("a warning counted as a failure")
	}
	d.Printf("removed '%s'", "b")
	d.Errorf("cannot remove '%s': %s", "c", "Permission denied")
	if !d.Failed() || !errors.Is(d.Result(), core.ErrFailed) {
		t.Fatal("an error did not count")
	}
	if want := "toolx: skipping 'a'\ntoolx: cannot remove 'c': Permission denied\n"; errb.String() != want {
		t.Fatalf("stderr = %q; want %q", errb.String(), want)
	}
	if out.String() != "removed 'b'\n" {
		t.Fatalf("stdout = %q", out.String())
	}
}

// TestDiagIsSafeForConcurrentUse: whole lines, and every error counted.
func TestDiagIsSafeForConcurrentUse(t *testing.T) {
	var errb bytes.Buffer
	d := core.NewDiag("toolx", core.Streams{In: strings.NewReader(""), Out: io.Discard, Err: &errb})
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			for range 20 {
				d.Errorf("failure")
			}
		})
	}
	wg.Wait()
	lines := strings.Split(strings.TrimSuffix(errb.String(), "\n"), "\n")
	if len(lines) != 1000 {
		t.Fatalf("%d lines, want 1000", len(lines))
	}
	for _, l := range lines {
		if l != "toolx: failure" {
			t.Fatalf("torn line %q", l)
		}
	}
}

func TestPrompter(t *testing.T) {
	tests := []struct {
		input string
		want  []bool
	}{
		{"y\n", []bool{true}},
		{"Yes please\n", []bool{true}},
		{"  y\n", []bool{true}},
		{"n\nY\n", []bool{false, true}},
		{"\n", []bool{false}},
		{"maybe\n", []bool{false}},
		{"y", []bool{true}}, // the last line needs no newline
		{"", []bool{false}}, // end of input is no
		{"y\n", []bool{true, false}},
	}
	for _, tt := range tests {
		var errb bytes.Buffer
		p := core.NewPrompter("rmx", core.Streams{In: strings.NewReader(tt.input), Out: io.Discard, Err: &errb})
		for i, want := range tt.want {
			if got := p.Ask("remove 'f'?"); got != want {
				t.Fatalf("input %q, answer %d = %v; want %v", tt.input, i, got, want)
			}
		}
		if want := strings.Repeat("rmx: remove 'f'? ", len(tt.want)); errb.String() != want {
			t.Fatalf("input %q: prompts %q; want %q", tt.input, errb.String(), want)
		}
	}

	var errb bytes.Buffer
	p := core.NewPrompter("rmx", core.Streams{In: strings.NewReader("y\n"), Out: io.Discard, Err: &errb})
	p.Fn = func(q string) bool { return q == "ok?" }
	if !p.Ask("ok?") || p.Ask("other?") || errb.Len() != 0 {
		t.Fatal("Fn was not consulted instead of the input")
	}
}

func TestDescribe(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty")
	full := filepath.Join(dir, "full")
	link := filepath.Join(dir, "link")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{dir: "directory", empty: "regular empty file", full: "regular file"}
	if err := os.Symlink(full, link); err == nil {
		cases[link] = "symbolic link"
	}
	for path, want := range cases {
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := core.Describe(info); got != want {
			t.Errorf("Describe(%s) = %q; want %q", path, got, want)
		}
	}
	if runtime.GOOS != "windows" {
		if info, err := os.Lstat(os.DevNull); err == nil {
			if got := core.Describe(info); got != "character special file" {
				t.Errorf("Describe(%s) = %q", os.DevNull, got)
			}
		}
	}
}

func TestIsRoot(t *testing.T) {
	root := filepath.VolumeName(t.TempDir()) + string(filepath.Separator)
	for path, want := range map[string]bool{
		root:                       true,
		filepath.Join(root, "."):   true,
		filepath.Join(root, ".."):  true,
		t.TempDir():                false,
		".":                        false,
		filepath.Join(root, "tmp"): false,
	} {
		if got := core.IsRoot(path); got != want {
			t.Errorf("IsRoot(%q) = %v; want %v", path, got, want)
		}
	}
}

func TestPlural(t *testing.T) {
	if core.Plural(0) != "s" || core.Plural(1) != "" || core.Plural(2) != "s" {
		t.Fatal("Plural")
	}
}

func TestIsTerminal(t *testing.T) {
	if core.IsTerminal(&bytes.Buffer{}) || core.IsTerminal(nil) {
		t.Fatal("a buffer is not a terminal")
	}
	f, err := os.CreateTemp(t.TempDir(), "f")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if core.IsTerminal(f) {
		t.Fatal("a regular file is not a terminal")
	}
}

func TestStreams(t *testing.T) {
	s := core.Stdio()
	if s.In != os.Stdin || s.Out != os.Stdout || s.Err != os.Stderr {
		t.Fatal("Stdio is not the process's streams")
	}
	d := core.Discard()
	if d.In == nil || d.Out == nil || d.Err == nil {
		t.Fatal("Discard left a stream unset")
	}
	if n, err := d.In.Read(make([]byte, 1)); n != 0 || err != io.EOF {
		t.Fatalf("Discard's input read %d, %v", n, err)
	}
}

func TestWritable(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if !core.Writable(f) {
		t.Fatal("a new file is not writable")
	}
	if err := os.Chmod(f, 0o444); err != nil {
		t.Fatal(err)
	}
	if core.Writable(f) && os.Geteuid() != 0 {
		t.Fatal("a read-only file is writable")
	}
}

func TestDeviceID(t *testing.T) {
	a, err := os.Stat(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.Stat(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	da, okA := core.DeviceID(a)
	db, okB := core.DeviceID(b)
	if okA != okB || (okA && da != db) {
		t.Fatalf("two temporary directories on different devices: %v %v, %v %v", da, okA, db, okB)
	}
}

// slowReader hands out its input one Read at a time, as a pipe from a slow
// producer does.
type slowReader struct{ chunks []string }

func (r *slowReader) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.chunks[0])
	r.chunks = r.chunks[1:]
	return n, nil
}

func TestFlushIfIdle(t *testing.T) {
	var out bytes.Buffer
	w := bufio.NewWriter(&out)
	in := bufio.NewReader(&slowReader{chunks: []string{"a\nb\n", "c\n"}})

	line, _ := in.ReadString('\n')
	_, _ = w.WriteString(line)
	core.FlushIfIdle(w, in) // "b\n" is still buffered: no flush yet
	if out.Len() != 0 {
		t.Fatalf("flushed with input still buffered: %q", out.String())
	}
	line, _ = in.ReadString('\n')
	_, _ = w.WriteString(line)
	core.FlushIfIdle(w, in) // the buffer has run dry: the next read would wait
	if out.String() != "a\nb\n" {
		t.Fatalf("not flushed when idle: %q", out.String())
	}
}
