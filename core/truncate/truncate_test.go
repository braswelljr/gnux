package truncate_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/truncate"
)

func run(opts truncate.Options, args ...string) (string, error) {
	var out, errb bytes.Buffer
	err := truncate.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb}).Run(args)
	return errb.String(), err
}

func sized(t *testing.T, dir, name string, n int) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, bytes.Repeat([]byte("x"), n), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func size(t *testing.T, p string) int64 {
	t.Helper()
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return info.Size()
}

func TestSizes(t *testing.T) {
	tests := []struct {
		spec  string
		start int
		want  int64
	}{
		{"0", 100, 0},
		{"10", 100, 10},
		{"1K", 0, 1024},
		{"1KB", 0, 1000},
		{"1KiB", 0, 1024},
		{"2M", 0, 2 << 20},
		{"+10", 100, 110},
		{"-10", 100, 90},
		{"-1000", 100, 0},
		{"<50", 100, 50},
		{"<500", 100, 100},
		{">50", 100, 100},
		{">500", 100, 500},
		{"/64", 100, 64},
		{"%64", 100, 128},
		{"%64", 128, 128},
		{"%64", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			p := sized(t, t.TempDir(), "f", tt.start)
			if msg, err := run(truncate.Options{Size: tt.spec}, p); err != nil {
				t.Fatalf("Run: %v (%s)", err, msg)
			}
			if got := size(t, p); got != tt.want {
				t.Fatalf("size = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestExtendPadsWithZeros(t *testing.T) {
	p := sized(t, t.TempDir(), "f", 2)
	if _, err := run(truncate.Options{Size: "+3"}, p); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "xx\x00\x00\x00" {
		t.Fatalf("content = %q", b)
	}
}

func TestCreatesAndNoCreate(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "new")
	if _, err := run(truncate.Options{Size: "5"}, p); err != nil {
		t.Fatal(err)
	}
	if size(t, p) != 5 {
		t.Fatal("file not created at 5 bytes")
	}
	q := filepath.Join(dir, "absent")
	msg, err := run(truncate.Options{Size: "5", NoCreate: true}, q)
	if err != nil || msg != "" {
		t.Fatalf("-c on a missing file: err=%v msg=%q", err, msg)
	}
	if _, err := os.Stat(q); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("-c created the file")
	}
}

func TestReference(t *testing.T) {
	dir := t.TempDir()
	ref := sized(t, dir, "ref", 300)
	p := sized(t, dir, "f", 10)
	if _, err := run(truncate.Options{Reference: ref}, p); err != nil {
		t.Fatal(err)
	}
	if size(t, p) != 300 {
		t.Fatalf("size = %d, want 300", size(t, p))
	}
	if _, err := run(truncate.Options{Reference: ref, Size: "+5"}, p); err != nil {
		t.Fatal(err)
	}
	if size(t, p) != 305 {
		t.Fatalf("size = %d, want 305 (reference plus 5)", size(t, p))
	}

	msg, err := run(truncate.Options{Reference: filepath.Join(dir, "nope")}, p)
	if !errors.Is(err, core.ErrFailed) || !strings.Contains(msg, "truncatex: cannot stat '") || !strings.Contains(msg, "No such file or directory") {
		t.Fatalf("err=%v msg=%q", err, msg)
	}
}

func TestIOBlocks(t *testing.T) {
	p := sized(t, t.TempDir(), "f", 0)
	if _, err := run(truncate.Options{Size: "2", IOBlocks: true}, p); err != nil {
		t.Fatal(err)
	}
	if got := size(t, p); got <= 0 || got%2 != 0 {
		t.Fatalf("size = %d, want 2 blocks", got)
	}
}

func TestUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		opts truncate.Options
		args []string
		want string
	}{
		{"nothing", truncate.Options{}, []string{"f"}, "you must specify either '--size' or '--reference'"},
		{"io blocks alone", truncate.Options{IOBlocks: true, Reference: "r"}, []string{"f"}, "'--io-blocks' was specified but '--size' was not"},
		{"bad number", truncate.Options{Size: "abc"}, []string{"f"}, "Invalid number: 'abc'"},
		{"bad suffix", truncate.Options{Size: "1X"}, []string{"f"}, "Invalid number: '1X'"},
		{"absolute with reference", truncate.Options{Size: "5", Reference: "r"}, []string{"f"}, "you must specify a relative '--size' with '--reference'"},
		{"division by zero", truncate.Options{Size: "%0"}, []string{"f"}, "division by zero"},
		{"no file", truncate.Options{Size: "1"}, nil, "missing file operand"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := run(tt.opts, tt.args...)
			var u *core.UsageError
			if !errors.As(err, &u) || u.Msg != tt.want {
				t.Fatalf("err = %v, want usage error %q", err, tt.want)
			}
		})
	}
}

func TestFailingOperandDoesNotStopTheRest(t *testing.T) {
	dir := t.TempDir()
	good := sized(t, dir, "good", 10)
	msg, err := run(truncate.Options{Size: "3"}, dir, good)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if !strings.HasPrefix(msg, "truncatex: cannot open '"+dir+"' for writing: ") {
		t.Fatalf("msg = %q", msg)
	}
	if size(t, good) != 3 {
		t.Fatal("the second operand was not truncated")
	}
}
