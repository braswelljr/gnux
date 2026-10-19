// Package rm implements the removal engine behind rmx, a GNU-compatible
// implementation of the GNU/UNIX rm command. It mirrors rm's flags, prompting behavior and
// diagnostics while adding concurrent removal on the non-interactive path.
package rm

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/braswelljr/gnux/core"
)

// New builds a removal engine. If s.In is an *os.File attached to a terminal,
// write-protected prompting is enabled in PromptDefault mode.
func New(opts Options, s core.Streams) *Rm {
	return &Rm{
		Options: opts,
		diag:    core.NewDiag(Name, s),
		prompt:  core.NewPrompter(Name, s),
		isTTY:   core.IsTerminal(s.In),
	}
}

// Run removes each of paths according to the configured options. Diagnostics
// are written to the error stream as they occur; the returned error is
// core.ErrFailed if any removal failed, otherwise nil.
func (r *Rm) Run(paths []string) error {
	if len(paths) == 0 {
		if r.Force {
			return nil
		}
		return core.Usagef("missing operand")
	}

	// -I: a single up-front prompt gates a bulk or recursive removal.
	if r.Interactive == PromptOnce {
		if n := len(paths); r.Recursive || n > 3 {
			q := fmt.Sprintf("remove %d argument%s?", n, core.Plural(n))
			if r.Recursive {
				q = fmt.Sprintf("remove %d argument%s recursively?", n, core.Plural(n))
			}
			if !r.ask(q) {
				return nil
			}
		}
		r.Interactive = PromptNever
	}

	// Prompting reads a shared stdin, and verbose output must stay in argument
	// order; both force sequential processing. Everything else fans out.
	if r.mayPrompt() || r.Verbose {
		for _, p := range paths {
			r.removeArg(p)
		}
	} else {
		r.runConcurrent(paths)
	}
	return r.diag.Result()
}

// runConcurrent removes independent top-level arguments in parallel, bounded to
// one worker per CPU. Only reached when no prompting can occur.
func (r *Rm) runConcurrent(paths []string) {
	if len(paths) == 1 {
		r.removeArg(paths[0])
		return
	}

	// len(paths) >= 2 here (the single-path case returned above), and
	// runtime.NumCPU() >= 1, so workers is always >= 1.
	workers := min(runtime.NumCPU(), len(paths))

	jobs := make(chan string)
	var wg sync.WaitGroup
	// A panic in a worker is caught per path, so the worker lives on to take
	// the paths still coming, and is raised again here once all have ended.
	var trap core.Trap
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for p := range jobs {
				func() {
					defer trap.Catch()
					r.removeArg(p)
				}()
			}
		}()
	}
	for _, p := range paths {
		jobs <- p
	}
	close(jobs)
	wg.Wait()
	trap.Release()
}

// removeArg handles a single top-level operand.
func (r *Rm) removeArg(path string) {
	// As in GNU rm, only a recursive removal refuses . and .. outright.
	// Without -r they meet the ordinary rules: a directory is refused, and
	// -d fails, since neither can be removed while it is in use.
	if base := filepath.Base(filepath.Clean(path)); r.Recursive && (base == "." || base == "..") {
		r.diag.Errorf("refusing to remove '.' or '..' directory: skipping '%s'", path)
		return
	}

	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if r.Force {
				return
			}
			r.diag.Errorf("cannot remove '%s': No such file or directory", path)
			return
		}
		r.diag.Errorf("cannot remove '%s': %s", path, core.ErrText(err))
		return
	}

	if info.IsDir() {
		if r.Recursive && !r.NoPreserveRoot && core.IsRoot(path) {
			r.diag.Errorf("it is dangerous to operate recursively on '%s'", path)
			r.diag.Warnf("use --no-preserve-root to override this failsafe")
			return
		}
		r.removeDirArg(path, info)
		return
	}

	r.removeFileEntry(path, info)
}

// removeDirArg dispatches directory removal based on -r and -d.
func (r *Rm) removeDirArg(path string, info os.FileInfo) {
	switch {
	case r.Recursive:
		if r.fastPathOK() {
			if err := os.RemoveAll(path); err != nil {
				r.diag.Errorf("cannot remove '%s': %s", path, core.ErrText(err))
			}
			return
		}
		dev, haveDev := uint64(0), false
		if r.OneFileSystem {
			dev, haveDev = core.DeviceID(info)
		}
		r.removeTree(path, dev, haveDev)

	case r.Dir:
		// Report a non-empty directory with rm's canonical message rather than
		// the OS-native errno text (which differs across platforms).
		entries, err := os.ReadDir(path)
		if err != nil {
			r.diag.Errorf("cannot remove '%s': %s", path, core.ErrText(err))
			return
		}
		if len(entries) > 0 {
			r.diag.Errorf("cannot remove '%s': Directory not empty", path)
			return
		}
		if !r.confirmDir(path) {
			return
		}
		if err := r.del(path); err != nil {
			r.diag.Errorf("cannot remove '%s': %s", path, core.ErrText(err))
			return
		}
		if r.Verbose {
			r.diag.Printf("removed directory '%s'", path)
		}

	default:
		r.diag.Errorf("cannot remove '%s': Is a directory", path)
	}
}

// removeTree removes a directory and its contents post-order, honoring
// prompting, --one-file-system and --verbose. It returns true only if the whole
// subtree (including the directory itself) was removed.
func (r *Rm) removeTree(path string, rootDev uint64, haveDev bool) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		r.diag.Errorf("cannot remove '%s': %s", path, core.ErrText(err))
		return false
	}

	if r.Interactive == PromptAlways && len(entries) > 0 {
		if !r.ask(fmt.Sprintf("descend into directory '%s'?", path)) {
			return false
		}
	}

	childSkipped := false
	for _, e := range entries {
		child := filepath.Join(path, e.Name())
		ci, err := os.Lstat(child)
		if err != nil {
			r.diag.Errorf("cannot remove '%s': %s", child, core.ErrText(err))
			childSkipped = true
			continue
		}

		if ci.IsDir() {
			if r.OneFileSystem && haveDev {
				if cdev, ok := core.DeviceID(ci); ok && cdev != rootDev {
					r.diag.Warnf("skipping '%s', since it's on a different device", child)
					childSkipped = true
					continue
				}
			}
			if !r.removeTree(child, rootDev, haveDev) {
				childSkipped = true
			}
		} else if !r.removeFileEntry(child, ci) {
			childSkipped = true
		}
	}

	if childSkipped {
		return false
	}

	if !r.confirmDir(path) {
		return false
	}
	if err := r.del(path); err != nil {
		r.diag.Errorf("cannot remove '%s': %s", path, core.ErrText(err))
		return false
	}
	if r.Verbose {
		r.diag.Printf("removed directory '%s'", path)
	}
	return true
}

// removeFileEntry removes a single non-directory entry, prompting as required.
// It returns true if the entry was removed.
func (r *Rm) removeFileEntry(path string, info os.FileInfo) bool {
	switch {
	case r.Interactive == PromptAlways:
		if !r.ask(fmt.Sprintf("remove %s '%s'?", core.Describe(info), path)) {
			return false
		}
	case r.Interactive == PromptDefault && r.isTTY && !core.Writable(path):
		if !r.ask(fmt.Sprintf("remove write-protected %s '%s'?", core.Describe(info), path)) {
			return false
		}
	}

	if err := r.del(path); err != nil {
		if r.Force && os.IsNotExist(err) {
			return false
		}
		r.diag.Errorf("cannot remove '%s': %s", path, core.ErrText(err))
		return false
	}
	if r.Verbose {
		r.diag.Printf("removed '%s'", path)
	}
	return true
}

// confirmDir asks for confirmation before removing an (empty) directory,
// returning true when removal should proceed.
func (r *Rm) confirmDir(path string) bool {
	switch {
	case r.Interactive == PromptAlways:
		return r.ask(fmt.Sprintf("remove directory '%s'?", path))
	case r.Interactive == PromptDefault && r.isTTY && !core.Writable(path):
		return r.ask(fmt.Sprintf("remove write-protected directory '%s'?", path))
	default:
		return true
	}
}

// mayPrompt reports whether any removal could trigger an interactive prompt.
func (r *Rm) mayPrompt() bool {
	return r.Interactive == PromptAlways || (r.Interactive == PromptDefault && r.isTTY)
}

// fastPathOK reports whether a directory can be removed via a single
// os.RemoveAll rather than a manual, decision-making walk.
func (r *Rm) fastPathOK() bool {
	return r.RemoveFn == nil && !r.mayPrompt() && !r.Verbose && !r.OneFileSystem
}

// del performs the actual unlink, honoring an injected RemoveFn.
func (r *Rm) del(path string) error {
	if r.RemoveFn != nil {
		return r.RemoveFn(path)
	}
	return os.Remove(path)
}

// ask poses a yes/no question, returning true on an affirmative answer.
func (r *Rm) ask(question string) bool {
	if r.PromptFn != nil {
		return r.PromptFn(question)
	}
	return r.prompt.Ask(question)
}
