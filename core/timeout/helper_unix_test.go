//go:build unix

package timeout_test

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// helperUnix adds the helper commands that need Unix signals:
// "ignore-term SECONDS" sleeps with TERM ignored, and "hup-self" sends itself
// HUP and exits 0 if it survives.
func helperUnix(args []string) {
	switch args[0] {
	case "ignore-term":
		signal.Ignore(syscall.SIGTERM)
		f, _ := strconv.ParseFloat(args[1], 64)
		time.Sleep(time.Duration(f * float64(time.Second)))
	case "hup-self":
		_ = syscall.Kill(os.Getpid(), syscall.SIGHUP)
		time.Sleep(200 * time.Millisecond)
	}
}
