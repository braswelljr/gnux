package pr_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/pr"
)

func run(opts pr.Options, stdin string, args ...string) (string, string, error) {
	var out, errb bytes.Buffer
	p := pr.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb})
	p.Now = func() time.Time { return time.Date(2026, 10, 20, 9, 5, 0, 0, time.UTC) }
	p.Location = time.UTC
	err := p.Run(args)
	return out.String(), errb.String(), err
}

func lines(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		b.WriteString(strings.Repeat("x", i%7+1))
		b.WriteByte('\n')
	}
	return b.String()
}

func TestOmitHeaderPassesLinesThrough(t *testing.T) {
	out, _, err := run(pr.Options{OmitHeader: true}, "a\nb\nc\n")
	if err != nil || out != "a\nb\nc\n" {
		t.Fatalf("%v %q", err, out)
	}
}

func TestPage(t *testing.T) {
	out, _, err := run(pr.Options{Length: 15}, "hello\n")
	if err != nil {
		t.Fatal(err)
	}
	header := "\n\n2026-10-20 09:05" + strings.Repeat(" ", 25) + strings.Repeat(" ", 25) + "Page 1\n\n\n"
	if want := header + "hello\n" + strings.Repeat("\n", 9); out != want {
		t.Fatalf("page =\n%q\nwant\n%q", out, want)
	}
}

func TestHeaderTitle(t *testing.T) {
	out, _, _ := run(pr.Options{Length: 11, Header: "T", HeaderSet: true, FormFeed: true}, "x\n")
	// 72 columns: 16 for the date, 1 for the title, 6 for "Page 1": 49 to share.
	want := "\n\n2026-10-20 09:05" + strings.Repeat(" ", 24) + "T" + strings.Repeat(" ", 25) + "Page 1\n\n\nx\n\f"
	if out != want {
		t.Fatalf("page =\n%q\nwant\n%q", out, want)
	}
}

func TestColumnsDown(t *testing.T) {
	out, _, err := run(pr.Options{Columns: 3, OmitHeader: true}, "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n")
	if err != nil {
		t.Fatal(err)
	}
	// 72 columns less two separators, in three: 23 each, joined by tabs.
	want := "1\t\t\t5\t\t\t8\n2\t\t\t6\t\t\t9\n3\t\t\t7\t\t\t10\n4\n"
	if out != want {
		t.Fatalf("got %q\nwant %q", out, want)
	}
}

func TestColumnsAcross(t *testing.T) {
	out, _, _ := run(pr.Options{Columns: 2, Across: true, OmitHeader: true, Width: 20}, "a\nb\nc\n")
	if want := "a\t  b\nc\n"; out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestSeparatorJoinsColumns(t *testing.T) {
	out, _, _ := run(pr.Options{Columns: 2, Across: true, OmitHeader: true, Separator: ",", UseSeparator: true, JoinLines: true}, "a\nb\nc\nd\n")
	if want := "a,b\nc,d\n"; out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestNumberLines(t *testing.T) {
	out, _, _ := run(pr.Options{NumberLines: true, OmitHeader: true}, "a\nb\n")
	if want := "    1\ta\n    2\tb\n"; out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
	out, _, _ = run(pr.Options{NumberLines: true, NumberSeparator: ':', NumberWidth: 3, FirstLineNumber: 9, FirstLineNumberSet: true, OmitHeader: true}, "a\nb\n")
	if want := "  9:a\n 10:b\n"; out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestMerge(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	if err := os.WriteFile(a, []byte("one\ntwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("uno\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, _, err := run(pr.Options{Merge: true, OmitHeader: true, Width: 20}, "", a, b)
	// The column of a file that has ended is still padded and separated, as
	// GNU's align_column does.
	if want := "one\t  uno\ntwo\t  \n"; err != nil || out != want {
		t.Fatalf("%v: got %q, want %q", err, out, want)
	}
}

func TestPageRange(t *testing.T) {
	out, _, _ := run(pr.Options{Length: 11, FirstPage: 2, LastPage: 2, OmitPagination: false}, "1\n2\n3\n")
	if !strings.Contains(out, "Page 2") || strings.Contains(out, "Page 1") || strings.Contains(out, "Page 3") ||
		!strings.Contains(out, "\n2\n") {
		t.Fatalf("page 2 only: %q", out)
	}
	_, errb, _ := run(pr.Options{Length: 11, FirstPage: 9}, "1\n")
	// The end of the input is found on the page after its last line, which
	// GNU counts.
	if errb != "prx: starting page number 9 exceeds page count 2\n" {
		t.Fatalf("past the end: %q", errb)
	}
}

func TestDoubleSpaceAndIndent(t *testing.T) {
	out, _, _ := run(pr.Options{DoubleSpace: true, Indent: 2, OmitHeader: true}, "a\nb\n")
	if want := "  a\n\n  b\n\n"; out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestControlCharacters(t *testing.T) {
	out, _, _ := run(pr.Options{ShowControlChars: true, OmitHeader: true}, "a\x07b\n")
	if out != "a^Gb\n" {
		t.Fatalf("-c: %q", out)
	}
	out, _, _ = run(pr.Options{ShowNonprinting: true, OmitHeader: true}, "a\x07b\n")
	if out != "a\\007b\n" {
		t.Fatalf("-v: %q", out)
	}
}

func TestFailures(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, []byte("ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, errb, err := run(pr.Options{OmitHeader: true}, "", missing, f)
	if !errors.Is(err, core.ErrFailed) || errb != "prx: "+missing+": No such file or directory\n" || out != "ok\n" {
		t.Fatalf("missing file: %v %q %q", err, errb, out)
	}
	_, errb, err = run(pr.Options{OmitHeader: true, NoFileWarnings: true}, "", missing)
	if !errors.Is(err, core.ErrFailed) || errb != "" {
		t.Fatalf("-r: %v %q", err, errb)
	}
	_, errb, err = run(pr.Options{Columns: 80}, "x\n")
	if !errors.Is(err, core.ErrFailed) || errb != "prx: page width too narrow\n" {
		t.Fatalf("narrow: %v %q", err, errb)
	}
	_, errb, err = run(pr.Options{OmitHeader: true}, "", dir)
	if !errors.Is(err, core.ErrFailed) || errb != "prx: "+dir+": Is a directory\n" {
		t.Fatalf("directory: %v %q", err, errb)
	}
}

func TestLongInputBalancesTheLastPage(t *testing.T) {
	out, _, err := run(pr.Options{Columns: 2, Length: 20}, lines(25))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(out, "Page "); got != 2 {
		t.Fatalf("%d pages, want 2:\n%s", got, out)
	}
}
