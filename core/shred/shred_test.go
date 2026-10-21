package shred_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/shred"
)

// zeros is a random source that always gives zero bytes, which makes the
// "random" passes and the choice of patterns predictable.
type zeros struct{}

func (zeros) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func newEngine(opts shred.Options) (*shred.Shred, *bytes.Buffer) {
	var errb bytes.Buffer
	s := shred.New(opts, core.Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &errb})
	s.Random = zeros{}
	return s, &errb
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestOverwritesAndRoundsUp(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f")
	write(t, f, "hello")
	s, errb := newEngine(shred.Options{Verbose: true})
	if err := s.Run([]string{f}); err != nil {
		t.Fatalf("Run: %v (%s)", err, errb)
	}
	data, _ := os.ReadFile(f)
	if len(data) < 5 || len(data)%512 != 0 || bytes.Contains(data, []byte("hello")) {
		t.Fatalf("file is %d bytes, still holding the data: %v", len(data), bytes.Contains(data, []byte("hello")))
	}
	want := ""
	for i := 1; i <= 3; i++ {
		want += "shredx: " + f + ": pass " + string(rune('0'+i)) + "/3 (random)...\n"
	}
	if errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}
}

func TestExactAndZero(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f")
	write(t, f, "abc")
	s, _ := newEngine(shred.Options{Exact: true, Iterations: -1, Zero: true})
	if err := s.Run([]string{f}); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(f); !bytes.Equal(data, []byte{0, 0, 0}) {
		t.Fatalf("file = %q, want three zero bytes", data)
	}
}

func TestSize(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f")
	write(t, f, "hello")
	s, _ := newEngine(shred.Options{Size: 2, SizeSet: true, Iterations: -1, Zero: true})
	if err := s.Run([]string{f}); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(f); string(data) != "\x00\x00llo" {
		t.Fatalf("file = %q", data)
	}
}

// TestPatterns checks the passes GNU's table gives for -n 5: random passes
// first, in the middle and last, and the two one-bit patterns between them.
// With zero random bytes the shuffle swaps nothing, and each random pass
// moves the pattern it displaces to the end, which puts ffffff first.
func TestPatterns(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f")
	write(t, f, "x")
	s, errb := newEngine(shred.Options{Iterations: 5, Verbose: true})
	if err := s.Run([]string{f}); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"random", "ffffff", "random", "000000", "random"} {
		line := "pass " + string(rune('1'+i)) + "/5 (" + name + ")"
		if !strings.Contains(errb.String(), line) {
			t.Fatalf("stderr %q lacks %q", errb.String(), line)
		}
	}
}

func TestRemoveRenamesFirst(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "abc")
	write(t, f, "x")
	write(t, filepath.Join(dir, "0"), "")
	s, errb := newEngine(shred.Options{Iterations: -1, Remove: shred.Wipe, Verbose: true})
	if err := s.Run([]string{f}); err != nil {
		t.Fatal(err)
	}
	p := func(name string) string { return filepath.Join(dir, name) }
	want := "shredx: " + f + ": removing\n" +
		"shredx: " + f + ": renamed to " + p("000") + "\n" +
		"shredx: " + p("000") + ": renamed to " + p("00") + "\n" +
		"shredx: " + p("00") + ": renamed to " + p("1") + "\n" +
		"shredx: " + f + ": removed\n"
	if errb.String() != want {
		t.Fatalf("stderr =\n%s\nwant\n%s", errb, want)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "0" {
		t.Fatalf("left behind: %v", entries)
	}
}

func TestUnlink(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f")
	write(t, f, "x")
	s, errb := newEngine(shred.Options{Remove: shred.Unlink, Verbose: true, Iterations: -1})
	if err := s.Run([]string{f}); err != nil {
		t.Fatal(err)
	}
	if want := "shredx: " + f + ": removing\nshredx: " + f + ": removed\n"; errb.String() != want {
		t.Fatalf("stderr = %q", errb)
	}
}

func TestFailuresDoNotStopTheRest(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	write(t, f, "hello")
	missing := filepath.Join(dir, "nope")
	s, errb := newEngine(shred.Options{Iterations: 1})
	if err := s.Run([]string{missing, dir, f}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	want := "shredx: " + missing + ": failed to open for writing: No such file or directory\n" +
		"shredx: " + dir + ": failed to open for writing: Is a directory\n"
	if errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
	if data, _ := os.ReadFile(f); bytes.Contains(data, []byte("hello")) {
		t.Fatal("the last file was not shredded")
	}
}

func TestRandomSourceRunsOut(t *testing.T) {
	dir := t.TempDir()
	empty, f := filepath.Join(dir, "empty"), filepath.Join(dir, "f")
	write(t, empty, "")
	write(t, f, "x")
	s, errb := newEngine(shred.Options{Iterations: 1, RandomSource: empty})
	if err := s.Run([]string{f, f}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if want := "shredx: '" + empty + "': end of file\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
}

func TestProgress(t *testing.T) {
	f := filepath.Join(t.TempDir(), "f")
	write(t, f, strings.Repeat("x", 200*1024))
	s, errb := newEngine(shred.Options{Iterations: 1, Exact: true, Verbose: true})
	now := time.Unix(0, 0)
	s.Now = func() time.Time { now = now.Add(3 * time.Second); return now }
	if err := s.Run([]string{f}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(errb.String(), "...200KiB/200KiB 100%\n") {
		t.Fatalf("stderr = %q", errb)
	}
}

func TestMissingOperand(t *testing.T) {
	s, _ := newEngine(shred.Options{})
	var usage *core.UsageError
	if err := s.Run(nil); !errors.As(err, &usage) || usage.Msg != "missing file operand" {
		t.Fatalf("err = %v", err)
	}
}
