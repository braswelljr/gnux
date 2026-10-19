// Package sandbox confines the gnux process to the directories it is given,
// with the operating system's own sandbox: Landlock on Linux, Seatbelt
// (sandbox-exec) on macOS, an AppContainer on Windows and Capsicum on
// FreeBSD. Once entered, the confinement covers every tool
// gnux runs in-process and every program it starts, and cannot be lifted.
//
// A Policy names the directories the process may write (and read) and those
// it may only read. The system's own directories (programs, libraries, /etc,
// time zones) stay readable, and /dev/null writable; everything else, the
// home directory included, can be neither read nor written. Names and sizes
// of files outside the policy may still be visible to stat on macOS, never
// their contents.
//
// Where there is no kernel sandbox to use (js, wasip1, a Linux kernel
// without Landlock, a build with cgo on Linux), Enter refuses rather than run
// unconfined: a sandbox that silently is not one would report a failure as
// success.
package sandbox

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Policy is what a sandboxed process may reach.
type Policy struct {
	// Write are the directories (or files) the process may read and write.
	Write []string
	// Read are the directories (or files) it may only read.
	Read []string
}

// ErrUnsupported says the system has no sandbox Enter can use.
var ErrUnsupported = errors.New("no kernel sandbox is available on this system")

// Enter confines the current process to p, or returns why it cannot. On
// macOS it does not return on success: the process is replaced by itself
// under sandbox-exec, which runs the same command line confined. A path in
// p that does not exist is an error.
func Enter(p Policy) error {
	resolved, err := p.resolve()
	if err != nil {
		return err
	}
	if err := enter(resolved); err != nil {
		return err
	}
	return resolved.verify()
}

// resolve makes every path absolute and free of symbolic links (the
// sandboxes match real paths), and drops repeats.
func (p Policy) resolve() (Policy, error) {
	var out Policy
	for _, list := range []struct {
		in  []string
		out *[]string
	}{{p.Write, &out.Write}, {p.Read, &out.Read}} {
		for _, path := range list.in {
			abs, err := filepath.Abs(path)
			if err != nil {
				return Policy{}, err
			}
			real, err := filepath.EvalSymlinks(abs)
			if err != nil {
				return Policy{}, fmt.Errorf("sandbox: %s: %w", path, err)
			}
			if !slices.Contains(*list.out, real) {
				*list.out = append(*list.out, real)
			}
		}
	}
	if len(out.Write) == 0 && len(out.Read) == 0 {
		return Policy{}, errors.New("sandbox: no directory to allow")
	}
	return out, nil
}

// verify checks that the confinement took: creating a file in a directory
// the policy does not reach (the temporary directory, the home directory or
// /tmp, whichever is outside it) must now fail. Should it succeed, the file
// is removed and the process refuses to go on.
func (p Policy) verify() error {
	var probes []string
	if home, err := os.UserHomeDir(); err == nil {
		probes = append(probes, home)
	}
	probes = append(probes, os.TempDir(), "/tmp")
	for _, dir := range probes {
		real, err := filepath.EvalSymlinks(dir)
		if err != nil {
			// Under Capsicum nothing can be looked up by path any more.
			real = filepath.Clean(dir)
		}
		if p.reaches(real) {
			continue
		}
		f, err := os.CreateTemp(real, ".gnux-sandbox-probe-*")
		if errors.Is(err, fs.ErrPermission) || refusedHere(err) {
			return nil // refused, as a sandbox refuses
		}
		if err != nil {
			continue // refused for some other reason: proves nothing
		}
		name := f.Name()
		_ = f.Close()
		_ = os.Remove(name)
		return errors.New("sandbox: the sandbox was entered but does not confine this process; refusing to go on")
	}
	return nil
}

// reaches reports whether path is one of p's paths or lies beneath one.
func (p Policy) reaches(path string) bool {
	for _, allowed := range append(append([]string(nil), p.Write...), p.Read...) {
		rel, err := filepath.Rel(allowed, path)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// existing returns the paths that exist, for the system directories, whose
// set differs between distributions.
func existing(paths []string) []string {
	var out []string
	for _, p := range paths {
		if real, err := filepath.EvalSymlinks(p); err == nil && !slices.Contains(out, real) {
			out = append(out, real)
		}
	}
	return out
}
