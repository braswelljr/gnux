package xargs

import (
	"bufio"
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/internal/run"
)

// runner runs the command lines of one Run and keeps score of how they ended.
type runner struct {
	trap core.Trap
	x    *Xargs
	diag *core.Diag
	// streams is what a command is given: an empty standard input unless
	// the items come from a file, and the caller's output streams, made safe
	// for concurrent writes when commands run in parallel.
	streams core.Streams
	stdin   io.Reader // the caller's standard input, for inheritStdin
	null    *os.File  // the null device standing in for standard input

	// sem bounds the commands running at once; nil runs them one at a time
	// on the calling goroutine.
	sem chan struct{}
	wg  sync.WaitGroup

	// launched counts the command lines built, declined prompts included. It
	// is only touched by the goroutine reading the input.
	launched int

	tty     *bufio.Reader // where -p answers are read from, opened on first use
	ttyFile *os.File

	mu     sync.Mutex // guards failed and fatal
	failed bool       // a command exited with a nonzero status
	fatal  int        // the status that stopped the run, 0 while it goes on
}

// newRunner prepares to run commands for x.
func newRunner(x *Xargs) *runner {
	r := &runner{x: x, streams: x.streams}

	procs := x.MaxProcs
	if procs < 0 {
		procs = MaxParallel
	}
	if procs > 1 {
		r.sem = make(chan struct{}, procs)
		// In-process tools read and write these streams from several
		// goroutines. A file takes concurrent use as it is (and a child
		// process must get the file itself, not a pipe); anything else,
		// such as a buffer, is serialized one call at a time.
		// Reads get a lock of their own: a tool waiting for input must not
		// hold up the output of the others.
		mu := &sync.Mutex{}
		r.streams.In = lockReader(r.streams.In, &sync.Mutex{})
		r.streams.Out = lockWriter(r.streams.Out, mu)
		r.streams.Err = lockWriter(r.streams.Err, mu)
	}
	r.diag = core.NewDiag(Name, r.streams)

	// GNU xargs redirects the command's standard input from /dev/null, so
	// that it cannot eat the items. The null device itself is used where
	// there is one: a child process inherits it directly.
	r.stdin = r.streams.In
	r.streams.In = emptyReader{}
	if f, err := os.Open(os.DevNull); err == nil {
		r.null, r.streams.In = f, f
	}
	return r
}

// inheritStdin gives the commands the caller's standard input, for when the
// items are read from somewhere else (-a FILE).
func (r *runner) inheritStdin() {
	r.streams.In = r.stdin
}

// close waits for the commands still running and releases what the runner
// opened.
func (r *runner) close() {
	r.wg.Wait()
	if r.null != nil {
		_ = r.null.Close()
	}
	if r.ttyFile != nil {
		_ = r.ttyFile.Close()
	}
	// A panic on a goroutine a command ran on (-P) is raised again here, on
	// the goroutine that called the engine.
	r.trap.Release()
}

// stopped reports that the run must not start another command.
func (r *runner) stopped() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.fatal != 0
}

// abort stops the run with the given exit status. The first reason wins.
func (r *runner) abort(code int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fatal == 0 {
		r.fatal = code
	}
}

// fail reports a failure of xargsx itself, which ends the run with status 1.
func (r *runner) fail(message string) {
	r.diag.Errorf("%s", message)
	r.abort(1)
}

// inputError reports why the input could not be read any further.
func (r *runner) inputError(err error) {
	var quote quoteError
	switch {
	case errors.As(err, &quote), errors.Is(err, errTooLong):
		r.fail(err.Error())
	default:
		r.fail("read error: " + core.ErrText(err))
	}
}

// result waits for the commands still running and returns the outcome of the
// run in the form Run documents.
func (r *runner) result() error {
	r.wg.Wait()
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case r.fatal == 1:
		return core.ErrFailed
	case r.fatal != 0:
		return &core.ExitError{Code: r.fatal}
	case r.failed:
		return &core.ExitError{Code: ExitCommandFailed}
	}
	return nil
}

// launch runs one command line, after printing it (-t) or asking about it
// (-p). It reports whether the run goes on. With a limit over one command at
// a time it returns as soon as the command has started, waiting first for a
// free slot.
func (r *runner) launch(argv []string) bool {
	if r.stopped() {
		return false
	}
	r.launched++

	switch {
	case r.x.Interactive:
		line := appendCommand(nil, argv)
		_, _ = r.streams.Err.Write(append(line, "?..."...))
		yes, err := r.ask(string(line))
		if err != nil {
			r.fail("failed to open " + ttyName + " for reading: " + core.ErrText(err))
			return false
		}
		if !yes {
			return true
		}
	case r.x.Verbose:
		_, _ = r.streams.Err.Write(append(appendCommand(nil, argv), '\n'))
	}

	if r.sem == nil {
		r.record(argv[0], r.command(argv))
		return !r.stopped()
	}

	r.sem <- struct{}{}
	if r.stopped() {
		// A command failed for good while this one waited for its slot.
		<-r.sem
		return false
	}
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer func() { <-r.sem }()
		defer r.trap.Catch()
		r.record(argv[0], r.command(argv))
	}()
	return true
}

// outcome is how one command ended.
type outcome struct {
	res run.Result
	err error
}

// command runs argv and waits for it.
func (r *runner) command(argv []string) outcome {
	res, err := run.Command(r.x.Exec, argv, "", r.streams)
	return outcome{res, err}
}

// record folds the outcome of the command called name into the run's status.
func (r *runner) record(name string, o outcome) {
	switch {
	case o.err != nil:
		r.diag.Errorf("%s: %s", name, core.ErrText(o.err))
		if errors.Is(o.err, fs.ErrNotExist) {
			r.abort(ExitNotFound)
		} else {
			r.abort(ExitCannotRun)
		}
	case o.res.Signaled:
		if o.res.Signal > 0 {
			r.diag.Errorf("%s: terminated by signal %d", name, o.res.Signal)
		} else {
			r.diag.Errorf("%s: terminated by a signal", name)
		}
		r.abort(ExitSignaled)
	case o.res.Status == 255:
		r.diag.Errorf("%s: exited with status 255; aborting", name)
		r.abort(ExitAborted)
	case o.res.Status != 0:
		r.mu.Lock()
		r.failed = true
		r.mu.Unlock()
	}
}

// ask reads the answer to a -p prompt: yes when it starts with 'y' or 'Y'.
// The answer comes from the terminal, not from standard input, which carries
// the items; the error is that of opening the terminal.
func (r *runner) ask(command string) (bool, error) {
	if r.x.PromptFn != nil {
		return r.x.PromptFn(command), nil
	}
	if r.tty == nil {
		f, err := os.Open(ttyName)
		if err != nil {
			return false, err
		}
		r.ttyFile, r.tty = f, core.NewReader(f)
	}
	line, _ := r.tty.ReadString('\n')
	return line != "" && (line[0] == 'y' || line[0] == 'Y'), nil
}

// appendCommand appends the command line argv to dst as -t and -p print it:
// the arguments separated by spaces, each quoted for the shell where it needs
// to be.
func appendCommand(dst []byte, argv []string) []byte {
	for i, a := range argv {
		if i > 0 {
			dst = append(dst, ' ')
		}
		dst = appendQuoted(dst, a)
	}
	return dst
}

// appendQuoted appends arg to dst in the "shell-escape" style GNU prints
// arguments in: as it is when the shell would read it back unchanged, in
// single quotes otherwise, with control characters as $'...' escapes.
func appendQuoted(dst []byte, arg string) []byte {
	if !needsQuoting(arg) {
		return append(dst, arg...)
	}
	const octal = "01234567"
	dst = append(dst, '\'')
	escaping := false // inside a $'...' section
	for i := 0; i < len(arg); i++ {
		c := arg[i]
		if c >= ' ' && c != 0x7f {
			if escaping {
				dst = append(dst, "''"...)
				escaping = false
			}
			if c == '\'' {
				dst = append(dst, `'\''`...)
			} else {
				dst = append(dst, c)
			}
			continue
		}
		if !escaping {
			dst = append(dst, "'$'"...)
			escaping = true
		}
		if at := strings.IndexByte("\a\b\f\n\r\t\v", c); at >= 0 {
			dst = append(dst, '\\', "abfnrtv"[at])
		} else {
			dst = append(dst, '\\', octal[c>>6], octal[c>>3&7], octal[c&7])
		}
	}
	return append(dst, '\'')
}

// needsQuoting reports whether the shell would read arg as anything but the
// one word it is.
func needsQuoting(arg string) bool {
	switch arg {
	case "", "{", "}":
		return true
	}
	if arg[0] == '#' || arg[0] == '~' {
		return true
	}
	for i := 0; i < len(arg); i++ {
		c := arg[i]
		if c <= ' ' || c == 0x7f || strings.IndexByte("!\"$&'()*;<=>?[\\^`|", c) >= 0 {
			return true
		}
	}
	return false
}

// emptyReader is a standard input with nothing on it, for where the null
// device cannot be opened. It has no state, so parallel commands can share it.
type emptyReader struct{}

func (emptyReader) Read([]byte) (int, error) { return 0, io.EOF }

// lockedWriter serializes the writes of parallel commands to one stream.
type lockedWriter struct {
	mu *sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// lockWriter returns w made safe for concurrent writes: w itself when it is a
// file, a serializing wrapper otherwise.
func lockWriter(w io.Writer, mu *sync.Mutex) io.Writer {
	if _, ok := w.(*os.File); ok {
		return w
	}
	return &lockedWriter{mu: mu, w: w}
}

// lockedReader serializes the reads of parallel commands from one stream.
type lockedReader struct {
	mu *sync.Mutex
	r  io.Reader
}

func (l *lockedReader) Read(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.r.Read(p)
}

// lockReader is lockWriter for a reader.
func lockReader(r io.Reader, mu *sync.Mutex) io.Reader {
	if _, ok := r.(*os.File); ok {
		return r
	}
	return &lockedReader{mu: mu, r: r}
}
