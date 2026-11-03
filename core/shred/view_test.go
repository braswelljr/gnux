package shred_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core/shred"
)

func TestPrettyVerbose(t *testing.T) {
	f := filepath.Join(t.TempDir(), "disk.img")
	write(t, f, strings.Repeat("x", 200*1024))
	s, errb := newEngine(shred.Options{Iterations: 2, Exact: true, Verbose: true, Remove: shred.Unlink, Pretty: true})
	clock := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { clock = clock.Add(300 * time.Millisecond); return clock }
	s.Getenv = func(k string) string { return map[string]string{"LANG": "C", "COLUMNS": "200"}[k] }
	if err := s.Run([]string{f}); err != nil {
		t.Fatal(err)
	}
	got := errb.String()
	head := "\rshredx: " + f + ": pass 1/2 (random) "
	for _, want := range []string{
		head + "....................   0%  0B/200KiB",
		head + "######..............  32%  64KiB/200KiB",
		// Each pass ends on 100% and a line of its own; the reports after
		// it are GNU's lines.
		head + "####################" + " 100%  200KiB/200KiB\n\rshredx: " + f + ": pass 2/2 (random) ",
		"100%  200KiB/200KiB\nshredx: " + f + ": removing\nshredx: " + f + ": removed\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%q", want, got)
		}
	}
	if strings.Contains(got, "...\n") || strings.Contains(got, "\x1b[") {
		t.Errorf("GNU's lines or colors in the uncolored view: %q", got)
	}
}
