package tail_test

import (
	"bytes"
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/tail"
)

// run executes the engine against in-memory streams.
func run(opts tail.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = tail.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSelection(t *testing.T) {
	const five = "1\n2\n3\n4\n5\n"
	const fiveOpen = "1\n2\n3\n4\n5" // no trailing newline

	lines := func(n int64) tail.Options { return tail.Options{Count: n} }
	bytesOpt := func(n int64) tail.Options { return tail.Options{Count: n, Bytes: true} }
	fromLine := func(n int64) tail.Options { return tail.Options{Count: n, FromStart: true} }
	fromByte := func(n int64) tail.Options { return tail.Options{Count: n, Bytes: true, FromStart: true} }

	tests := []struct {
		name  string
		opts  tail.Options
		input string
		want  string
	}{
		{"empty input", lines(10), "", ""},
		{"fewer lines than asked", lines(10), five, five},
		{"last lines", lines(2), five, "4\n5\n"},
		{"last lines, unterminated", lines(2), fiveOpen, "4\n5"},
		{"last line, unterminated", lines(1), fiveOpen, "5"},
		{"exactly all lines", lines(5), five, five},
		{"exactly all lines, unterminated", lines(5), fiveOpen, fiveOpen},
		{"zero lines", lines(0), five, ""},
		{"huge line count", lines(math.MaxInt64), five, five},
		{"trailing blank lines count", lines(2), "a\n\n\n", "\n\n"},
		{"only newlines", lines(1), "\n\n", "\n"},

		{"last bytes", bytesOpt(3), five, "\n5\n"},
		{"zero bytes", bytesOpt(0), five, ""},
		{"more bytes than there are", bytesOpt(100), five, five},

		{"from line", fromLine(4), five, "4\n5\n"},
		{"from line, unterminated", fromLine(5), fiveOpen, "5"},
		{"from line one is everything", fromLine(1), five, five},
		{"from line zero is everything", fromLine(0), five, five},
		{"from a line past the end", fromLine(9), five, ""},

		{"from byte", fromByte(3), five, "2\n3\n4\n5\n"},
		{"from byte one is everything", fromByte(1), five, five},
		{"from byte zero is everything", fromByte(0), five, five},
		{"from a byte past the end", fromByte(100), five, ""},

		{"zero terminated", tail.Options{Count: 2, ZeroTerminated: true}, "a\x00b\nc\x00d\x00", "b\nc\x00d\x00"},
		{"zero terminated, from line", tail.Options{Count: 2, FromStart: true, ZeroTerminated: true}, "a\x00b\x00c", "b\x00c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Standard input exercises the streaming paths, a file the
			// seek-from-the-end ones; both must agree.
			out, errb, err := run(tt.opts, tt.input)
			if err != nil || errb != "" {
				t.Fatalf("stdin: err = %v, stderr = %q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdin: stdout = %q, want %q", out, tt.want)
			}

			out, errb, err = run(tt.opts, "", write(t, t.TempDir(), "f", tt.input))
			if err != nil || errb != "" {
				t.Fatalf("file: err = %v, stderr = %q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("file: stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

// TestLargeInputs crosses the read-chunk boundary: the backward scan needs
// several blocks and the streaming window has to release data as it goes.
func TestLargeInputs(t *testing.T) {
	var b strings.Builder
	for i := range 50000 {
		b.WriteString(strings.Repeat("y", i%37))
		b.WriteByte('\n')
	}
	input := b.String()
	all := strings.SplitAfter(input, "\n")
	all = all[:len(all)-1]

	tests := []struct {
		name  string
		opts  tail.Options
		input string
		want  string
	}{
		{"last 3", tail.Options{Count: 3}, input, strings.Join(all[len(all)-3:], "")},
		{"last 30000", tail.Options{Count: 30000}, input, strings.Join(all[20000:], "")},
		{"last 30000, unterminated", tail.Options{Count: 30000}, input + "end", strings.Join(all[20001:], "") + "end"},
		{"from line 40001", tail.Options{Count: 40001, FromStart: true}, input, strings.Join(all[40000:], "")},
		{"last 100000 bytes", tail.Options{Count: 100000, Bytes: true}, input, input[len(input)-100000:]},
		{"from byte 100001", tail.Options{Count: 100001, Bytes: true, FromStart: true}, input, input[100000:]},
	}
	dir := t.TempDir()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if out, _, err := run(tt.opts, tt.input); err != nil || out != tt.want {
				t.Fatalf("stdin: got %d bytes, want %d (err = %v)", len(out), len(tt.want), err)
			}
			if out, _, err := run(tt.opts, "", write(t, dir, "big", tt.input)); err != nil || out != tt.want {
				t.Fatalf("file: got %d bytes, want %d (err = %v)", len(out), len(tt.want), err)
			}
		})
	}
}

func TestHeaders(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "A1\nA2\n")
	b := write(t, dir, "b", "B1")

	tests := []struct {
		name  string
		opts  tail.Options
		stdin string
		args  []string
		want  string
	}{
		{"single file has no header", tail.Options{Count: 1}, "", []string{a}, "A2\n"},
		{"verbose forces one", tail.Options{Count: 1, Verbose: true}, "", []string{a}, "==> " + a + " <==\nA2\n"},
		{"verbose names stdin", tail.Options{Count: 1, Verbose: true}, "in\n", nil, "==> standard input <==\nin\n"},
		{
			"several files", tail.Options{Count: 1}, "in\n", []string{a, "-", b},
			"==> " + a + " <==\nA2\n\n==> standard input <==\nin\n\n==> " + b + " <==\nB1",
		},
		{"quiet", tail.Options{Count: 1, Quiet: true}, "", []string{a, b}, "A2\nB1"},
		{"quiet beats verbose", tail.Options{Count: 1, Quiet: true, Verbose: true}, "", []string{a, b}, "A2\nB1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, tt.stdin, tt.args...)
			if err != nil || errb != "" {
				t.Fatalf("err = %v, stderr = %q", err, errb)
			}
			if out != tt.want {
				t.Fatalf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestErrorsDoNotStopLaterOperands(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "A\n")
	b := write(t, dir, "b", "B\n")
	missing := filepath.Join(dir, "missing")

	out, errb, err := run(tail.Options{Count: 10}, "", a, missing, dir, b)
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v, want ErrFailed", err)
	}
	wantOut := "==> " + a + " <==\nA\n\n==> " + dir + " <==\n\n==> " + b + " <==\nB\n"
	if out != wantOut {
		t.Fatalf("stdout = %q, want %q", out, wantOut)
	}
	wantErr := "tailx: cannot open '" + missing + "' for reading: No such file or directory\n" +
		"tailx: error reading '" + dir + "': Is a directory\n"
	if errb != wantErr {
		t.Fatalf("stderr = %q, want %q", errb, wantErr)
	}
}

func TestUsageWarnings(t *testing.T) {
	path := write(t, t.TempDir(), "f", "x\n")

	tests := []struct {
		name    string
		opts    tail.Options
		wantErr string
	}{
		{"retry without follow", tail.Options{Count: 1, Retry: true},
			"tailx: warning: --retry ignored; --retry is useful only when following\n"},
		{"pid without follow", tail.Options{Count: 1, PID: os.Getpid()},
			"tailx: warning: PID ignored; --pid=PID is useful only when following\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errb, err := run(tt.opts, "", path)
			if err != nil || out != "x\n" {
				t.Fatalf("err = %v, stdout = %q", err, out)
			}
			if errb != tt.wantErr {
				t.Fatalf("stderr = %q, want %q", errb, tt.wantErr)
			}
		})
	}

	t.Run("stdin cannot be followed by name", func(t *testing.T) {
		out, errb, err := run(tail.Options{Count: 1, Follow: tail.FollowName}, "x\n", path, "-")
		if !errors.Is(err, core.ErrFailed) || out != "" {
			t.Fatalf("err = %v, stdout = %q", err, out)
		}
		if want := "tailx: cannot follow '-' by name\n"; errb != want {
			t.Fatalf("stderr = %q, want %q", errb, want)
		}
	})
}

// failWriter fails every write with err.
type failWriter struct{ err error }

func (w failWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWriteErrors(t *testing.T) {
	tests := []struct {
		name    string
		werr    error
		wantErr string
		failed  bool
	}{
		{"closed pipe is quiet", os.NewSyscallError("write", syscall.EPIPE), "", false},
		{"other errors are reported", errors.New("disk on fire"), "tailx: write error: Disk on fire\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errb bytes.Buffer
			err := tail.New(tail.Options{Count: 1}, core.Streams{In: strings.NewReader("in\n"), Out: failWriter{tt.werr}, Err: &errb}).
				Run([]string{"-", "-"})
			if errors.Is(err, core.ErrFailed) != tt.failed {
				t.Fatalf("err = %v, want failed = %v", err, tt.failed)
			}
			if errb.String() != tt.wantErr {
				t.Fatalf("stderr = %q, want %q", errb.String(), tt.wantErr)
			}
		})
	}
}

// --- follow mode -----------------------------------------------------------

// syncBuffer is a bytes.Buffer the engine can write while the test reads.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// follower is a tail engine running in follow mode in the background.
type follower struct {
	t      *testing.T
	out    syncBuffer
	errb   syncBuffer
	cancel context.CancelFunc
	done   chan error
}

// follow starts the engine with a short poll interval. Nothing in these tests
// sleeps for a fixed time: each step waits for the engine's observable output
// and the loop is stopped by canceling its context.
func follow(t *testing.T, opts tail.Options, args ...string) *follower {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	f := &follower{t: t, cancel: cancel, done: make(chan error, 1)}
	if opts.SleepInterval == 0 {
		opts.SleepInterval = 2 * time.Millisecond
	}
	engine := tail.New(opts, core.Streams{In: strings.NewReader(""), Out: &f.out, Err: &f.errb})
	go func() { f.done <- engine.RunContext(ctx, args) }()
	t.Cleanup(func() {
		cancel()
		<-f.done
	})
	return f
}

// waitFor blocks until cond holds, failing the test if it never does.
func (f *follower) waitFor(what string, cond func() bool) {
	f.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			f.t.Fatalf("timed out waiting for %s\nstdout: %q\nstderr: %q", what, f.out.String(), f.errb.String())
		}
		time.Sleep(time.Millisecond)
	}
}

func (f *follower) waitOut(want string) {
	f.t.Helper()
	f.waitFor("stdout "+want, func() bool { return f.out.String() == want })
}

func (f *follower) waitErr(want string) {
	f.t.Helper()
	f.waitFor("stderr "+want, func() bool { return f.errb.String() == want })
}

// stop cancels the loop and returns what RunContext returned.
func (f *follower) stop() error {
	f.t.Helper()
	f.cancel()
	return f.wait()
}

// wait returns what RunContext returned once it ends on its own.
func (f *follower) wait() error {
	f.t.Helper()
	select {
	case err := <-f.done:
		f.done <- err // keep the cleanup from blocking
		return err
	case <-time.After(10 * time.Second):
		f.t.Fatalf("follow loop did not stop\nstdout: %q\nstderr: %q", f.out.String(), f.errb.String())
		return nil
	}
}

// eventually retries a filesystem change until it succeeds. Following by name
// opens the file briefly on every poll, and on Windows an open handle makes a
// concurrent rename or removal fail.
func eventually(t *testing.T, op func() error) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		err := op()
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal(err)
		}
		time.Sleep(time.Millisecond)
	}
}

func appendTo(t *testing.T, path, s string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(s); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestFollowAppend(t *testing.T) {
	for _, mode := range []tail.FollowMode{tail.FollowDescriptor, tail.FollowName} {
		name := map[tail.FollowMode]string{tail.FollowDescriptor: "descriptor", tail.FollowName: "name"}[mode]
		t.Run(name, func(t *testing.T) {
			path := write(t, t.TempDir(), "log", "1\n2\n3\n")
			f := follow(t, tail.Options{Count: 2, Follow: mode}, path)

			f.waitOut("2\n3\n")
			appendTo(t, path, "4\n")
			f.waitOut("2\n3\n4\n")
			appendTo(t, path, "5\npartial")
			f.waitOut("2\n3\n4\n5\npartial")

			if err := f.stop(); err != nil {
				t.Fatalf("RunContext = %v, want nil", err)
			}
			if got := f.errb.String(); got != "" {
				t.Fatalf("stderr = %q, want empty", got)
			}
		})
	}
}

func TestFollowTruncation(t *testing.T) {
	for _, mode := range []tail.FollowMode{tail.FollowDescriptor, tail.FollowName} {
		name := map[tail.FollowMode]string{tail.FollowDescriptor: "descriptor", tail.FollowName: "name"}[mode]
		t.Run(name, func(t *testing.T) {
			path := write(t, t.TempDir(), "log", "old line one\nold line two\n")
			f := follow(t, tail.Options{Count: 10, Follow: mode}, path)
			f.waitOut("old line one\nold line two\n")

			// Shorter than what was there, so the shrink is visible whether
			// the poll lands before or after the new data is written.
			if err := os.Truncate(path, 0); err != nil {
				t.Fatal(err)
			}
			appendTo(t, path, "new\n")

			f.waitOut("old line one\nold line two\nnew\n")
			f.waitErr("tailx: " + path + ": file truncated\n")
			if err := f.stop(); err != nil {
				t.Fatalf("RunContext = %v, want nil", err)
			}
		})
	}
}

func TestFollowHeadersSwitchBetweenFiles(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a", "A\n")
	b := write(t, dir, "b", "B\n")
	f := follow(t, tail.Options{Count: 10, Follow: tail.FollowDescriptor}, a, b)

	want := "==> " + a + " <==\nA\n\n==> " + b + " <==\nB\n"
	f.waitOut(want)

	// b printed last, so more of b needs no header...
	appendTo(t, b, "B2\n")
	want += "B2\n"
	f.waitOut(want)

	// ...but output moving to a does.
	appendTo(t, a, "A2\n")
	want += "\n==> " + a + " <==\nA2\n"
	f.waitOut(want)

	if err := f.stop(); err != nil {
		t.Fatalf("RunContext = %v, want nil", err)
	}
}

func TestFollowNameRotation(t *testing.T) {
	dir := t.TempDir()
	path := write(t, dir, "log", "before\n")
	f := follow(t, tail.Options{Count: 10, Follow: tail.FollowName, Retry: true}, path)
	f.waitOut("before\n")

	// Rotate the log away, wait for tail to notice, then start a new one.
	eventually(t, func() error { return os.Rename(path, path+".1") })
	wantErr := "tailx: '" + path + "' has become inaccessible: No such file or directory\n"
	f.waitErr(wantErr)

	appendTo(t, path, "after\n")
	wantErr += "tailx: '" + path + "' has appeared;  following new file\n"
	f.waitErr(wantErr)
	f.waitOut("before\nafter\n")

	// The rotated file is no longer the one being followed.
	appendTo(t, path+".1", "ignored\n")
	appendTo(t, path, "more\n")
	f.waitOut("before\nafter\nmore\n")

	if err := f.stop(); err != nil {
		t.Fatalf("RunContext = %v, want nil", err)
	}
}

func TestFollowNameReplacement(t *testing.T) {
	dir := t.TempDir()
	path := write(t, dir, "log", "before\n")
	f := follow(t, tail.Options{Count: 10, Follow: tail.FollowName}, path)
	f.waitOut("before\n")

	// Swap a different file into place in one step, so the name never
	// disappears: the change is only visible as a new file identity.
	next := write(t, dir, "next", "replacement\n")
	eventually(t, func() error { return os.Rename(next, path) })
	f.waitErr("tailx: '" + path + "' has been replaced;  following new file\n")
	f.waitOut("before\nreplacement\n")

	if err := f.stop(); err != nil {
		t.Fatalf("RunContext = %v, want nil", err)
	}
}

func TestFollowNameWithoutRetryGivesUp(t *testing.T) {
	path := write(t, t.TempDir(), "log", "x\n")
	f := follow(t, tail.Options{Count: 10, Follow: tail.FollowName}, path)
	f.waitOut("x\n")

	eventually(t, func() error { return os.Remove(path) })
	// With nothing left to follow the loop ends by itself.
	if err := f.wait(); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("RunContext = %v, want ErrFailed", err)
	}
	want := "tailx: '" + path + "' has become inaccessible: No such file or directory\n" +
		"tailx: no files remaining\n"
	if got := f.errb.String(); got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestFollowRetryWaitsForMissingFile(t *testing.T) {
	for _, mode := range []tail.FollowMode{tail.FollowDescriptor, tail.FollowName} {
		name := map[tail.FollowMode]string{tail.FollowDescriptor: "descriptor", tail.FollowName: "name"}[mode]
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "log")
			f := follow(t, tail.Options{Count: 10, Follow: mode, Retry: true}, path)

			wantErr := "tailx: cannot open '" + path + "' for reading: No such file or directory\n"
			if mode == tail.FollowDescriptor {
				wantErr = "tailx: warning: --retry only effective for the initial open\n" + wantErr
			}
			f.waitErr(wantErr)

			appendTo(t, path, "hello\n")
			f.waitErr(wantErr + "tailx: '" + path + "' has appeared;  following new file\n")
			f.waitOut("hello\n")

			// The missing file was reported, so the run still counts as failed.
			if err := f.stop(); !errors.Is(err, core.ErrFailed) {
				t.Fatalf("RunContext = %v, want ErrFailed", err)
			}
		})
	}
}

func TestFollowMissingFileWithoutRetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log")
	f := follow(t, tail.Options{Count: 10, Follow: tail.FollowDescriptor}, path)

	if err := f.wait(); !errors.Is(err, core.ErrFailed) {
		t.Fatalf("RunContext = %v, want ErrFailed", err)
	}
	want := "tailx: cannot open '" + path + "' for reading: No such file or directory\n" +
		"tailx: no files remaining\n"
	if got := f.errb.String(); got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

// TestFollowUnseekableStdinEnds checks that -f on piped standard input prints
// the tail and returns, instead of polling something that cannot grow.
func TestFollowUnseekableStdinEnds(t *testing.T) {
	var out, errb bytes.Buffer
	engine := tail.New(tail.Options{Count: 1, Follow: tail.FollowDescriptor},
		core.Streams{In: strings.NewReader("a\nb\n"), Out: &out, Err: &errb})
	if err := engine.Run(nil); err != nil {
		t.Fatalf("Run = %v, want nil", err)
	}
	if out.String() != "b\n" || errb.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q", out.String(), errb.String())
	}
}

func TestFollowEndsWhenPIDDies(t *testing.T) {
	// Start and reap a short-lived process to get a PID that is surely dead.
	self, err := os.Executable()
	if err != nil {
		t.Skipf("no executable path: %v", err)
	}
	proc, err := os.StartProcess(self, []string{self, "-test.run=^$"}, &os.ProcAttr{})
	if err != nil {
		t.Skipf("cannot start a process: %v", err)
	}
	if _, err := proc.Wait(); err != nil {
		t.Fatal(err)
	}

	path := write(t, t.TempDir(), "log", "x\n")

	t.Run("dead process stops the loop", func(t *testing.T) {
		f := follow(t, tail.Options{Count: 10, Follow: tail.FollowDescriptor, PID: proc.Pid}, path)
		if err := f.wait(); err != nil {
			t.Fatalf("RunContext = %v, want nil", err)
		}
		if got := f.out.String(); got != "x\n" {
			t.Fatalf("stdout = %q, want %q", got, "x\n")
		}
	})

	t.Run("live process keeps it going", func(t *testing.T) {
		f := follow(t, tail.Options{Count: 10, Follow: tail.FollowDescriptor, PID: os.Getpid()}, path)
		f.waitOut("x\n")
		appendTo(t, path, "y\n")
		f.waitOut("x\ny\n")
		if err := f.stop(); err != nil {
			t.Fatalf("RunContext = %v, want nil", err)
		}
	})
}
