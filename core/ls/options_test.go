package ls_test

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/ls"
)

func TestQuotingStyles(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "a b", "plain", "it's")
	tests := []struct {
		quoting ls.Quoting
		want    string
	}{
		{ls.QuoteLiteral, "a b\nit's\nplain\n"},
		{ls.QuoteShell, "'a b'\n\"it's\"\nplain\n"},
		{ls.QuoteShellAlways, "'a b'\n\"it's\"\n'plain'\n"},
		{ls.QuoteShellEscape, "'a b'\n\"it's\"\nplain\n"},
		{ls.QuoteC, "\"a b\"\n\"it's\"\n\"plain\"\n"},
		{ls.QuoteCMaybe, "a b\nit's\nplain\n"},
		{ls.QuoteEscape, "a\\ b\nit's\nplain\n"},
		{ls.QuoteLocale, "'a b'\n'it\\'s'\n'plain'\n"},
		{ls.QuoteCLocale, "\"a b\"\n\"it's\"\n\"plain\"\n"},
	}
	for _, tt := range tests {
		if got := list(t, ls.Options{Quoting: tt.quoting}, dir); got != tt.want {
			t.Errorf("%v: got %q, want %q", tt.quoting, got, tt.want)
		}
	}
}

func TestQuotingEnvironment(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "a b")
	run := func(opts ls.Options, env string) (string, string) {
		l, out, errb := newEngine(opts)
		l.Getenv = func(k string) string {
			if k == "QUOTING_STYLE" {
				return env
			}
			return ""
		}
		if err := l.Run([]string{dir}); err != nil {
			t.Fatal(err)
		}
		return out.String(), errb.String()
	}
	if out, _ := run(ls.Options{}, "shell"); out != "'a b'\n" {
		t.Errorf("QUOTING_STYLE=shell: %q", out)
	}
	if out, _ := run(ls.Options{Quoting: ls.QuoteLiteral}, "shell"); out != "a b\n" {
		t.Errorf("-N over QUOTING_STYLE: %q", out)
	}
	if out, _ := run(ls.Options{QuotingDefault: ls.QuoteEscape}, ""); out != "a\\ b\n" {
		t.Errorf("dir's default: %q", out)
	}
	out, errs := run(ls.Options{}, "bogus")
	if out != "a b\n" || errs != "lsx: ignoring invalid value of environment variable QUOTING_STYLE: 'bogus'\n" {
		t.Errorf("invalid QUOTING_STYLE: %q, %q", out, errs)
	}
}

func TestQuotedNamesAlign(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "a b", "c")
	got := list(t, ls.Options{Format: ls.FormatColumns, Width: 80, Quoting: ls.QuoteShellEscape}, dir)
	if got != "'a b'   c\n" {
		t.Errorf("got %q", got)
	}
	// Nothing quoted, nothing padded.
	touch(t, filepath.Join(dir, "sub"), "x", "y")
	if got := list(t, ls.Options{Format: ls.FormatColumns, Width: 80, Quoting: ls.QuoteShellEscape}, filepath.Join(dir, "sub")); got != "x  y\n" {
		t.Errorf("got %q", got)
	}
}

func TestZero(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "a", "b")
	if got := list(t, ls.Options{Zero: true, Format: ls.FormatSingle}, dir); got != "a\x00b\x00" {
		t.Errorf("got %q", got)
	}
	l, _, errb := newEngine(ls.Options{Zero: true, Dired: true, Format: ls.FormatLong})
	var exit *core.ExitError
	if err := l.Run([]string{dir}); !errors.As(err, &exit) || exit.Code != 2 || errb.String() != "lsx: --dired and --zero are incompatible\n" {
		t.Errorf("err = %v, stderr = %q", err, errb.String())
	}
}

func TestDired(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "ab")
	got := list(t, ls.Options{Dired: true, Format: ls.FormatLong, NumericIDs: true, TimeStyle: "+T"}, dir)
	lines := strings.Split(got, "\n")
	if len(lines) != 5 || !strings.HasPrefix(lines[0], "  total ") || !strings.HasPrefix(lines[1], "  -rw") {
		t.Fatalf("got %q", got)
	}
	// The recorded positions frame the name.
	var start, end int
	if _, err := fmtSscan(lines[2], &start, &end); err != nil || got[start:end] != "ab" {
		t.Fatalf("//DIRED// line %q does not frame the name in %q", lines[2], got)
	}
	if lines[3] != "//DIRED-OPTIONS// --quoting-style=literal" {
		t.Errorf("options line = %q", lines[3])
	}
	// Without the long format, --dired does nothing.
	if got := list(t, ls.Options{Dired: true, Format: ls.FormatSingle}, dir); got != "ab\n" {
		t.Errorf("got %q", got)
	}
}

func TestBlockSizes(t *testing.T) {
	dir := t.TempDir()
	sized(t, filepath.Join(dir, "f"), 3000)
	long := func(o ls.Options) string {
		o.Format, o.NumericIDs, o.TimeStyle, o.NoOwner, o.NoGroup = ls.FormatLong, true, "+", true, true
		out := list(t, o, filepath.Join(dir, "f"))
		return strings.Fields(out)[2]
	}
	for spec, want := range map[string]string{"": "3000", "1": "3000", "K": "3K", "1K": "3", "KB": "3kB", "human-readable": "3.0K", "si": "3.0k"} {
		if got := long(ls.Options{BlockSize: spec}); got != want {
			t.Errorf("--block-size=%q: size %q, want %q", spec, got, want)
		}
	}
	l, _, errb := newEngine(ls.Options{BlockSize: "x"})
	var exit *core.ExitError
	if err := l.Run([]string{dir}); !errors.As(err, &exit) || exit.Code != 2 || errb.String() != "lsx: invalid --block-size argument 'x'\n" {
		t.Errorf("err = %v, stderr = %q", err, errb.String())
	}
}

func TestAuthor(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "f")
	got := list(t, ls.Options{Format: ls.FormatLong, Author: true, NumericIDs: true, TimeStyle: "+T"}, filepath.Join(dir, "f"))
	fields := strings.Fields(got)
	// mode, links, owner, group, author, size, time, name
	if len(fields) != 8 || fields[4] != fields[2] {
		t.Errorf("got %q", got)
	}
}

func TestHyperlink(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "a b")
	name := filepath.Join(dir, "a b")
	l, out, _ := newEngine(ls.Options{Hyperlink: ls.WhenAlways})
	l.Hostname = func() (string, error) { return "host", nil }
	if err := l.Run([]string{name}); err != nil {
		t.Fatal(err)
	}
	// An OSC 8 link to the percent-encoded absolute path, around the name.
	got := out.String()
	if !strings.HasPrefix(got, "\x1b]8;;file://host/") || !strings.HasSuffix(got, "/a%20b\a"+name+"\x1b]8;;\a\n") {
		t.Errorf("got %q", got)
	}
	if strings.Contains(got[:strings.Index(got, "\a")], " ") {
		t.Errorf("the URL is not encoded: %q", got)
	}
}

// fmtSscan reads the two numbers of a "//DIRED// START END" line.
func fmtSscan(line string, start, end *int) (int, error) {
	return fmt.Sscanf(line, "//DIRED// %d %d", start, end)
}
