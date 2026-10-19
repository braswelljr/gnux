package run_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/run"
)

// helperEnv makes the test binary act as the external program: TestMain
// sees it and does what its arguments say instead of running the tests.
const helperEnv = "GNUX_RUN_HELPER"

func TestMain(m *testing.M) {
	if os.Getenv(helperEnv) == "1" {
		helper(os.Args[1:])
	}
	os.Exit(m.Run())
}

// helper is the external program: "exit N", "echo ARG...", "pwd", "cat",
// "stderr TEXT" or "kill".
func helper(args []string) {
	switch args[0] {
	case "exit":
		n, _ := strconv.Atoi(args[1])
		os.Exit(n)
	case "echo":
		fmt.Println(strings.Join(args[1:], " "))
	case "pwd":
		dir, _ := os.Getwd()
		fmt.Println(dir)
	case "cat":
		_, _ = io.Copy(os.Stdout, os.Stdin)
	case "stderr":
		fmt.Fprintln(os.Stderr, args[1])
	case "kill":
		p, _ := os.FindProcess(os.Getpid())
		_ = p.Kill()
		select {}
	}
	os.Exit(0)
}

// program runs the helper as an external program with argv.
func program(t *testing.T, dir, stdin string, argv ...string) (run.Result, string, string, error) {
	t.Helper()
	t.Setenv(helperEnv, "1")
	exe, err := os.Executable()
	if err != nil {
		t.Skipf("cannot find the test binary: %v", err)
	}
	var out, errb bytes.Buffer
	s := core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}
	res, err := run.Command(nil, append([]string{exe}, argv...), dir, s)
	return res, out.String(), errb.String(), err
}

func TestProgramStreams(t *testing.T) {
	res, out, _, err := program(t, "", "", "echo", "a", "b c")
	if err != nil || res != (run.Result{}) || out != "a b c\n" {
		t.Fatalf("res = %+v, out = %q, err = %v", res, out, err)
	}
	_, out, _, _ = program(t, "", "from stdin\n", "cat")
	if out != "from stdin\n" {
		t.Fatalf("stdin was not passed: %q", out)
	}
	_, _, errOut, _ := program(t, "", "", "stderr", "oops")
	if errOut != "oops\n" {
		t.Fatalf("stderr was not passed: %q", errOut)
	}
}

func TestProgramExitStatus(t *testing.T) {
	for _, code := range []int{1, 2, 126, 255} {
		res, _, _, err := program(t, "", "", "exit", strconv.Itoa(code))
		if err != nil || res.Status != code || res.Signaled {
			t.Fatalf("exit %d: res = %+v, err = %v", code, res, err)
		}
	}
}

func TestProgramDirectory(t *testing.T) {
	dir := t.TempDir()
	_, out, _, err := program(t, dir, "", "pwd")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := filepath.EvalSymlinks(strings.TrimSpace(out))
	want, _ := filepath.EvalSymlinks(dir)
	if got != want {
		t.Fatalf("ran in %q; want %q", got, want)
	}
}

func TestProgramKilledBySignal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ends a process with an exit status, not a signal")
	}
	res, _, _, err := program(t, "", "", "kill")
	if err != nil || !res.Signaled || res.Status != -1 || res.Signal != 9 {
		t.Fatalf("res = %+v, err = %v", res, err)
	}
}

// TestNotExecutable: a match that cannot be run keeps its own error rather
// than being "not found".
func TestNotExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows decides what is executable by extension")
	}
	f := filepath.Join(t.TempDir(), "script")
	if err := os.WriteFile(f, []byte("not a program"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := run.Command(nil, []string{f}, "", core.Discard())
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v; want a permission error", err)
	}
}

// TestToolNotFoundFallsBackToPath: a ToolFunc that does not know the name
// leaves it to PATH.
func TestToolNotFoundFallsBackToPath(t *testing.T) {
	t.Setenv(helperEnv, "1")
	exe, err := os.Executable()
	if err != nil {
		t.Skip(err)
	}
	asked := ""
	tool := func(name string, _ []string, _ core.Streams) (int, bool) { asked = name; return 0, false }
	var out bytes.Buffer
	res, err := run.Command(tool, []string{exe, "echo", "hi"}, "", core.Streams{In: strings.NewReader(""), Out: &out, Err: io.Discard})
	if err != nil || res.Status != 0 || out.String() != "hi\n" || asked != exe {
		t.Fatalf("res = %+v, err = %v, out = %q, asked %q", res, err, out.String(), asked)
	}
}
