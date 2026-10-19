package cmd

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/braswelljr/gnux/core"
)

// textBlock is 64 KiB of distinct lines, the unit the generated inputs of the
// robustness tests and the benchmarks are made of.
var textBlock = func() []byte {
	var b []byte
	for i := 0; len(b) < 64<<10; i++ {
		b = fmt.Appendf(b, "line %06d: the quick brown fox jumps over the lazy dog\tend\n", i)
	}
	return b
}()

// generated is an input of size bytes of text that is never held in memory
// whole: textBlock over and over. Before each Read it calls sample, if set.
type generated struct {
	size, off int
	sample    func()
}

func (g *generated) Read(p []byte) (int, error) {
	if g.sample != nil {
		g.sample()
	}
	if g.off >= g.size {
		return 0, io.EOF
	}
	n := copy(p, textBlock[g.off%len(textBlock):])
	n = min(n, g.size-g.off)
	g.off += n
	return n, nil
}

// streaming lists command lines that must handle any amount of input in a
// fixed amount of memory: filters, counters, digests and encoders.
// Tools that need the whole input by nature (sortx, tacx, shufx) are not here.
var streaming = []string{
	"cat", "cat -n", "cat -A", "tee", "wc", "wc -l", "head -n 3", "head -c 1M", "tail -n 3", "tail -c 100",
	"grep fox", "grep -c lazy", "grep -v zebra", "grep -n -E 0{3}9", "grep -F -o brown",
	"sed s/quick/slow/", "sed -n $p", "sed -E s/(fox|dog)/[\\1]/g", "sed y/abc/xyz/",
	"cut -c1-10", "cut -f2", "tr a-z A-Z", "tr -d o", "uniq", "uniq -c", "nl", "expand", "unexpand -a",
	"fold -w 20", "fmt", "paste - -", "od", "od -c", "base64", "base32", "cksum", "sum", "sum -s", "md5sum", "sha256sum", "b2sum",
	"dd status=none", "xargs -n 50 true", "xargs true",
}

// The memory test's input, and how far a streaming tool may grow the live heap
// while reading it. The tools stay under half a mebibyte, whatever the input;
// a tool that keeps what it reads grows the heap by about the input's size.
// That gap is why a small input is enough, which keeps the test quick under
// the race detector.
const (
	memoryInput   = 4 << 20
	memoryAllowed = 1 << 20
)

// heapGrowth runs a command line over memoryInput bytes of generated text and
// returns how much the live heap (what a garbage collection leaves) grew at
// its largest while the tool was reading.
func heapGrowth(t *testing.T, line string) uint64 {
	t.Helper()
	argv := strings.Fields(line)
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	base, peak := m.HeapAlloc, m.HeapAlloc

	reads := 0
	in := &generated{size: memoryInput}
	in.sample = func() {
		// A sample collects first, so that it counts what the tool still
		// holds and not garbage awaiting collection. That is not free: once
		// every few reads is plenty to see a heap that grows with the input.
		if reads++; reads%8 != 0 {
			return
		}
		runtime.GC()
		runtime.ReadMemStats(&m)
		peak = max(peak, m.HeapAlloc)
	}
	status, ok := RunTool(argv[0], argv[1:], core.Streams{In: in, Out: io.Discard, Err: io.Discard})
	if !ok || status != 0 {
		t.Fatalf("%s: status = %d, found = %v", line, status, ok)
	}
	return peak - min(peak, base)
}

// TestStreamingToolsUseBoundedMemory runs each streaming tool over an input
// larger than the memory it is allowed. A tool that keeps what it has read
// fails; one that streams stays near its buffers.
func TestStreamingToolsUseBoundedMemory(t *testing.T) {
	for _, line := range streaming {
		t.Run(line, func(t *testing.T) {
			if grown := heapGrowth(t, line); grown > memoryAllowed {
				t.Errorf("the heap grew by %d KiB while reading %d KiB; a streaming tool is allowed %d KiB",
					grown>>10, memoryInput>>10, memoryAllowed>>10)
			}
		})
	}
}

// TestMemoryTestSeesAToolThatKeepsItsInput is the control for the test above:
// sortx and tacx must hold their whole input, and the measurement has to show
// it. If it ever stops doing so, a passing streaming test proves nothing.
func TestMemoryTestSeesAToolThatKeepsItsInput(t *testing.T) {
	for _, line := range []string{"sort", "tac"} {
		if grown := heapGrowth(t, line); grown < 2*memoryAllowed {
			t.Errorf("%s grew the heap by only %d KiB while reading %d KiB: the measurement would miss a tool that keeps its input",
				line, grown>>10, memoryInput>>10)
		}
	}
}

// TestNoGoroutineIsLeftBehind runs the commands that start goroutines (shell
// pipelines, including ones whose reader stops early, and parallel xargsx)
// and checks that every one of them has ended afterwards. A host that runs
// many scripts in one process (the playground, an embedding program) would
// otherwise collect them.
func TestNoGoroutineIsLeftBehind(t *testing.T) {
	scripts := []string{
		"seq 1 2000 | sort -rn | head -n 3 | wc -l",
		"yes | head -n 2",
		"yes abc | grep -m 1 abc",
		"seq 1 100000 | sed 5q | cat -n | tail -n 1",
		"seq 1 50000 | xargs -n 100 -P 4 true",
		"seq 1 200 | xargs -P 8 -I {} echo {} | wc -l",
		"seq 1 100000 | head -n 1 | nosuchtool; true",
		"echo a | cat | cat | cat | cat | cat",
	}
	run := func() {
		for _, script := range scripts {
			s := core.Streams{In: strings.NewReader(""), Out: io.Discard, Err: io.Discard}
			if status := Main([]string{"gnux", "shell", "-c", script}, s); status != 0 {
				t.Fatalf("%q: status %d", script, status)
			}
		}
	}

	run() // start whatever starts once per process
	before := settledGoroutines()
	for range 5 {
		run()
	}
	if after := settledGoroutines(); after > before {
		buf := make([]byte, 1<<16)
		t.Fatalf("%d goroutines before, %d after:\n%s", before, after, buf[:runtime.Stack(buf, true)])
	}
}

// settledGoroutines returns the number of goroutines once it has stopped
// falling: a stage that has just been told to stop needs a moment to do so.
func settledGoroutines() int {
	n := runtime.NumGoroutine()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		time.Sleep(20 * time.Millisecond)
		runtime.GC()
		if now := runtime.NumGoroutine(); now >= n {
			return now
		} else {
			n = now
		}
	}
	return n
}

// lineFilters lists command lines that turn each input line into output
// without needing the next one, so they can answer a line at a time.
var lineFilters = []string{
	"cat", "cat -n", "cat -A", "tee", "nl", "cut -c1-3", "cut -f1", "tr a-z A-Z", "expand", "unexpand -a",
	"fold -w 40", "paste -", "head -n 5", "grep line", "grep -n -v zebra", "sed s/line/LINE/", "sed -n p",
	"xargs -n 1 echo", "xargs -I {} echo {}", "factor",
}

// lineInput is the line a filter in lineFilters is given when "line one"
// is not something it accepts.
var lineInput = map[string]string{
	"factor": "12\n",
}

// watched is an output a test can wait on.
type watched struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	seen chan struct{}
}

func (w *watched) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.buf.Len() == 0 && len(p) > 0 {
		close(w.seen)
	}
	return w.buf.Write(p)
}

// TestLineFiltersAnswerEachLine gives each filter one line and leaves its
// input open, as a terminal or a slow producer (tail -f, a build log) does.
// The output for that line must arrive without waiting for more input: a
// filter that sat on it until its buffer filled would look hung.
func TestLineFiltersAnswerEachLine(t *testing.T) {
	for _, line := range lineFilters {
		t.Run(line, func(t *testing.T) {
			argv := strings.Fields(line)
			in, feed := io.Pipe()
			out := &watched{seen: make(chan struct{})}
			done := make(chan int, 1)
			go func() {
				status, _ := RunTool(argv[0], argv[1:], core.Streams{In: in, Out: out, Err: io.Discard})
				done <- status
			}()

			input, ok := lineInput[line]
			if !ok {
				input = "line one\n"
			}
			if _, err := feed.Write([]byte(input)); err != nil {
				t.Fatal(err)
			}
			select {
			case <-out.seen:
			case <-time.After(3 * time.Second):
				t.Error("no output for the first line while the input stayed open")
			}
			_ = feed.Close()
			if status := <-done; status != 0 {
				t.Errorf("status = %d", status)
			}
		})
	}
}
