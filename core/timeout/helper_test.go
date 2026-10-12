package timeout_test

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"
	"time"
)

// helperEnv marks a run of the test binary as the helper command the engine
// starts, so the tests need no external programs on any platform.
const helperEnv = "GNUX_TIMEOUT_HELPER"

// TestMain runs the helper when asked to, and otherwise the tests, with
// helperEnv set so that every child the engine starts is the helper.
func TestMain(m *testing.M) {
	if os.Getenv(helperEnv) != "" {
		helper(os.Args[1:])
		return
	}
	_ = os.Setenv(helperEnv, "1")
	os.Exit(m.Run())
}

// helper is the command the tests run: "sleep SECONDS", "exit CODE",
// "echo TEXT...", "cat" (standard input to standard output), "stderr TEXT"
// or "pid" (prints its process ID); helperUnix adds the Unix-only ones.
func helper(args []string) {
	if len(args) == 0 {
		os.Exit(0)
	}
	switch args[0] {
	case "sleep":
		f, _ := strconv.ParseFloat(args[1], 64)
		time.Sleep(time.Duration(f * float64(time.Second)))
	case "exit":
		n, _ := strconv.Atoi(args[1])
		os.Exit(n)
	case "echo":
		for _, a := range args[1:] {
			fmt.Println(a)
		}
	case "cat":
		if _, err := io.Copy(os.Stdout, os.Stdin); err != nil {
			fmt.Fprintln(os.Stderr, "cat:", err)
			os.Exit(1)
		}
	case "stderr":
		fmt.Fprintln(os.Stderr, args[1])
	case "pid":
		fmt.Println(os.Getpid())
	default:
		helperUnix(args)
	}
	os.Exit(0)
}
