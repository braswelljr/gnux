package dd_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/dd"
)

// clock returns times one tenth of a millisecond apart, so the statistics
// line is deterministic.
func clock() func() time.Time {
	t := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return func() time.Time {
		now := t
		t = t.Add(100 * time.Microsecond)
		return now
	}
}

func run(stdin string, args ...string) (stdout, stderr string, err error) {
	return runIn(strings.NewReader(stdin), args...)
}

func runIn(in io.Reader, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	d := dd.New(dd.Options{}, core.Streams{In: in, Out: &out, Err: &errb})
	d.Now = clock()
	err = d.Run(args)
	return out.String(), errb.String(), err
}

func TestCopyAndSummary(t *testing.T) {
	input := strings.Repeat("x", 512)
	out, errs, err := run(input)
	if err != nil || out != input {
		t.Fatalf("copy: err %v, %d bytes out", err, len(out))
	}
	want := "1+0 records in\n1+0 records out\n512 bytes copied, 0.0001 s, 5.1 MB/s\n"
	if errs != want {
		t.Fatalf("summary = %q, want %q", errs, want)
	}
}

func TestSummaryUnits(t *testing.T) {
	tests := []struct {
		size int
		want string
	}{
		{1, "0+1 records in\n0+1 records out\n1 byte copied, 0.0001 s, 10 kB/s\n"},
		{1000, "1+1 records in\n1+1 records out\n1000 bytes (1.0 kB) copied, 0.0001 s, 10 MB/s\n"},
		{1 << 20, "2048+0 records in\n2048+0 records out\n1048576 bytes (1.0 MB, 1.0 MiB) copied, 0.0001 s, 10 GB/s\n"},
	}
	for _, tt := range tests {
		_, errs, err := run(strings.Repeat("a", tt.size))
		if err != nil || errs != tt.want {
			t.Errorf("size %d: stderr %q (err %v), want %q", tt.size, errs, err, tt.want)
		}
	}
}

func TestStatus(t *testing.T) {
	if _, errs, _ := run("abc", "status=none"); errs != "" {
		t.Errorf("status=none printed %q", errs)
	}
	if _, errs, _ := run("abc", "status=noxfer"); errs != "0+1 records in\n0+1 records out\n" {
		t.Errorf("status=noxfer printed %q", errs)
	}
}

func TestBlocksAndCount(t *testing.T) {
	tests := []struct {
		name, in string
		args     []string
		out      string
		records  string
	}{
		{"bs and count", "abcdefghij", []string{"bs=3", "count=2"}, "abcdef", "2+0 records in\n2+0 records out\n"},
		{"partial last block", "abcdefg", []string{"bs=3"}, "abcdefg", "2+1 records in\n2+1 records out\n"},
		{"skip blocks", "abcdefghij", []string{"bs=2", "skip=2"}, "efghij", "3+0 records in\n3+0 records out\n"},
		{"skip bytes", "abcdefghij", []string{"bs=4", "skip=3", "iflag=skip_bytes"}, "defghij", "1+1 records in\n1+1 records out\n"},
		{"count bytes", "abcdefghij", []string{"bs=4", "count=6", "iflag=count_bytes"}, "abcdef", "1+1 records in\n1+1 records out\n"},
		{"ibs and obs reblock", "abcdefghij", []string{"ibs=2", "obs=4"}, "abcdefghij", "5+0 records in\n2+1 records out\n"},
		{"count=0 copies nothing", "abc", []string{"count=0"}, "", "0+0 records in\n0+0 records out\n"},
		{"sync pads with NUL", "abc", []string{"bs=5", "conv=sync"}, "abc\x00\x00", "0+1 records in\n1+0 records out\n"},
		{"ucase", "Hello", []string{"conv=ucase"}, "HELLO", "0+1 records in\n0+1 records out\n"},
		{"lcase", "Hello", []string{"conv=lcase"}, "hello", "0+1 records in\n0+1 records out\n"},
		{"swab", "abcde", []string{"conv=swab"}, "badce", "0+1 records in\n0+1 records out\n"},
		{"block", "ab\nabcdef\n", []string{"cbs=4", "conv=block"}, "ab  abcd", "0+1 records in\n0+1 records out\n1 truncated record\n"},
		{"block unterminated last line", "ab", []string{"cbs=4", "conv=block"}, "ab  ", "0+1 records in\n0+1 records out\n"},
		{"unblock", "ab  cd  ef", []string{"cbs=4", "conv=unblock"}, "ab\ncd\nef\n", "0+1 records in\n0+1 records out\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errs, err := run(tt.in, append(tt.args, "status=noxfer")...)
			if err != nil {
				t.Fatalf("err %v (stderr %q)", err, errs)
			}
			if out != tt.out {
				t.Errorf("stdout = %q, want %q", out, tt.out)
			}
			if errs != tt.records {
				t.Errorf("stderr = %q, want %q", errs, tt.records)
			}
		})
	}
}

// chunked returns at most n bytes per Read, like a pipe.
type chunked struct {
	r io.Reader
	n int
}

func (c chunked) Read(p []byte) (int, error) {
	if len(p) > c.n {
		p = p[:c.n]
	}
	return c.r.Read(p)
}

func TestFullBlock(t *testing.T) {
	in := chunked{strings.NewReader("abcdefgh"), 3}
	_, errs, _ := runIn(in, "bs=4", "status=noxfer")
	if errs != "0+3 records in\n0+3 records out\n" {
		t.Errorf("short reads: %q", errs)
	}
	in = chunked{strings.NewReader("abcdefgh"), 3}
	out, errs, _ := runIn(in, "bs=4", "iflag=fullblock", "status=noxfer")
	if out != "abcdefgh" || errs != "2+0 records in\n2+0 records out\n" {
		t.Errorf("fullblock: %q %q", out, errs)
	}
}

func TestFiles(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	if err := os.WriteFile(src, []byte("0123456789"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("abcdefghijklmnop"), 0o644); err != nil {
		t.Fatal(err)
	}

	// The output is truncated at the seek offset.
	if _, errs, err := run("", "if="+src, "of="+dst, "bs=2", "skip=1", "seek=2", "status=none"); err != nil {
		t.Fatalf("err %v %q", err, errs)
	}
	if got, _ := os.ReadFile(dst); string(got) != "abcd23456789" {
		t.Fatalf("seek + truncate: %q", got)
	}

	// conv=notrunc overwrites in place.
	if _, _, err := run("XY", "of="+dst, "conv=notrunc", "status=none"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dst); string(got) != "XYcd23456789" {
		t.Fatalf("notrunc: %q", got)
	}

	// oflag=append adds to the end.
	if _, _, err := run("!", "of="+dst, "oflag=append", "conv=notrunc", "status=none"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dst); string(got) != "XYcd23456789!" {
		t.Fatalf("append: %q", got)
	}

	// seek with oflag=seek_bytes.
	if _, _, err := run("Z", "of="+dst, "seek=1", "oflag=seek_bytes", "conv=notrunc", "status=none"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dst); string(got) != "XZcd23456789!" {
		t.Fatalf("seek_bytes: %q", got)
	}

	// conv=excl refuses an existing file; conv=nocreat refuses a missing one.
	_, errs, err := run("", "of="+dst, "conv=excl")
	if !errors.Is(err, core.ErrFailed) || !strings.HasPrefix(errs, "ddx: failed to open '"+dst+"': File exists") {
		t.Fatalf("excl: %v %q", err, errs)
	}
	missing := filepath.Join(dir, "missing")
	_, errs, err = run("", "of="+missing, "conv=nocreat")
	if !errors.Is(err, core.ErrFailed) || errs != "ddx: failed to open '"+missing+"': No such file or directory\n" {
		t.Fatalf("nocreat: %v %q", err, errs)
	}
	_, errs, err = run("", "if="+missing)
	if !errors.Is(err, core.ErrFailed) || errs != "ddx: failed to open '"+missing+"': No such file or directory\n" {
		t.Fatalf("missing input: %v %q", err, errs)
	}

	// fsync on a real file.
	if _, _, err := run("data", "of="+filepath.Join(dir, "synced"), "conv=fsync", "status=none"); err != nil {
		t.Fatalf("fsync: %v", err)
	}
}

func TestSeekOnStdoutWritesNULs(t *testing.T) {
	out, _, err := run("ab", "bs=2", "seek=1", "status=none")
	if err != nil || out != "\x00\x00ab" {
		t.Fatalf("stdout seek: %q %v", out, err)
	}
}

func TestSkipPastEnd(t *testing.T) {
	out, errs, err := run("abc", "bs=2", "skip=5", "status=noxfer")
	if err != nil || out != "" {
		t.Fatalf("out %q err %v", out, err)
	}
	if want := "ddx: 'standard input': cannot skip to specified offset\n0+0 records in\n0+0 records out\n"; errs != want {
		t.Fatalf("stderr %q, want %q", errs, want)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("input/output error") }

func TestReadError(t *testing.T) {
	_, errs, err := runIn(failingReader{}, "status=noxfer")
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if want := "ddx: error reading 'standard input': Input/output error\n0+0 records in\n0+0 records out\n"; errs != want {
		t.Fatalf("stderr %q, want %q", errs, want)
	}
}

type closedPipe struct{}

func (closedPipe) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestClosedPipeIsQuiet(t *testing.T) {
	var errb bytes.Buffer
	d := dd.New(dd.Options{}, core.Streams{In: strings.NewReader("abc"), Out: closedPipe{}, Err: &errb})
	if err := d.Run(nil); err != nil || errb.Len() != 0 {
		t.Fatalf("err %v stderr %q", err, errb.String())
	}
}

func TestBadOperands(t *testing.T) {
	_, errs, err := run("", "bs=12q")
	if !errors.Is(err, core.ErrFailed) || errs != "ddx: invalid number: '12q'\n" {
		t.Fatalf("bad number: %v %q", err, errs)
	}
	_, _, err = run("", "foo")
	var usage *core.UsageError
	if !errors.As(err, &usage) || usage.Msg != "unrecognized operand 'foo'" {
		t.Fatalf("unknown operand: %v", err)
	}
}

func TestLibraryOptions(t *testing.T) {
	var out bytes.Buffer
	d := dd.New(dd.Options{BlockSize: 2, Count: 1, HasCount: true, Status: dd.StatusNone},
		core.Streams{In: strings.NewReader("abcdef"), Out: &out, Err: io.Discard})
	if err := d.Run(nil); err != nil || out.String() != "ab" {
		t.Fatalf("library: %q %v", out.String(), err)
	}
}
