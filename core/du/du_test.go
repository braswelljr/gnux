package du_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/du"
)

func run(t *testing.T, opts du.Options, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	var out, errb bytes.Buffer
	d := du.New(opts, core.Streams{In: strings.NewReader(""), Out: &out, Err: &errb})
	d.Getenv = func(string) string { return "" }
	d.IsTerminal = func(any) bool { return false }
	d.Location = time.UTC
	err = d.Run(args)
	return out.String(), errb.String(), err
}

// tree makes root/{a (1000 bytes), sub/{b (3000 bytes), deep/c (5 bytes)}}.
func tree(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "root")
	for _, d := range []string{"sub/deep"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for name, size := range map[string]int{"a": 1000, "sub/b": 3000, "sub/deep/c": 5} {
		if err := os.WriteFile(filepath.Join(root, name), bytes.Repeat([]byte("x"), size), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func lines(s string) []string { return strings.Split(strings.TrimSuffix(s, "\n"), "\n") }

func p(parts ...string) string { return strings.Join(parts, string(os.PathSeparator)) }

func TestApparentSizes(t *testing.T) {
	root := tree(t)
	out, errb, err := run(t, du.Options{Apparent: true, BlockSize: "1"}, root)
	if err != nil {
		t.Fatalf("%v %q", err, errb)
	}
	// Directories count no apparent size of their own, as in GNU.
	want := []string{"5\t" + p(root, "sub", "deep"), "3005\t" + p(root, "sub"), "4005\t" + root}
	if got := lines(out); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAllDepthSummarize(t *testing.T) {
	root := tree(t)
	b := du.Options{Apparent: true, BlockSize: "1"}

	o := b
	o.All = true
	out, _, _ := run(t, o, root)
	want := []string{"1000\t" + p(root, "a"), "3000\t" + p(root, "sub", "b"), "5\t" + p(root, "sub", "deep", "c"),
		"5\t" + p(root, "sub", "deep"), "3005\t" + p(root, "sub"), "4005\t" + root}
	if got := lines(out); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("-a: got %q, want %q", got, want)
	}

	o = b
	o.MaxDepth, o.MaxDepthSet = 1, true
	out, _, _ = run(t, o, root)
	if got := lines(out); len(got) != 2 || got[0] != "3005\t"+p(root, "sub") {
		t.Fatalf("-d 1: %q", got)
	}

	o = b
	o.Summarize, o.Total = true, true
	out, _, _ = run(t, o, root, p(root, "a"))
	// The file under root was already counted: with several operands
	// everything is counted once, so it is not listed again.
	if out != "4005\t"+root+"\n4005\ttotal\n" {
		t.Fatalf("-sc: %q", out)
	}
}

func TestSeparateDirsAndThreshold(t *testing.T) {
	root := tree(t)
	out, _, _ := run(t, du.Options{Apparent: true, BlockSize: "1", SeparateDirs: true}, root)
	if got := lines(out); got[2] != "1000\t"+root || got[1] != "3000\t"+p(root, "sub") {
		t.Fatalf("-S: %q", got)
	}
	out, _, _ = run(t, du.Options{Apparent: true, BlockSize: "1", Threshold: 3005}, root)
	if got := lines(out); len(got) != 2 {
		t.Fatalf("-t 3005: %q", got)
	}
	out, _, _ = run(t, du.Options{Apparent: true, BlockSize: "1", Threshold: -5}, root)
	if got := lines(out); len(got) != 1 || got[0] != "5\t"+p(root, "sub", "deep") {
		t.Fatalf("-t -5: %q", got)
	}
}

func TestUnits(t *testing.T) {
	root := tree(t)
	tests := []struct {
		block string
		want  string
	}{
		{"1K", "4\t"},
		{"human-readable", "4.0K\t"},
		{"si", "4.1k\t"},
		{"K", "4K\t"},
		{"1", "4005\t"},
	}
	for _, tt := range tests {
		out, _, _ := run(t, du.Options{Apparent: true, Summarize: true, BlockSize: tt.block}, root)
		if !strings.HasPrefix(out, tt.want) {
			t.Errorf("-B %s: %q, want prefix %q", tt.block, out, tt.want)
		}
	}
	// Disk usage is counted in allocated blocks, so it is at least the size.
	out, _, _ := run(t, du.Options{Summarize: true, BlockSize: "1"}, root)
	if n := strings.Fields(out)[0]; len(n) < 4 {
		t.Fatalf("disk usage: %q", out)
	}
}

func TestInodesExcludeNull(t *testing.T) {
	root := tree(t)
	out, _, _ := run(t, du.Options{Inodes: true, Summarize: true}, root)
	if out != "6\t"+root+"\n" {
		t.Fatalf("--inodes: %q", out)
	}
	out, _, _ = run(t, du.Options{Apparent: true, BlockSize: "1", Summarize: true, Exclude: []string{"deep"}}, root)
	if out != "4000\t"+root+"\n" {
		t.Fatalf("--exclude: %q", out)
	}
	out, _, _ = run(t, du.Options{Apparent: true, BlockSize: "1", Summarize: true, Exclude: []string{"*/b"}}, root)
	if out != "1005\t"+root+"\n" {
		t.Fatalf("--exclude */b: %q", out)
	}
	out, _, _ = run(t, du.Options{Apparent: true, BlockSize: "1", Summarize: true, Null: true}, root)
	if out != "4005\t"+root+"\x00" {
		t.Fatalf("-0: %q", out)
	}
}

func TestHardLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no inode numbers")
	}
	root := tree(t)
	if err := os.Link(filepath.Join(root, "a"), filepath.Join(root, "sub", "a2")); err != nil {
		t.Skip("hard links unavailable")
	}
	out, _, _ := run(t, du.Options{Apparent: true, BlockSize: "1", Summarize: true}, root)
	if out != "4005\t"+root+"\n" {
		t.Fatalf("counted once: %q", out)
	}
	out, _, _ = run(t, du.Options{Apparent: true, BlockSize: "1", Summarize: true, CountLinks: true}, root)
	if out != "5005\t"+root+"\n" {
		t.Fatalf("-l: %q", out)
	}
	// The same directory twice is counted once.
	out, _, _ = run(t, du.Options{Apparent: true, BlockSize: "1", Summarize: true}, root, root)
	if out != "4005\t"+root+"\n" {
		t.Fatalf("repeated operand: %q", out)
	}
}

func TestTime(t *testing.T) {
	root := tree(t)
	when := time.Date(2030, 1, 2, 3, 4, 0, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(root, "sub", "b"), when, when); err != nil {
		t.Fatal(err)
	}
	out, _, _ := run(t, du.Options{Apparent: true, BlockSize: "1", Summarize: true, Time: "mtime"}, root)
	if out != "4005\t2030-01-02 03:04\t"+root+"\n" {
		t.Fatalf("--time: %q", out)
	}
	out, _, _ = run(t, du.Options{Apparent: true, BlockSize: "1", Summarize: true, Time: "mtime", TimeStyle: "+%Y"}, root)
	if out != "4005\t2030\t"+root+"\n" {
		t.Fatalf("--time-style: %q", out)
	}
}

func TestErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	_, errb, err := run(t, du.Options{}, missing)
	if !errors.Is(err, core.ErrFailed) || errb != "dux: cannot access '"+missing+"': No such file or directory\n" {
		t.Fatalf("missing: %v %q", err, errb)
	}
	var usage *core.UsageError
	for _, tt := range []struct {
		opts du.Options
		msg  string
	}{
		{du.Options{All: true, Summarize: true}, "cannot both summarize and show all entries"},
		{du.Options{Summarize: true, MaxDepth: 2, MaxDepthSet: true}, "warning: summarizing conflicts with --max-depth=2"},
		{du.Options{BlockSize: "x"}, "invalid --block-size argument 'x'"},
		{du.Options{Time: "btime"}, "invalid argument 'btime' for '--time'"},
	} {
		_, _, err := run(t, tt.opts, ".")
		if !errors.As(err, &usage) || usage.Msg != tt.msg {
			t.Errorf("%+v: %v, want %q", tt.opts, err, tt.msg)
		}
	}
}

func TestFiles0From(t *testing.T) {
	root := tree(t)
	list := filepath.Join(t.TempDir(), "list")
	if err := os.WriteFile(list, []byte(filepath.Join(root, "a")+"\x00"+filepath.Join(root, "sub", "b")+"\x00"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, err := run(t, du.Options{Apparent: true, BlockSize: "1", Files0From: list})
	if err != nil || out != "1000\t"+p(root, "a")+"\n3000\t"+p(root, "sub", "b")+"\n" {
		t.Fatalf("--files0-from: %q %v", out, err)
	}
}

func TestPretty(t *testing.T) {
	root := tree(t)
	out, errb, err := run(t, du.Options{Pretty: true, Apparent: true, BlockSize: "1"}, root)
	if err != nil {
		t.Fatalf("%v %q", err, errb)
	}
	got := lines(out)
	if len(got) != 3 {
		t.Fatalf("want one line per directory:\n%s", out)
	}
	// Sizes right-aligned, then each line's share of the operand.
	for i, want := range []string{"   5  ", "3005  ", "4005  "} {
		if !strings.HasPrefix(got[i], want) {
			t.Errorf("line %d = %q, want it to start %q", i, got[i], want)
		}
	}
	if !strings.Contains(got[2], " 100%  "+root) || !strings.Contains(got[1], "  75%  ") {
		t.Errorf("percentages:\n%s", out)
	}

	// -0 keeps GNU's format, which a script reads.
	out, _, _ = run(t, du.Options{Pretty: true, Null: true, Summarize: true, Apparent: true, BlockSize: "1"}, root)
	if out != "4005\t"+root+"\x00" {
		t.Errorf("-0 with the view: %q", out)
	}
}
