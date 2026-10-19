//go:build darwin

package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// sandboxExec is macOS's command that runs a program under a Seatbelt
// profile.
const sandboxExec = "/usr/bin/sandbox-exec"

// marker is set in the environment of gnux re-run under sandbox-exec, to the
// digest of the profile it runs under, so that the re-run gnux knows it is
// already confined to that policy (and checks it).
const marker = "GNUX_SANDBOX"

// systemRead are what stays readable: the programs, libraries, frameworks
// and configuration a tool or a program it starts needs.
var systemRead = []string{
	"/usr", "/bin", "/sbin", "/System", "/Library", "/opt/homebrew",
	"/private/etc", "/private/var/db/timezone", "/private/var/db/dyld",
	"/dev/zero", "/dev/random", "/dev/urandom", "/dev/fd",
}

// systemWrite are the devices that stay writable.
var systemWrite = []string{"/dev/null", "/dev/tty", "/dev/dtracehelper"}

// enter re-runs the command line under sandbox-exec with a profile for p.
// The re-run gnux, finding the marker for the same profile, returns at once
// (and Enter then checks the confinement).
func enter(p Policy) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("sandbox: %w", err)
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return fmt.Errorf("sandbox: %w", err)
	}
	profile := profileFor(p, exe)
	digest := sha256.Sum256([]byte(profile))
	id := hex.EncodeToString(digest[:8])
	if os.Getenv(marker) == id {
		// Already confined to this policy; the programs this process starts
		// must not take the marker for theirs.
		_ = os.Unsetenv(marker)
		return nil
	}
	if _, err := os.Stat(sandboxExec); err != nil {
		return fmt.Errorf("sandbox: %w: %s is missing", ErrUnsupported, sandboxExec)
	}
	env := append(os.Environ(), marker+"="+id)
	argv := append([]string{"sandbox-exec", "-p", profile, exe}, os.Args[1:]...)
	if err := syscall.Exec(sandboxExec, argv, env); err != nil {
		return fmt.Errorf("sandbox: running %s: %w", sandboxExec, err)
	}
	return nil // not reached
}

// profileFor writes the Seatbelt profile for p: nothing is allowed but what
// it lists. Metadata (stat) is readable everywhere, as resolving a path
// needs it; contents only where p and the system directories say.
func profileFor(p Policy, exe string) string {
	var b strings.Builder
	b.WriteString("(version 1)\n(deny default)\n")
	// The Go runtime and the programs gnux starts: processes (which inherit
	// this profile), signals among them, system information.
	b.WriteString("(allow process-fork)\n(allow process-exec)\n(allow signal (target same-sandbox))\n")
	b.WriteString("(allow sysctl-read)\n(allow mach-lookup)\n(allow ipc-posix-shm-read-data)\n")
	b.WriteString("(allow file-read-metadata)\n")
	// libSystem reads the root directory itself as a process starts; that
	// shows the names at the top of the disk, nothing beneath them.
	b.WriteString("(allow file-read-data (literal \"/\"))\n")
	b.WriteString("(allow file-ioctl (literal \"/dev/tty\") (literal \"/dev/null\"))\n")
	writeRule(&b, "file-read*", existing(systemRead))
	writeRule(&b, "file-read* file-write*", existing(systemWrite))
	writeRule(&b, "file-read*", []string{exe})
	writeRule(&b, "file-read*", p.Read)
	writeRule(&b, "file-read* file-write*", p.Write)
	return b.String()
}

// writeRule allows ops on every path, and beneath each one that is a
// directory.
func writeRule(b *strings.Builder, ops string, paths []string) {
	if len(paths) == 0 {
		return
	}
	b.WriteString("(allow ")
	b.WriteString(ops)
	for _, p := range paths {
		kind := "literal"
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			kind = "subpath"
		}
		b.WriteString(" (")
		b.WriteString(kind)
		b.WriteString(" ")
		b.WriteString(quote(p))
		b.WriteString(")")
	}
	b.WriteString(")\n")
}

// quote writes s as a Seatbelt (Scheme) string.
func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
