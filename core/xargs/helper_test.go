package xargs_test

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
)

// helperEnv marks a run of the test binary as the helper program the engine
// starts, so the tests of real programs need none installed on any platform.
const helperEnv = "GNUX_XARGS_HELPER"

// TestMain runs the helper when asked to, and otherwise the tests, with
// helperEnv set so that the test binary started by the engine is the helper.
func TestMain(m *testing.M) {
	// A fuzz worker is this binary too, started by the test with the same
	// environment: its -test. flags tell it apart from a helper run.
	worker := len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "-test.")
	if os.Getenv(helperEnv) != "" && !worker {
		ownCoverDir()
		helper(os.Args[1:])
		return
	}
	_ = os.Setenv(helperEnv, "1")
	if os.Getenv(coverDirEnv) != "" {
		// Helpers write their coverage files under here: see ownCoverDir.
		if dir, err := os.MkdirTemp("", "xargs-helper-cover"); err == nil {
			_ = os.Setenv(helperCoverEnv, dir)
		}
	}
	// A binary built with the race detector sleeps a second before it exits;
	// the helper has nothing to report, so its runs skip that.
	if os.Getenv("GORACE") == "" {
		_ = os.Setenv("GORACE", "atexit_sleep_ms=0")
	}
	status := m.Run()
	if dir := os.Getenv(helperCoverEnv); dir != "" {
		_ = os.RemoveAll(dir)
	}
	os.Exit(status)
}

// The Go runtime's coverage directory, and where this test's helpers put
// theirs.
const (
	coverDirEnv    = "GOCOVERDIR"
	helperCoverEnv = "GNUX_XARGS_HELPER_COVER"
)

// ownCoverDir gives a helper a coverage directory of its own. Under
// `go test -cover` every helper writes the same meta-data file as it exits,
// into the directory it inherited; several at once (xargs -P) collide on
// Windows, where one cannot rename over a file another holds open, and the
// loser complains on standard error, which the tests compare.
func ownCoverDir() {
	base := os.Getenv(helperCoverEnv)
	if base == "" {
		return
	}
	if dir, err := os.MkdirTemp(base, "run"); err == nil {
		_ = os.Setenv(coverDirEnv, dir)
	}
}

// helper is the program the tests run: "echo TEXT..." prints its arguments on
// one line, "exit CODE ..." exits with CODE, "cat" copies standard input to
// standard output; helperUnix adds the Unix-only ones.
func helper(args []string) {
	if len(args) == 0 {
		os.Exit(0)
	}
	switch args[0] {
	case "echo":
		fmt.Println(strings.Join(args[1:], " "))
	case "exit":
		n, _ := strconv.Atoi(args[1])
		os.Exit(n)
	case "cat":
		_, _ = io.Copy(os.Stdout, os.Stdin)
	default:
		helperUnix(args)
	}
	os.Exit(0)
}
