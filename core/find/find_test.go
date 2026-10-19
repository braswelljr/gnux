package find_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/find"
)

// now is the clock every test runs at.
var now = time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

type result struct {
	out, err string
	e        error
}

// run drives the engine in the current directory with a fixed clock, UTC and
// no terminal. cfg, when not nil, adjusts the engine first.
func run(t *testing.T, cfg func(*find.Find), args ...string) result {
	t.Helper()
	return runIn(t, "", cfg, args...)
}

func runIn(t *testing.T, stdin string, cfg func(*find.Find), args ...string) result {
	t.Helper()
	var out, errb bytes.Buffer
	f := find.New(find.Options{}, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb})
	f.Now = func() time.Time { return now }
	f.Location = time.UTC
	f.IsTerminal = func(any) bool { return false }
	if cfg != nil {
		cfg(f)
	}
	err := f.Run(args)
	return result{out.String(), errb.String(), err}
}

// lines is the expected output: one path per line, written with slashes.
func lines(paths ...string) string {
	var b strings.Builder
	for _, p := range paths {
		b.WriteString(p)
		b.WriteByte('\n')
	}
	return b.String()
}

func write(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), size), 0o644); err != nil {
		t.Fatal(err)
	}
}

// tree builds the standard tree and makes it the working directory:
//
//	.hidden (3)  a.txt (5)  b.log (0)
//	dir/c.txt (2000)  dir/sub/d.TXT (1)
//	empty/
func tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, filepath.Join(root, ".hidden"), 3)
	write(t, filepath.Join(root, "a.txt"), 5)
	write(t, filepath.Join(root, "b.log"), 0)
	write(t, filepath.Join(root, "dir", "c.txt"), 2000)
	write(t, filepath.Join(root, "dir", "sub", "d.TXT"), 1)
	if err := os.Mkdir(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	return root
}

// check runs find and compares standard output, expecting success.
func check(t *testing.T, want string, args ...string) {
	t.Helper()
	r := run(t, nil, args...)
	if r.e != nil || r.err != "" {
		t.Fatalf("findx %q: err = %v, stderr = %q", args, r.e, r.err)
	}
	if r.out != want {
		t.Fatalf("findx %q:\n got %q\nwant %q", args, r.out, want)
	}
}

func TestDefaultPrint(t *testing.T) {
	tree(t)
	all := lines(".", "./.hidden", "./a.txt", "./b.log", "./dir", "./dir/c.txt", "./dir/sub", "./dir/sub/d.TXT", "./empty")
	check(t, all)
	check(t, all, ".")
	check(t, all, ".", "-print")
	check(t, all, "-P", "--", ".")
	check(t, lines("dir", "dir/c.txt", "dir/sub", "dir/sub/d.TXT", "a.txt"), "dir", "a.txt")
	// A trailing separator on the starting point is kept, not doubled.
	check(t, lines("dir/", "dir/c.txt", "dir/sub", "dir/sub/d.TXT"), "dir/")
}

func TestMissingStartingPoint(t *testing.T) {
	tree(t)
	r := run(t, nil, "nope", "a.txt", "gone")
	if !errors.Is(r.e, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", r.e)
	}
	if want := "findx: 'nope': No such file or directory\nfindx: 'gone': No such file or directory\n"; r.err != want {
		t.Fatalf("stderr = %q, want %q", r.err, want)
	}
	if r.out != "a.txt\n" {
		t.Fatalf("stdout = %q", r.out)
	}
}

func TestDepthOptions(t *testing.T) {
	tree(t)
	check(t, lines("./.hidden", "./a.txt", "./b.log", "./dir/c.txt", "./dir/sub/d.TXT", "./dir/sub", "./dir", "./empty", "."), "-depth")
	check(t, lines("dir/c.txt", "dir/sub/d.TXT", "dir/sub", "dir"), "dir", "-d")
	check(t, lines(".", "./.hidden", "./a.txt", "./b.log", "./dir", "./empty"), "-maxdepth", "1")
	check(t, lines("."), "-maxdepth", "0")
	check(t, lines("./dir/c.txt", "./dir/sub"), "-mindepth", "2", "-maxdepth", "2")
	check(t, lines("./dir/sub/d.TXT"), "-mindepth", "3")
}

func TestNameTests(t *testing.T) {
	tree(t)
	check(t, lines("./a.txt", "./dir/c.txt"), "-name", "*.txt")
	check(t, lines("./a.txt", "./dir/c.txt", "./dir/sub/d.TXT"), "-iname", "*.TXT")
	// A leading dot is matched by a wildcard, unlike in the shell.
	check(t, lines("./.hidden"), "-name", "*hid*")
	check(t, lines("./a.txt", "./b.log"), "-name", "[ab].???")
	check(t, lines("."), "-name", ".")
	check(t, lines("dir/"), "dir/", "-name", "dir")
	check(t, lines("./dir/c.txt", "./dir/sub", "./dir/sub/d.TXT"), "-path", "./dir/*")
	check(t, lines("./dir/sub/d.TXT"), "-wholename", "*sub/d*")
	check(t, lines("./dir/sub/d.TXT"), "-ipath", "*SUB/D.txt")
	check(t, lines("./dir/sub/d.TXT"), "-iwholename", "*SUB/D.txt")
	check(t, "", "-name", "dir/c.txt")
}

func TestRegex(t *testing.T) {
	tree(t)
	// The whole path must match, in Emacs syntax by default.
	check(t, lines("./a.txt", "./dir/c.txt"), "-regex", `.*\.txt`)
	check(t, "", "-regex", `a\.txt`)
	check(t, lines("./a.txt", "./b.log"), "-regex", `\./\(a\|b\)\..+`)
	check(t, lines("./a.txt", "./dir/c.txt", "./dir/sub/d.TXT"), "-iregex", `.*\.txt`)
	check(t, lines("./a.txt", "./b.log"), "-regextype", "posix-extended", "-regex", `\./(a|b)\.[a-z]{3}`)
	check(t, lines("./a.txt", "./b.log"), "-regextype", "egrep", "-regex", `\./(a|b)\..+`)
	check(t, lines("./a.txt", "./b.log"), "-regextype", "posix-basic", "-regex", `\./\(a\|b\)\.[a-z]\{3\}`)
	check(t, lines("./a.txt", "./b.log"), "-regextype", "sed", "-regex", `\./[ab]\..\+`)
	// In Emacs syntax braces and bare parentheses are literal.
	check(t, "", "-regex", `\./(a|b)\..*`)
}

func TestType(t *testing.T) {
	tree(t)
	check(t, lines(".", "./dir", "./dir/sub", "./empty"), "-type", "d")
	check(t, lines("./.hidden", "./a.txt", "./b.log", "./dir/c.txt", "./dir/sub/d.TXT"), "-type", "f")
	check(t, lines("dir", "dir/c.txt", "dir/sub", "dir/sub/d.TXT"), "dir", "-type", "f,d")
	check(t, lines("dir", "dir/sub"), "dir", "-type", "l,d")
	check(t, "", "-type", "l,p,s,b,c")
	check(t, lines(".", "./dir", "./dir/sub", "./empty"), "-xtype", "d")
}

func TestSize(t *testing.T) {
	tree(t)
	files := func(args ...string) []string { return append([]string{".", "-type", "f"}, args...) }
	check(t, lines("./a.txt"), files("-size", "5c")...)
	check(t, lines("./dir/c.txt"), files("-size", "+5c")...)
	check(t, lines("./.hidden", "./b.log", "./dir/sub/d.TXT"), files("-size", "-5c")...)
	// Sizes are rounded up to whole units: every non-empty file here is "1k"
	// except the 2000-byte one, which is 2k and four 512-byte blocks.
	check(t, lines("./.hidden", "./a.txt", "./dir/sub/d.TXT"), files("-size", "1k")...)
	check(t, lines("./dir/c.txt"), files("-size", "2k")...)
	check(t, lines("./b.log"), files("-size", "-1k")...)
	check(t, lines("./dir/c.txt"), files("-size", "4")...)
	check(t, lines("./dir/c.txt"), files("-size", "4b")...)
	check(t, lines("./dir/c.txt"), files("-size", "1000w")...)
	check(t, lines("./.hidden", "./a.txt", "./dir/c.txt", "./dir/sub/d.TXT"), files("-size", "1M")...)
	check(t, lines("./b.log"), files("-size", "0G")...)
}

func TestEmpty(t *testing.T) {
	tree(t)
	check(t, lines("./b.log", "./empty"), "-empty")
	check(t, lines("./empty"), "-empty", "-type", "d")
}

func TestOperators(t *testing.T) {
	tree(t)
	check(t, lines("./a.txt", "./b.log"), "-name", "a.txt", "-o", "-name", "b.log")
	check(t, lines("./a.txt", "./b.log"), "-name", "a.txt", "-or", "-name", "b.log")
	// -a binds tighter than -o: only the right branch prints.
	check(t, lines("./b.log"), "-name", "a.txt", "-o", "-name", "b.log", "-print")
	check(t, lines("./a.txt", "./b.log"), "(", "-name", "a.txt", "-o", "-name", "b.log", ")", "-print")
	check(t, lines("./a.txt"), "-type", "f", "-a", "-name", "a*")
	check(t, lines("./a.txt"), "-type", "f", "-and", "-name", "a*")
	check(t, lines(".", "./dir", "./dir/sub", "./empty"), "!", "-type", "f")
	check(t, lines(".", "./dir", "./dir/sub", "./empty"), "-not", "-type", "f")
	check(t, lines("./a.txt"), "!", "!", "-name", "a.txt")
	// ! binds tighter than the implicit and.
	check(t, lines("./dir/c.txt"), "!", "-name", "a.txt", "-name", "*.txt")
	// The comma evaluates both sides and yields the right one.
	check(t, lines("./a.txt"), "-false", ",", "-name", "a.txt")
	check(t, lines("./a.txt", "./a.txt"), "-maxdepth", "1", "-name", "a.txt", "-print", ",", "-name", "a.txt", "-print")
	check(t, lines("./a.txt", "./b.log"), "-maxdepth", "1", "(", "-name", "a.txt", "-print", "-o", "-true", ")", ",", "-name", "b.log", "-print")
	check(t, lines("./a.txt"), "-true", "-name", "a.txt")
	check(t, "", "-false")
	// Short circuit: the print after a false test never runs.
	check(t, "", "-false", "-print")
	check(t, lines("./a.txt", "./b.log", "./b.log"), "-maxdepth", "1", "-type", "f", "(", "-name", "a.txt", "-print", "-o", "-name", "b.log", "-print", "-print", ")")
}

func TestPrune(t *testing.T) {
	tree(t)
	check(t, lines(".", "./.hidden", "./a.txt", "./b.log", "./empty"), "-name", "dir", "-prune", "-o", "-print")
	// -prune alone still gets the implicit -print, around the whole expression.
	check(t, lines("./dir"), "-name", "dir", "-prune")
	check(t, lines("./dir", "./dir/c.txt"), "-name", "sub", "-prune", "-o", "-path", "./dir*", "-print")
	// With -depth the contents are visited before -prune can stop them.
	check(t, lines("./dir/c.txt", "./dir/sub/d.TXT", "./dir/sub"), "-depth", "-name", "dir", "-prune", "-o", "-path", "./dir/*", "-print")
}

func TestQuit(t *testing.T) {
	tree(t)
	check(t, lines(".", "./.hidden", "./a.txt"), "-print", "-name", "a.txt", "-quit")
	check(t, "", "-quit")
	check(t, lines("./a.txt"), ".", ".", "-name", "a.txt", "-print", "-quit", "-print")
}

func TestPrint0(t *testing.T) {
	tree(t)
	check(t, "./a.txt\x00./dir/c.txt\x00", "-name", "*.txt", "-print0")
}

func TestPrintf(t *testing.T) {
	tree(t)
	const sep = "/"
	tests := []struct {
		format, want string
	}{
		{`%p\n`, "dir" + sep + "sub" + sep + "d.TXT\n"},
		{`%f|%h|%H|%P\n`, "d.TXT|dir" + sep + "sub|dir|sub" + sep + "d.TXT\n"},
		{`%s %d %y %Y\n`, "1 2 f f\n"},
		{`[%5s][%-5s][%05s][%+d]\n`, "[    1][1    ][00001][+2]\n"},
		{`[%10f][%-10f][%.3f]\n`, "[     d.TXT][d.TXT     ][d.T]\n"},
		{`100%%\n`, "100%\n"},
		{`a\tb\\c\101\0\a\b\f\r\v\n`, "a\tb\\cA\x00\a\b\f\r\v\n"},
		{`one\ctwo`, "one"},
		{`%l|\n`, "|\n"},
		{`%n\n`, "1\n"},
		{`%T@ %TY-%Tm-%Td %TH:%TM:%TS %TT %TF %TD %Tj %Ta %Tb %TZ\n`,
			"1718366400.0000000000 2024-06-14 12:00:00.0000000000 12:00:00.0000000000 2024-06-14 06/14/24 166 Fri Jun UTC\n"},
		{`%T+|%Tc|%Tk|%Tl|%Tp|%Ty|%Te|%TB|%TA|%TR|%Ts|%TC\n`,
			"2024-06-14+12:00:00.0000000000|Fri Jun 14 12:00:00 2024|12|12|PM|24|14|June|Friday|12:00|1718366400|20\n"},
		{`%t\n`, "Fri Jun 14 12:00:00.0000000000 2024\n"},
		{`%A@\n`, "1718280000.0000000000\n"},
		{`%a\n`, "Thu Jun 13 12:00:00.0000000000 2024\n"},
		{`%AY%Am%Ad\n`, "20240613\n"},
		{`no newline`, "no newline"},
		{`%`, "%"},
	}
	target := filepath.Join("dir", "sub", "d.TXT")
	if err := os.Chtimes(target, now.Add(-48*time.Hour), now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		check(t, tt.want, "dir", "-name", "d.TXT", "-printf", tt.format)
	}
	check(t, "0 . .\n1 a.txt .\n", "-maxdepth", "1", "(", "-name", ".", "-o", "-name", "a.txt", ")", "-printf", `%d %f %h\n`)
	check(t, "d:|f:.hidden|f:a.txt|", "-maxdepth", "1", "!", "-name", "[bde]*", "-printf", "%y:%P|")
}

func TestPrintfWarnings(t *testing.T) {
	tree(t)
	r := run(t, nil, "a.txt", "-printf", `%q|\q|%Tq|\`)
	if r.e != nil {
		t.Fatalf("err = %v", r.e)
	}
	if want := `%q|\q|%Tq|\`; r.out != want {
		t.Fatalf("stdout = %q, want %q", r.out, want)
	}
	want := "findx: warning: unrecognized format directive '%q'\n" +
		"findx: warning: unrecognized escape '\\q'\n" +
		"findx: warning: unrecognized format directive '%Tq'\n" +
		"findx: warning: escape '\\' followed by nothing at all\n"
	if r.err != want {
		t.Fatalf("stderr = %q, want %q", r.err, want)
	}
}

func TestFileActions(t *testing.T) {
	tree(t)
	out := filepath.Join(t.TempDir(), "out")
	out0 := filepath.Join(t.TempDir(), "out0")
	outf := filepath.Join(t.TempDir(), "outf")
	// Two actions naming one file share it, in order.
	r := run(t, nil, ".", "-name", "*.txt", "-fprint", out, "-fprintf", out, `<%f>\n`, "-fprint0", out0, "-fprintf", outf, "%s,")
	if r.e != nil || r.err != "" || r.out != "" {
		t.Fatalf("result = %+v", r)
	}
	for path, want := range map[string]string{
		out:  lines("./a.txt") + "<a.txt>\n" + lines("./dir/c.txt") + "<c.txt>\n",
		out0: "./a.txt\x00./dir/c.txt\x00",
		outf: "5,2000,",
	} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("%s = %q (%v), want %q", path, got, err, want)
		}
	}

	// The file is created even when nothing matches.
	none := filepath.Join(t.TempDir(), "none")
	check(t, "", ".", "-false", "-fprint", none)
	if got, err := os.ReadFile(none); err != nil || len(got) != 0 {
		t.Fatalf("empty -fprint file: %q, %v", got, err)
	}

	// /dev/stdout and /dev/stderr are the engine's own streams.
	r = run(t, nil, "a.txt", "-fprint", "/dev/stdout", "-fprint", "/dev/stderr")
	if r.out != "a.txt\n" || r.err != "a.txt\n" || r.e != nil {
		t.Fatalf("result = %+v", r)
	}

	r = run(t, nil, ".", "-fprint", filepath.Join(t.TempDir(), "no", "such", "dir"))
	if !errors.Is(r.e, core.ErrFailed) || !strings.HasSuffix(r.err, ": No such file or directory\n") {
		t.Fatalf("err = %v, stderr = %q", r.e, r.err)
	}
}

func TestDelete(t *testing.T) {
	root := tree(t)
	check(t, "", "dir", "-name", "c.txt", "-delete")
	if _, err := os.Stat(filepath.Join(root, "dir", "c.txt")); !os.IsNotExist(err) {
		t.Fatalf("c.txt still there: %v", err)
	}

	// -delete implies -depth, so a whole tree goes, and "." is left alone.
	check(t, "", "dir", "-delete")
	if _, err := os.Stat(filepath.Join(root, "dir")); !os.IsNotExist(err) {
		t.Fatalf("dir still there: %v", err)
	}
	check(t, lines(".", "./.hidden", "./a.txt", "./b.log", "./empty"), "-print")
	check(t, "", ".", "-delete")
	check(t, lines("."), ".")

	// -delete is true when it worked, so what follows it runs.
	write(t, "x", 1)
	check(t, lines("x"), "x", "-delete", "-print")
}

func TestDeleteFailure(t *testing.T) {
	tree(t)
	// A non-empty directory cannot be removed: diagnosed, false, status 1.
	r := run(t, nil, ".", "-name", "dir", "-delete", "-print")
	if !errors.Is(r.e, core.ErrFailed) || r.out != "" {
		t.Fatalf("result = %+v", r)
	}
	if want := "findx: cannot delete './dir': Directory not empty\n"; r.err != want {
		t.Fatalf("stderr = %q, want %q", r.err, want)
	}
}

// ages sets the modification and access times of four files to 30.5 minutes,
// 25, 49 and 73 hours before now.
func ages(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	for name, age := range map[string]time.Duration{
		"f0": 30*time.Minute + 30*time.Second,
		"f1": 25 * time.Hour,
		"f2": 49 * time.Hour,
		"f3": 73 * time.Hour,
	} {
		write(t, name, 1)
		if err := os.Chtimes(name, now.Add(-age), now.Add(-age)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTimeTests(t *testing.T) {
	ages(t)
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"-mtime", "0"}, lines("./f0")},
		{[]string{"-mtime", "1"}, lines("./f1")},
		{[]string{"-mtime", "2"}, lines("./f2")},
		// Fractions of a day are dropped: +1 means at least two days old.
		{[]string{"-mtime", "+1"}, lines("./f2", "./f3")},
		{[]string{"-mtime", "+0"}, lines("./f1", "./f2", "./f3")},
		{[]string{"-mtime", "-1"}, lines("./f0")},
		{[]string{"-mtime", "-2"}, lines("./f0", "./f1")},
		{[]string{"-mtime", "+2", "-mtime", "-4"}, lines("./f3")},
		{[]string{"-atime", "1"}, lines("./f1")},
		{[]string{"-atime", "+1"}, lines("./f2", "./f3")},
		{[]string{"-mmin", "-60"}, lines("./f0")},
		{[]string{"-mmin", "+60"}, lines("./f1", "./f2", "./f3")},
		{[]string{"-mmin", "+30"}, lines("./f0", "./f1", "./f2", "./f3")},
		{[]string{"-mmin", "-31"}, lines("./f0")},
		{[]string{"-mmin", "-30"}, ""},
		// N alone rounds the age up: 30.5 minutes is "31".
		{[]string{"-mmin", "31"}, lines("./f0")},
		{[]string{"-mmin", "30"}, ""},
		{[]string{"-amin", "31"}, lines("./f0")},
		{[]string{"-amin", "-1500"}, lines("./f0")},
		{[]string{"-amin", "-1501"}, lines("./f0", "./f1")},
		// -daystart counts from midnight: f0 is today, f1 yesterday.
		{[]string{"-daystart", "-mtime", "0"}, lines("./f0")},
		{[]string{"-daystart", "-mtime", "1"}, lines("./f1")},
		{[]string{"-daystart", "-mtime", "+1"}, lines("./f2", "./f3")},
		{[]string{"-daystart", "-mtime", "+2"}, lines("./f3")},
		{[]string{"-daystart", "-mtime", "-2"}, lines("./f0", "./f1")},
		// With -daystart the minute tests count back from the end of today.
		{[]string{"-daystart", "-mmin", "-751"}, lines("./f0")},
		{[]string{"-daystart", "-mmin", "-750"}, ""},
		// A fraction moves the one-day window: 0.5 is 12 to 36 hours old.
		{[]string{"-mtime", "0.5"}, lines("./f1")},
		{[]string{"-mtime", "-0.5"}, lines("./f0")},
		{[]string{"-newer", "f1"}, lines("./f0")},
		{[]string{"-anewer", "f2"}, lines("./f0", "./f1")},
		{[]string{"!", "-newer", "f2"}, lines("./f2", "./f3")},
		{[]string{"-newermm", "f1"}, lines("./f0")},
		{[]string{"-neweram", "f2"}, lines("./f0", "./f1")},
		{[]string{"-newerma", "f2"}, lines("./f0", "./f1")},
		{[]string{"-newermt", "2024-06-14"}, lines("./f0", "./f1")},
		{[]string{"-newermt", "2024-06-14 11:30"}, lines("./f0")},
		{[]string{"-newermt", "2024-06-13T10:59:59"}, lines("./f0", "./f1", "./f2")},
		{[]string{"-newermt", "2024-06-13 11:00:00 +0000"}, lines("./f0", "./f1")},
		{[]string{"-newerat", "@1718363000"}, lines("./f0")},
		{[]string{"!", "-newermt", "@1718200000.5"}, lines("./f3")},
	}
	for _, tt := range tests {
		check(t, tt.want, append([]string{".", "-type", "f"}, tt.args...)...)
	}
}

func TestSamefile(t *testing.T) {
	tree(t)
	check(t, lines("./a.txt"), ".", "-samefile", "a.txt")
	check(t, lines("./dir"), ".", "-samefile", "dir/sub/..")
}

// fakeTools is an Exec that records every call. "echo" prints its arguments,
// "fail" exits 1, "odd" exits 1 for a path containing "c.txt"; any other
// name is not a tool.
type fakeTools struct {
	calls [][]string
}

func (ft *fakeTools) exec(name string, args []string, s core.Streams) (int, bool) {
	ft.calls = append(ft.calls, append([]string{name}, args...))
	switch name {
	case "echo":
		_, _ = s.Out.Write([]byte(strings.Join(args, " ") + "\n"))
		return 0, true
	case "fail":
		return 1, true
	case "odd":
		if strings.Contains(strings.Join(args, " "), "c.txt") {
			return 1, true
		}
		return 0, true
	case "cat-stdin":
		var b bytes.Buffer
		_, _ = b.ReadFrom(s.In)
		_, _ = s.Out.Write([]byte("[" + b.String() + "]"))
		return 0, true
	}
	return 0, false
}

func withTools(ft *fakeTools) func(*find.Find) {
	return func(f *find.Find) { f.Exec = ft.exec }
}

func TestExec(t *testing.T) {
	tree(t)
	ft := &fakeTools{}
	r := run(t, withTools(ft), ".", "-name", "*.txt", "-exec", "echo", "got", "{}", "<{}>", ";")
	if r.e != nil || r.err != "" {
		t.Fatalf("result = %+v", r)
	}
	// {} is replaced wherever it appears, and the output keeps its place
	// among find's own.
	if want := "got ./a.txt <./a.txt>\ngot ./dir/c.txt <./dir/c.txt>\n"; r.out != want {
		t.Fatalf("stdout = %q, want %q", r.out, want)
	}

	// The action is true when the command exits 0, and find's own status is
	// not changed by a command that fails.
	r = run(t, withTools(&fakeTools{}), ".", "-name", "*.txt", "-exec", "odd", "{}", ";", "-print")
	if r.e != nil || r.out != lines("./a.txt") {
		t.Fatalf("result = %+v", r)
	}
	r = run(t, withTools(&fakeTools{}), ".", "-name", "*.txt", "-print", "-exec", "echo", "ran", ";", "-print")
	if want := lines("./a.txt") + "ran\n" + lines("./a.txt", "./dir/c.txt") + "ran\n" + lines("./dir/c.txt"); r.out != want {
		t.Fatalf("stdout = %q, want %q", r.out, want)
	}
	// A command reads find's input.
	r = runIn(t, "data", withTools(&fakeTools{}), "a.txt", "-exec", "cat-stdin", ";")
	if r.out != "[data]" {
		t.Fatalf("stdout = %q", r.out)
	}
}

func TestExecPlus(t *testing.T) {
	tree(t)
	ft := &fakeTools{}
	r := run(t, withTools(ft), ".", "-type", "f", "-exec", "echo", "all:", "{}", "+")
	if r.e != nil || r.err != "" {
		t.Fatalf("result = %+v", r)
	}
	if want := "all: ./.hidden ./a.txt ./b.log ./dir/c.txt ./dir/sub/d.TXT\n"; r.out != want {
		t.Fatalf("stdout = %q, want %q", r.out, want)
	}
	if len(ft.calls) != 1 {
		t.Fatalf("calls = %q, want one batch", ft.calls)
	}

	// The + form is always true, runs nothing for no files, and a failing
	// batch makes the status 1 without a diagnostic.
	ft = &fakeTools{}
	r = run(t, withTools(ft), ".", "-false", "-exec", "echo", "{}", "+")
	if r.e != nil || len(ft.calls) != 0 {
		t.Fatalf("result = %+v, calls = %q", r, ft.calls)
	}
	r = run(t, withTools(&fakeTools{}), ".", "-name", "*.txt", "-exec", "fail", "{}", "+", "-print")
	if !errors.Is(r.e, core.ErrFailed) || r.err != "" || r.out != lines("./a.txt", "./dir/c.txt") {
		t.Fatalf("result = %+v", r)
	}
	// -quit still runs what was collected.
	ft = &fakeTools{}
	r = run(t, withTools(ft), ".", "-name", "a.txt", "-exec", "echo", "{}", "+", "-quit")
	if r.out != lines("./a.txt") {
		t.Fatalf("stdout = %q", r.out)
	}
}

func TestExecPlusBatches(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	long := strings.Repeat("n", 150)
	for i := range 1200 {
		write(t, filepath.Join("d", long+string(rune('a'+i%26))+string(rune('a'+i/26%26))+string(rune('a'+i/676))), 0)
	}
	ft := &fakeTools{}
	r := run(t, withTools(ft), "d", "-type", "f", "-exec", "count", "{}", "+", "-o", "-exec", "echo", "dir", ";")
	if r.e == nil {
		// "count" is not a tool and not a program: every batch is diagnosed.
		t.Fatalf("result = %+v", r)
	}
	batches, files := 0, 0
	for _, c := range ft.calls {
		if c[0] == "count" {
			batches++
			files += len(c) - 1
		}
	}
	if batches != 2 || files != 1200 {
		t.Fatalf("batches = %d, files = %d, want 2 and 1200", batches, files)
	}
}

func TestExecdir(t *testing.T) {
	tree(t)
	ft := &fakeTools{}
	// An in-process tool cannot change directory, so it is given the path.
	r := run(t, withTools(ft), ".", "-name", "*.txt", "-execdir", "echo", "{}", ";")
	if r.e != nil || r.out != lines("./a.txt", "./dir/c.txt") {
		t.Fatalf("result = %+v", r)
	}
	ft = &fakeTools{}
	r = run(t, withTools(ft), ".", "-type", "f", "-execdir", "echo", "{}", "+")
	if want := "./.hidden ./a.txt ./b.log\n./dir/c.txt\n./dir/sub/d.TXT\n"; r.e != nil || r.out != want {
		t.Fatalf("result = %+v, want %q", r, want)
	}
}

func TestExecNotFound(t *testing.T) {
	tree(t)
	r := run(t, withTools(&fakeTools{}), "a.txt", "-exec", "gnux-no-such-program-anywhere", "{}", ";", "-print")
	if !errors.Is(r.e, core.ErrFailed) || r.out != "" {
		t.Fatalf("result = %+v", r)
	}
	if want := "findx: 'gnux-no-such-program-anywhere': No such file or directory\n"; r.err != want {
		t.Fatalf("stderr = %q, want %q", r.err, want)
	}
	// With no Exec at all the command is looked for on PATH only.
	r = run(t, nil, "a.txt", "-exec", "gnux-no-such-program-anywhere", ";")
	if !errors.Is(r.e, core.ErrFailed) || r.err != "findx: 'gnux-no-such-program-anywhere': No such file or directory\n" {
		t.Fatalf("result = %+v", r)
	}
}

func TestOk(t *testing.T) {
	tree(t)
	var asked []string
	ft := &fakeTools{}
	cfg := func(f *find.Find) {
		f.Exec = ft.exec
		f.PromptFn = func(q string) bool {
			asked = append(asked, q)
			return strings.Contains(q, "a.txt")
		}
	}
	r := run(t, cfg, ".", "-name", "*.txt", "-ok", "echo", "rm", "{}", ";", "-print")
	if r.e != nil || r.err != "" {
		t.Fatalf("result = %+v", r)
	}
	// A refusal makes the action false.
	if want := "rm ./a.txt\n./a.txt\n"; r.out != want {
		t.Fatalf("stdout = %q, want %q", r.out, want)
	}
	want := []string{"< echo ... ./a.txt > ? ", "< echo ... ./dir/c.txt > ? "}
	if strings.Join(asked, "|") != strings.Join(want, "|") {
		t.Fatalf("asked = %q, want %q", asked, want)
	}

	// Without PromptFn the question goes to the error stream and the answers
	// come from the input stream, which the command itself does not read.
	ft = &fakeTools{}
	r = runIn(t, "n\nYes\n", withTools(ft), ".", "-name", "*.txt", "-okdir", "cat-stdin", "{}", ";")
	if r.e != nil || r.out != "[]" {
		t.Fatalf("result = %+v", r)
	}
	if want := "< cat-stdin ... ./a.txt > ? < cat-stdin ... ./dir/c.txt > ? "; r.err != want {
		t.Fatalf("stderr = %q, want %q", r.err, want)
	}
	// End of input is a no.
	ft = &fakeTools{}
	r = runIn(t, "", withTools(ft), "a.txt", "-ok", "echo", ";")
	if r.e != nil || r.out != "" || len(ft.calls) != 0 {
		t.Fatalf("result = %+v, calls = %q", r, ft.calls)
	}
}

func TestUsageErrors(t *testing.T) {
	tree(t)
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"-foo"}, "unknown predicate `-foo'"},
		{[]string{".", "-name"}, "missing argument to `-name'"},
		{[]string{".", "-maxdepth"}, "missing argument to `-maxdepth'"},
		{[]string{".", "-print", "x"}, "paths must precede expression: `x'"},
		{[]string{".", "-name", "a", "b"}, "paths must precede expression: `b'"},
		{[]string{".", "-o", "-print"}, "invalid expression; you have used a binary operator '-o' with nothing before it."},
		{[]string{".", "-a", "-print"}, "invalid expression; you have used a binary operator '-a' with nothing before it."},
		{[]string{".", ",", "-print"}, "invalid expression; you have used a binary operator ',' with nothing before it."},
		{[]string{".", "-print", "-o"}, "expected an expression after '-o'"},
		{[]string{".", "-print", "-a"}, "expected an expression after '-a'"},
		{[]string{".", "!"}, "expected an expression after '!'"},
		{[]string{".", "-print", ","}, "expected an expression after ','"},
		{[]string{".", "(", "-print", "-o", ")"}, "expected an expression between '-o' and ')'"},
		{[]string{".", "(", ")"}, "invalid expression; empty parentheses are not allowed."},
		{[]string{".", "(", "-print"}, "invalid expression; I was expecting to find a ')' somewhere but did not see one."},
		{[]string{".", "("}, "invalid expression; I was expecting to find a ')' somewhere but did not see one."},
		{[]string{".", "-print", ")"}, "invalid expression; you have too many ')'"},
		{[]string{".", ")"}, "invalid expression; you have too many ')'"},
		{[]string{".", "-maxdepth", "x"}, "Expected a positive decimal integer argument to -maxdepth, but got 'x'"},
		{[]string{".", "-mindepth", "-1"}, "Expected a positive decimal integer argument to -mindepth, but got '-1'"},
		{[]string{".", "-type", ""}, "Arguments to -type should contain at least one letter"},
		{[]string{".", "-type", "x"}, "Unknown argument to -type: x"},
		{[]string{".", "-type", "fd"}, "Must separate multiple arguments to -type using: ','"},
		{[]string{".", "-type", "f,"}, "Last file type in list argument to -type is missing, i.e., list is ending on: ','"},
		{[]string{".", "-xtype", "f,f"}, "Duplicate file type 'f' in the argument list to -xtype."},
		{[]string{".", "-size", ""}, "invalid null argument to -size"},
		{[]string{".", "-size", "5x"}, "invalid -size type 'x'"},
		{[]string{".", "-size", "k"}, "Invalid argument 'k' to -size"},
		{[]string{".", "-size", "1.5k"}, "Invalid argument '1.5k' to -size"},
		{[]string{".", "-mtime", "x"}, "invalid argument `x' to `-mtime'"},
		{[]string{".", "-mmin", ""}, "invalid argument `' to `-mmin'"},
		{[]string{".", "-links", "two"}, "invalid argument `two' to `-links'"},
		{[]string{".", "-uid", "-"}, "invalid argument `-' to `-uid'"},
		{[]string{".", "-perm", "rwx"}, "invalid mode 'rwx'"},
		{[]string{".", "-perm", "88"}, "invalid mode '88'"},
		{[]string{".", "-perm", "-"}, "invalid mode '-'"},
		{[]string{".", "-perm", "77777"}, "invalid mode '77777'"},
		{[]string{".", "-user", "no-such-user-gnux"}, "'no-such-user-gnux' is not the name of a known user"},
		{[]string{".", "-group", "no-such-group-gnux"}, "'no-such-group-gnux' is not the name of an existing group"},
		{[]string{".", "-newer", "missing"}, "'missing': No such file or directory"},
		{[]string{".", "-samefile", "missing"}, "'missing': No such file or directory"},
		{[]string{".", "-newermt", "next tuesday-ish"}, "I cannot figure out how to interpret 'next tuesday-ish' as a date or time"},
		{[]string{".", "-newerxy", "a.txt"}, "unknown predicate `-newerxy'"},
		{[]string{".", "-newertm", "a.txt"}, "unknown predicate `-newertm'"},
		{[]string{".", "-regextype", "perl", "-regex", "x"}, "Unknown regular expression type 'perl'; valid types are 'findutils-default', 'ed', 'emacs', 'gnu-awk', 'grep', 'posix-awk', 'awk', 'posix-basic', 'posix-egrep', 'egrep', 'posix-extended', 'posix-minimal-basic', 'sed'."},
		{[]string{".", "-regex", `\(a`}, `Unmatched ( or \(`},
		{[]string{".", "-regex", `a\)`}, `Unmatched ) or \)`},
		{[]string{".", "-regex", `[a`}, "Unmatched [, [^, [:, [., or [="},
		{[]string{".", "-regex", `a\`}, "Trailing backslash"},
		{[]string{".", "-regex", `\(a\)\1`}, "back references are not supported in a regular expression"},
		{[]string{".", "-regextype", "posix-basic", "-regex", `\(a\)\1`}, "back references are not supported"},
		{[]string{".", "-exec", "echo", "{}"}, "missing argument to `-exec'"},
		{[]string{".", "-exec"}, "missing argument to `-exec'"},
		{[]string{".", "-exec", ";"}, "invalid argument `;' to `-exec'"},
		{[]string{".", "-ok", "echo", "{}", "+"}, "missing argument to `-ok'"},
		{[]string{".", "-exec", "echo", "{}", "x{}", "{}", "+"}, "Only one instance of {} is supported with -exec ... +"},
		{[]string{".", "-execdir", "{}", "+"}, "invalid argument `{}' to `-execdir'"},
		{[]string{".", "-printf"}, "missing argument to `-printf'"},
		{[]string{".", "-printf", "%F"}, "the %F directive of -printf is not supported"},
		{[]string{".", "-fprintf", "x"}, "missing argument to `-fprintf'"},
		{[]string{".", "-fstype", "ext4"}, "the -fstype predicate is not supported"},
		{[]string{".", "-context", "x"}, "the -context predicate is not supported"},
		{[]string{".", "-used", "1"}, "the -used predicate is not supported"},
		{[]string{"-files0-from", "x"}, "the -files0-from predicate is not supported"},
		{[]string{"-D", "tree", "."}, "the -D debug options are not supported"},
		{[]string{"-O", "."}, "The -O option must be immediately followed by a decimal integer"},
		{[]string{"-Ofast", "."}, "The -O option must be immediately followed by a decimal integer"},
	}
	for _, tt := range tests {
		// As in GNU find: the message, status 1, and no pointer to --help.
		r := run(t, nil, tt.args...)
		if !errors.Is(r.e, core.ErrFailed) {
			t.Errorf("findx %q: err = %v, want ErrFailed", tt.args, r.e)
			continue
		}
		if want := "findx: " + tt.want + "\n"; r.err != want {
			t.Errorf("findx %q:\n got %q\nwant %q", tt.args, r.err, want)
		}
		if r.out != "" {
			t.Errorf("findx %q: stdout = %q, want none", tt.args, r.out)
		}
	}
}

func TestAcceptedOptions(t *testing.T) {
	tree(t)
	want := lines("./a.txt")
	for _, args := range [][]string{
		{"-O3", ".", "-name", "a.txt"},
		{"-O0", "-P", ".", "-name", "a.txt"},
		{".", "-noleaf", "-name", "a.txt"},
		{".", "-ignore_readdir_race", "-name", "a.txt"},
		{".", "-noignore_readdir_race", "-name", "a.txt"},
		{".", "-warn", "-name", "a.txt"},
		{".", "-nowarn", "-name", "a.txt"},
		{".", "-xdev", "-name", "a.txt"},
		{".", "-mount", "-name", "a.txt"},
		{".", "-regextype", "emacs", "-name", "a.txt"},
		{".", "-daystart", "-name", "a.txt"},
		{".", "-follow", "-name", "a.txt"},
	} {
		check(t, want, args...)
	}
	// A lone "-" is a starting point, not an expression.
	r := run(t, nil, "-")
	if r.err != "findx: '-': No such file or directory\n" {
		t.Fatalf("stderr = %q", r.err)
	}
}

func TestWarnings(t *testing.T) {
	tree(t)
	global := "findx: warning: you have specified the global option -maxdepth after the argument -name, but global options are not positional, i.e., -maxdepth affects tests specified before it as well as those specified after it.  Please specify global options before other arguments.\n"

	r := run(t, nil, ".", "-warn", "-name", "a.txt", "-maxdepth", "1")
	if r.e != nil || r.out != lines("./a.txt") || r.err != global {
		t.Fatalf("result = %+v", r)
	}
	// Warnings are off when the input is not a terminal, on when it is, on
	// with Options.Warn, and -nowarn wins from where it stands.
	if r = run(t, nil, ".", "-name", "a.txt", "-maxdepth", "1"); r.err != "" {
		t.Fatalf("stderr = %q", r.err)
	}
	tty := func(f *find.Find) { f.IsTerminal = func(any) bool { return true } }
	if r = run(t, tty, ".", "-name", "a.txt", "-maxdepth", "1"); r.err != global {
		t.Fatalf("stderr = %q", r.err)
	}
	if r = run(t, func(f *find.Find) { f.Warn = true }, ".", "-name", "a.txt", "-maxdepth", "1"); r.err != global {
		t.Fatalf("stderr = %q", r.err)
	}
	if r = run(t, tty, ".", "-nowarn", "-name", "a.txt", "-maxdepth", "1"); r.err != "" {
		t.Fatalf("stderr = %q", r.err)
	}
	// Positional options do not warn.
	if r = run(t, tty, ".", "-name", "a.txt", "-daystart", "-regextype", "sed", "-warn"); r.err != "" {
		t.Fatalf("stderr = %q", r.err)
	}

	r = run(t, tty, ".", "-name", "dir/c.txt")
	if want := "findx: warning: '-name' matches against basenames only, but the given pattern contains a directory separator ('/'), thus the expression will evaluate to false all the time.  Did you mean '-wholename'?\n"; r.err != want || r.e != nil {
		t.Fatalf("result = %+v", r)
	}
	r = run(t, tty, ".", "-d", "-maxdepth", "0")
	if want := "findx: warning: the -d option is deprecated; please use -depth instead, because the latter is a POSIX-compliant feature.\n"; r.err != want || r.out != lines(".") {
		t.Fatalf("result = %+v", r)
	}
}

func TestRunTwice(t *testing.T) {
	tree(t)
	var out bytes.Buffer
	f := find.New(find.Options{}, core.Streams{In: strings.NewReader(""), Out: &out, Err: &out})
	if err := f.Run([]string{".", "-depth", "-maxdepth", "0"}); err != nil {
		t.Fatal(err)
	}
	// Nothing of the first run's options survives into the second.
	if err := f.Run([]string{"dir"}); err != nil {
		t.Fatal(err)
	}
	if want := lines(".", "dir", "dir/c.txt", "dir/sub", "dir/sub/d.TXT"); out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

// failWriter fails every write with err.
type failWriter struct{ err error }

func (w failWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWriteErrors(t *testing.T) {
	tree(t)
	var errb bytes.Buffer
	f := find.New(find.Options{}, core.Streams{In: strings.NewReader(""), Out: failWriter{errors.New("disk full")}, Err: &errb})
	if err := f.Run([]string{"."}); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
	if want := "findx: write error: Disk full\n"; errb.String() != want {
		t.Fatalf("stderr = %q, want %q", errb.String(), want)
	}

	// A reader that went away is a quiet stop.
	errb.Reset()
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = pr.Close()
	defer pw.Close()
	f = find.New(find.Options{}, core.Streams{In: strings.NewReader(""), Out: pw, Err: &errb})
	if err := f.Run([]string{"."}); err != nil || errb.Len() != 0 {
		t.Fatalf("err = %v, stderr = %q", err, errb.String())
	}
}
