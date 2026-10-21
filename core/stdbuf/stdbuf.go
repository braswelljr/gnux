// Package stdbuf implements the engine behind stdbufx, a GNU-compatible
// implementation of the GNU/UNIX stdbuf command: run a command with
// different buffering of its standard streams.
package stdbuf

import (
	"errors"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/fsys"
	"github.com/braswelljr/gnux/core/internal/quotearg"
	"github.com/braswelljr/gnux/core/internal/run"
)

// Name is the command name, used as the prefix of every diagnostic.
const Name = "stdbufx"

// The exit statuses of stdbuf itself, as in GNU.
const (
	ExitCanceled     = 125 // stdbufx failed
	ExitCannotInvoke = 126 // COMMAND was found but could not be run
	ExitNotFound     = 127 // COMMAND was not found
)

// LibName is the library GNU stdbuf preloads into the command to change its
// buffering.
const LibName = "libstdbuf.so"

// Options mirrors the GNU stdbuf flag surface. At least one mode must be
// set; each is "L" for line buffering (not for Input), "0" for none, or a
// buffer size such as 4096, 64K or 1MB.
type Options struct {
	Input, Output, Error string
}

// Stdbuf is the engine. Construct it with New.
type Stdbuf struct {
	Options

	// Tool runs a gnux tool in this process, when COMMAND names one; nil
	// runs every COMMAND as a program. Getenv and Environ read the
	// environment, Executable is gnux's own path, and LibDirs are where
	// libstdbuf.so is looked for after gnux's own directory. They are the
	// injection points tests use.
	Tool       run.ToolFunc
	Getenv     func(string) string
	Environ    func() []string
	Executable func() (string, error)
	LibDirs    []string

	streams core.Streams
	diag    *core.Diag
}

// New builds a stdbuf engine running its command with the streams s.
func New(opts Options, s core.Streams) *Stdbuf {
	return &Stdbuf{
		Options:    opts,
		Getenv:     os.Getenv,
		Environ:    os.Environ,
		Executable: os.Executable,
		LibDirs:    libDirs,
		streams:    s,
		diag:       core.NewDiag(Name, s),
	}
}

func quote(s string) string { return quotearg.Quote(s, quotearg.Options{Style: quotearg.Locale}) }

// ParseMode reads a buffering mode: "L", or a size as GNU's xstrtoumax reads
// it with stdbuf's suffixes (K, M, G, ... are powers of 1024, KB, MB, ...
// powers of 1000). It returns the size, -1 for line buffering.
func ParseMode(s string) (int64, error) {
	s = strings.TrimLeft(s, " \t\n\v\f\r")
	if s == "L" {
		return -1, nil
	}
	digits := len(s) - len(strings.TrimLeft(s, "0123456789"))
	bad := errors.New("invalid")
	if digits == 0 {
		return 0, bad
	}
	n, err := strconv.ParseUint(s[:digits], 10, 63)
	if err != nil {
		return 0, errOverflow
	}
	mult := uint64(1)
	if suffix := s[digits:]; suffix != "" {
		power := strings.IndexByte("KMGTPEZYRQ", suffix[0]) + 1
		if suffix[0] == 'k' {
			power = 1
		}
		base := uint64(0)
		switch suffix[1:] {
		case "", "iB":
			base = 1024
		case "B", "D":
			base = 1000
		}
		if power == 0 || base == 0 {
			return 0, bad
		}
		for range power {
			if mult > math.MaxInt64/base {
				return 0, errOverflow
			}
			mult *= base
		}
	}
	if n != 0 && mult > math.MaxInt64/n {
		return 0, errOverflow
	}
	return int64(n * mult), nil
}

var errOverflow = errors.New("Value too large for defined data type") //nolint:revive,staticcheck // strerror(EOVERFLOW)

// Run runs COMMAND with ARGs under the buffering the options ask for. The
// result is COMMAND's own status, or 125 when stdbufx failed, 126 when the
// command could not be run and 127 when it was not found.
func (b *Stdbuf) Run(args []string) error {
	streams := []struct {
		letter byte
		mode   string
	}{{'I', b.Input}, {'O', b.Output}, {'E', b.Error}}
	var env []string
	for _, s := range streams {
		mode := strings.TrimLeft(s.mode, " \t\n\v\f\r")
		if s.mode == "" {
			continue
		}
		if s.letter == 'I' && strings.HasPrefix(mode, "L") {
			return core.Usagef("line buffering stdin is meaningless")
		}
		size, err := ParseMode(mode)
		if err != nil {
			if errors.Is(err, errOverflow) {
				b.diag.Errorf("invalid mode %s: %s", quote(mode), err)
			} else {
				b.diag.Errorf("invalid mode %s", quote(mode))
			}
			return &core.ExitError{Code: ExitCanceled}
		}
		value := "L"
		if size >= 0 {
			value = strconv.FormatInt(size, 10)
		}
		env = append(env, "_STDBUF_"+string(s.letter)+"="+value)
	}
	if len(args) == 0 {
		return core.Usagef("missing operand")
	}
	if len(env) == 0 {
		return core.Usagef("you must specify a buffering mode option")
	}

	// A gnux tool writes each record as soon as its input runs dry, which
	// is all that line buffering or no buffering would change: it runs
	// here, as it is.
	if b.Tool != nil {
		if status, found := b.Tool(args[0], args[1:], b.streams); found {
			if status != 0 {
				return &core.ExitError{Code: status}
			}
			return nil
		}
	}

	lib, ok := b.findLib()
	if !ok || preloadVar == "" {
		b.diag.Errorf("failed to find %s", quote(LibName))
		return &core.ExitError{Code: ExitCanceled}
	}
	preload := lib
	if old := b.Getenv(preloadVar); old != "" {
		preload = old + ":" + lib
	}
	env = append(env, preloadVar+"="+preload)
	env = append(env, extraEnv...)

	path, err := exec.LookPath(args[0])
	if errors.Is(err, exec.ErrDot) {
		err = nil
	}
	if err != nil {
		return b.cannotRun(args[0], err)
	}
	cmd := &exec.Cmd{Path: path, Args: args, Env: append(b.Environ(), env...),
		Stdin: b.streams.In, Stdout: b.streams.Out, Stderr: b.streams.Err}
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code := exitErr.ExitCode()
			if code < 0 {
				code = 128 + signalOf(exitErr.ProcessState)
			}
			return &core.ExitError{Code: code}
		}
		return b.cannotRun(args[0], err)
	}
	return nil
}

func (b *Stdbuf) cannotRun(name string, err error) error {
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
		b.diag.Errorf("failed to run command %s: No such file or directory", quote(name))
		return &core.ExitError{Code: ExitNotFound}
	}
	b.diag.Errorf("failed to run command %s: %s", quote(name), core.ErrText(err))
	return &core.ExitError{Code: ExitCannotInvoke}
}

// findLib looks for libstdbuf.so beside gnux, then where GNU coreutils
// installs it.
func (b *Stdbuf) findLib() (string, bool) {
	var dirs []string
	if exe, err := b.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	for _, dir := range append(dirs, b.LibDirs...) {
		p := filepath.Join(dir, LibName)
		if _, err := fsys.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}
