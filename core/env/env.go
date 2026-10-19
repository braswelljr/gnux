// Package env implements the engine behind envx, a GNU-compatible implementation of the
// GNU/UNIX env command: print the environment, or run a command in a modified
// one.
package env

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/braswelljr/gnux/core/fsys"

	"github.com/braswelljr/gnux/core"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "envx"

// Exit statuses env reserves, so a caller can tell envx's own failures from
// those of the command it ran.
const (
	// ExitCanceled means envx itself failed before running the command.
	ExitCanceled = 125
	// ExitCannotInvoke means the command was found but could not be run.
	ExitCannotInvoke = 126
	// ExitNotFound means the command could not be found.
	ExitNotFound = 127
)

// Options mirrors the GNU env flag surface.
type Options struct {
	// IgnoreEnvironment (-i) starts from an empty environment.
	IgnoreEnvironment bool
	// Unset (-u NAME) lists variables to remove from the environment.
	Unset []string
	// Null (-0) ends each printed entry with NUL instead of a newline.
	Null bool
	// Chdir (-C DIR) is the working directory to run the command in.
	Chdir string
	// Argv0 (-a ARG) is passed to the command as its zeroth argument in place
	// of the command name.
	Argv0 string
	// Debug (-v) traces each processing step on the error stream.
	Debug bool
}

// Env is the engine. Construct it with New.
type Env struct {
	Options

	// Environ supplies the inherited environment as NAME=VALUE entries. It is
	// the injection point tests use to stay independent of the real
	// environment; it defaults to os.Environ.
	Environ func() []string

	streams core.Streams
	diag    *core.Diag
}

// New builds an env engine using the streams s, which are also the streams a
// command it runs inherits.
func New(opts Options, s core.Streams) *Env {
	return &Env{
		Options: opts,
		Environ: os.Environ,
		streams: s,
		diag:    core.NewDiag(Name, s),
	}
}

// Run applies the options and the leading NAME=VALUE operands to the
// environment, then either prints the result or, when operands remain, runs
// them as a command. The command's exit status is returned as a
// *core.ExitError.
func (e *Env) Run(args []string) error {
	ignore := e.IgnoreEnvironment
	// A lone "-" is the historical spelling of -i.
	if len(args) > 0 && args[0] == "-" {
		ignore = true
		args = args[1:]
	}

	var vars []string
	if ignore {
		e.debugf("cleaning environ")
	} else {
		vars = slices.Clone(e.Environ())
	}

	for _, name := range e.Unset {
		e.debugf("unset:    %s", name)
		if name == "" || strings.Contains(name, "=") {
			return e.fail(ExitCanceled, "cannot unset '%s': Invalid argument", name)
		}
		vars = unset(vars, name)
	}

	for len(args) > 0 && strings.Contains(args[0], "=") {
		e.debugf("setenv:   %s", args[0])
		vars = set(vars, args[0])
		args = args[1:]
	}

	switch hasCommand := len(args) > 0; {
	case e.Null && hasCommand:
		return core.Usagef("cannot specify --null (-0) with command")
	case e.Chdir != "" && !hasCommand:
		return core.Usagef("must specify command with --chdir (-C)")
	case e.Argv0 != "" && !hasCommand:
		return core.Usagef("must specify command with --argv0 (-a)")
	case !hasCommand:
		return e.print(vars)
	}
	return e.exec(vars, args)
}

// print writes the environment, one entry per line (or NUL-terminated).
func (e *Env) print(vars []string) error {
	end := byte('\n')
	if e.Null {
		end = 0
	}
	var buf bytes.Buffer
	for _, v := range vars {
		buf.WriteString(v)
		buf.WriteByte(end)
	}
	if _, err := e.streams.Out.Write(buf.Bytes()); err != nil {
		return e.fail(ExitCanceled, "write error: %s", core.ErrText(err))
	}
	return nil
}

// exec runs args as a command with the environment vars and waits for it.
//
// GNU env replaces itself with the command (execvp). That is not possible on
// every platform, so envx always runs the command as a child wired to the same
// streams and reports the child's exit status as its own.
func (e *Env) exec(vars, args []string) error {
	name := args[0]

	if e.Chdir != "" {
		e.debugf("chdir:    '%s'", e.Chdir)
		// Checked up front so a bad directory is reported as envx's failure
		// rather than as a failure to start the command.
		info, err := fsys.Stat(e.Chdir)
		switch {
		case err != nil:
			return e.fail(ExitCanceled, "cannot change directory to '%s': %s", e.Chdir, core.ErrText(err))
		case !info.IsDir():
			return e.fail(ExitCanceled, "cannot change directory to '%s': Not a directory", e.Chdir)
		}
	}

	argv := slices.Clone(args)
	if e.Argv0 != "" {
		argv[0] = e.Argv0
	}
	e.debugf("executing: %s", name)
	for i, a := range argv {
		e.debugf("   arg[%d]= '%s'", i, a)
	}

	// The search uses the PATH the command will see, not the one envx was
	// started with, and happens relative to the new working directory.
	path, err := lookPath(name, e.Chdir, func(key string) (string, bool) { return lookup(vars, key) })
	if err != nil {
		return e.cannotRun(name, err)
	}

	cmd := &exec.Cmd{
		Path: path,
		Args: argv,
		// A nil Env would make the child inherit envx's own environment, so
		// an emptied one must still be a non-nil slice.
		Env:    append([]string{}, vars...),
		Dir:    e.Chdir,
		Stdin:  e.streams.In,
		Stdout: e.streams.Out,
		Stderr: e.streams.Err,
	}
	if err := cmd.Start(); err != nil {
		return e.cannotRun(name, err)
	}

	err = cmd.Wait()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &exitErr):
		return &core.ExitError{Code: exitStatus(exitErr.ProcessState)}
	default:
		// The command ran, but relaying its streams failed.
		return e.fail(ExitCanceled, "%s", core.ErrText(err))
	}
}

// cannotRun reports why the command name could not be started and picks the
// matching exit status: 127 when it does not exist, 126 for anything else.
func (e *Env) cannotRun(name string, err error) error {
	e.diag.Errorf("'%s': %s", name, core.ErrText(err))
	if !errors.Is(err, fs.ErrNotExist) {
		return &core.ExitError{Code: ExitCannotInvoke}
	}
	// A "command" with spaces in it is nearly always a #! line whose
	// interpreter arguments arrived as one word.
	if strings.ContainsAny(name, " \t\n\v\f\r") {
		e.diag.Warnf("use -[v]S to pass options in shebang lines")
	}
	return &core.ExitError{Code: ExitNotFound}
}

// fail writes a diagnostic and returns the exit status code.
func (e *Env) fail(code int, format string, a ...any) error {
	e.diag.Errorf(format, a...)
	return &core.ExitError{Code: code}
}

// debugf traces a step when -v is in effect.
func (e *Env) debugf(format string, a ...any) {
	if e.Debug {
		e.diag.Warnf(format, a...)
	}
}

// nameOf returns the variable name of a NAME=VALUE entry. The search for '='
// skips the first byte because Windows keeps per-drive working directories in
// variables whose names begin with one ("=C:=C:\dir").
func nameOf(entry string) string {
	if entry == "" {
		return entry
	}
	if i := strings.IndexByte(entry[1:], '='); i >= 0 {
		return entry[:i+1]
	}
	if entry[0] == '=' {
		return ""
	}
	return entry
}

// set applies one NAME=VALUE assignment. As putenv does, an existing variable
// keeps its position in the environment and a new one is appended.
func set(vars []string, entry string) []string {
	name := nameOf(entry)
	for i, v := range vars {
		if sameName(nameOf(v), name) {
			vars[i] = entry
			// Drop any later duplicates so the new value is the only one.
			return append(vars[:i+1], unset(vars[i+1:], name)...)
		}
	}
	return append(vars, entry)
}

// unset removes every entry for the variable name.
func unset(vars []string, name string) []string {
	return slices.DeleteFunc(vars, func(v string) bool { return sameName(nameOf(v), name) })
}

// lookup returns the value of the variable name in vars.
func lookup(vars []string, name string) (string, bool) {
	for _, v := range vars {
		if n := nameOf(v); sameName(n, name) && len(v) > len(n) {
			return v[len(n)+1:], true
		}
	}
	return "", false
}

// resolve returns the absolute form of p as seen from dir, the directory the
// command will run in ("" for the current one). Candidates are made absolute
// because a relative program path would otherwise be interpreted differently
// by the platforms once a working directory is also set.
func resolve(dir, p string) string {
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, p)
	}
	if abs, err := fsys.Abs(p); err == nil {
		return abs
	}
	return p
}
