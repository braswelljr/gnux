//go:build conformance

package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestConformance runs every command line in testdata/conformance twice, in
// two identical fresh directories: once with GNU's tools, once with gnux
// linked under the GNU names ahead of them on PATH. Standard output, standard
// error and the exit status must be the same, once gnux's tool names in
// diagnostics ("rmx: ...", "Try 'rmx --help'") are read as GNU's.
//
// It needs GNU coreutils, grep, sed and findutils, so it is behind the
// conformance build tag and skips anywhere ls is not GNU's: run it with
// make conformance (Linux) or make conformance/docker (anywhere with Docker).
// A line is a command for sh -c; '#' starts a comment line, and a line ending
// in '\' continues on the next.
func TestConformance(t *testing.T) {
	if out, err := exec.Command("ls", "--version").Output(); err != nil || !bytes.Contains(out, []byte("GNU coreutils")) {
		t.Skip("GNU coreutils is not the system's ls")
	}

	bin := t.TempDir()
	gnux := filepath.Join(bin, "gnux")
	build := exec.Command("go", "build", "-o", gnux, "github.com/braswelljr/gnux")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	var tools []string
	for _, s := range Tools() {
		if err := os.Symlink(gnux, filepath.Join(bin, s.GNU)); err != nil {
			t.Fatal(err)
		}
		tools = append(tools, s.GNU)
	}
	if err := os.Symlink(gnux, filepath.Join(bin, "[")); err != nil {
		t.Fatal(err)
	}

	files, err := filepath.Glob(filepath.Join("testdata", "conformance", "*.txt"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no cases: %v", err)
	}
	for _, file := range files {
		cases, err := readCases(file)
		if err != nil {
			t.Fatal(err)
		}
		group := strings.TrimSuffix(filepath.Base(file), ".txt")
		for _, c := range cases {
			t.Run(group+"/"+c.line, func(t *testing.T) {
				t.Parallel()
				want := runCase(t, c.line, "")
				got := runCase(t, c.line, bin)
				// Verbose output (mkdir -v) carries the name too.
				got.stdout = gnuNames(got.stdout, tools)
				got.stderr = gnuNames(got.stderr, tools)
				if got != want {
					t.Errorf("%s:%d: %s\n%s", file, c.no, c.line, diffResults(want, got))
				}
			})
		}
	}
}

type conformanceCase struct {
	no   int
	line string
}

func readCases(file string) ([]conformanceCase, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cases []conformanceCase
	sc := bufio.NewScanner(f)
	no, pending, start := 0, "", 0
	for sc.Scan() {
		no++
		line := sc.Text()
		if pending == "" {
			start = no
			if t := strings.TrimSpace(line); t == "" || strings.HasPrefix(t, "#") {
				continue
			}
		}
		if strings.HasSuffix(line, `\`) {
			pending += strings.TrimSuffix(line, `\`) + "\n"
			continue
		}
		cases = append(cases, conformanceCase{start, pending + line})
		pending = ""
	}
	return cases, sc.Err()
}

type result struct {
	stdout, stderr string
	status         int
}

// runCase runs line with sh in a fresh copy of the fixture, with gnuxBin
// ahead of the system's PATH when it is not empty.
func runCase(t *testing.T, line, gnuxBin string) result {
	t.Helper()
	dir := t.TempDir()
	fixture(t, dir)
	path := os.Getenv("PATH")
	if gnuxBin != "" {
		path = gnuxBin + string(os.PathListSeparator) + path
	}
	cmd := exec.Command("sh", "-c", line)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + path, "LC_ALL=C", "TZ=UTC", "HOME=" + dir, "COLUMNS=80"}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	cmd.WaitDelay = 10 * time.Second
	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { done <- cmd.Wait() }()
	var err error
	select {
	case err = <-done:
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatalf("%q did not finish", line)
	}
	status := 0
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		status = exit.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	// The two runs have directories of their own: a path that names one
	// is written as $DIR.
	clean := func(s string) string { return strings.ReplaceAll(s, dir, "$DIR") }
	return result{clean(out.String()), clean(errb.String()), status}
}

// fixture is the directory every case starts in. Its times are fixed, so
// that a long listing is the same in both runs.
func fixture(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"a.txt":           "alpha\nbeta\ngamma\n",
		"b.txt":           "beta\ndelta\n",
		"empty":           "",
		"nums":            "10\n9\n100\n-3\n2.5\n1e2\n0x10\n\n",
		"words":           "the quick brown fox\njumps over\nthe lazy dog\n",
		"dup":             "a\na\nb\nB\nb\nc\nc\nc\n",
		"csv":             "name,age,city\nann,31,oslo\nbob,25,rome\ncid,,lima\n",
		"tabs":            "a\tb\tc\n1\t2\t3\n",
		"noeol":           "no newline at the end",
		"crlf":            "dos\r\nline\r\n",
		"bin":             "\x00\x01\x02text\xff\n",
		"space name.txt":  "spaced\n",
		"dir/one":         "1\n",
		"dir/two":         "2\n",
		"dir/sub/three":   "3\n",
		"dir/sub/.hidden": "h\n",
		"deps":            "lib app\nlib test\napp release\ntest release\n",
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	stamp := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	for _, name := range names {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(files[name]), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("a.txt", filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	_ = filepath.Walk(dir, func(p string, _ os.FileInfo, _ error) error {
		_ = os.Chtimes(p, stamp, stamp)
		return nil
	})
}

// gnuNames rewrites gnux's tool names in diagnostics to the GNU names they
// mirror, which is the one difference gnux makes on purpose.
func gnuNames(s string, tools []string) string {
	for _, gnu := range tools {
		re := regexp.MustCompile(`(^|\n|'|\b)` + regexp.QuoteMeta(gnu) + `x([: ])`)
		s = re.ReplaceAllString(s, "${1}"+gnu+"${2}")
	}
	return s
}

func diffResults(want, got result) string {
	var b strings.Builder
	if want.status != got.status {
		fmt.Fprintf(&b, "status: GNU %d, gnux %d\n", want.status, got.status)
	}
	if want.stdout != got.stdout {
		fmt.Fprintf(&b, "stdout:\n  GNU:  %q\n  gnux: %q\n", want.stdout, got.stdout)
	}
	if want.stderr != got.stderr {
		fmt.Fprintf(&b, "stderr:\n  GNU:  %q\n  gnux: %q\n", want.stderr, got.stderr)
	}
	return b.String()
}
