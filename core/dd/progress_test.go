package dd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/dd"
)

// slowClock returns times 600 ms apart, so a short copy spans seconds.
func slowClock() func() time.Time {
	t := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return func() time.Time {
		now := t
		t = t.Add(600 * time.Millisecond)
		return now
	}
}

func TestProgress(t *testing.T) {
	var out, errb bytes.Buffer
	d := dd.New(dd.Options{}, core.Streams{In: strings.NewReader(strings.Repeat("x", 2048)), Out: &out, Err: &errb})
	d.Now = slowClock()
	if err := d.Run([]string{"status=progress"}); err != nil {
		t.Fatal(err)
	}
	// Checked before each block: at 1.2 s, 2.4 s and 3.0 s (the read that
	// meets the end), each line over the last, then a newline and GNU's
	// statistics.
	want := "\r512 bytes copied, 1 s, 427 B/s" +
		"\r1536 bytes (1.5 kB, 1.5 KiB) copied, 2 s, 640 B/s" +
		"\r2048 bytes (2.0 kB, 2.0 KiB) copied, 3 s, 683 B/s\n" +
		"4+0 records in\n4+0 records out\n2048 bytes (2.0 kB, 2.0 KiB) copied, 3.6 s, 569 B/s\n"
	if errb.String() != want {
		t.Fatalf("got  %q\nwant %q", errb.String(), want)
	}
}

func TestProgressPads(t *testing.T) {
	// A shorter line pads over the end of the longer one before it.
	var errb bytes.Buffer
	in := strings.NewReader(strings.Repeat("x", 1024*1024))
	d := dd.New(dd.Options{}, core.Streams{In: in, Out: &bytes.Buffer{}, Err: &errb})
	d.Now = slowClock()
	if err := d.Run([]string{"status=progress", "bs=1000000"}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.SplitN(errb.String(), "\n", 2)[0], "\r")[1:]
	for i := 1; i < len(lines); i++ {
		if len(lines[i]) < len(lines[i-1]) {
			t.Fatalf("line %d is shorter than the one before: %q", i, lines)
		}
	}
}

func TestProgressView(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "disk.img")
	if err := os.WriteFile(src, bytes.Repeat([]byte("x"), 2048), 0o644); err != nil {
		t.Fatal(err)
	}
	var errb bytes.Buffer
	d := dd.New(dd.Options{}, core.Streams{In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &errb})
	d.Now = slowClock()
	d.Getenv = func(k string) string { return map[string]string{"LANG": "C", "COLUMNS": "120"}[k] }
	opts, err := dd.Parse([]string{"if=" + src, "of=" + filepath.Join(dir, "copy.img"), "status=progress"})
	if err != nil {
		t.Fatal(err)
	}
	opts.Pretty = true
	d.Options = opts
	if err := d.Run(nil); err != nil {
		t.Fatal(err)
	}
	got := errb.String()
	for _, want := range []string{
		"\r........................   0%  0 B of 2.0 kB  0 B/s  -:-- left",
		"\r######.................. ",
		// The last state, then GNU's statistics on a line of their own.
		"\r######################## 100%  2.0 kB of 2.0 kB  ",
		" 0:00 left\n4+0 records in\n4+0 records out\n2048 bytes",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%q", want, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Errorf("colored without Color: %q", got)
	}
}
