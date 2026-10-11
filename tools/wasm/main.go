//go:build js && wasm

// Command wasm is the WebAssembly build of gnux that powers the web
// playground. It is not a command-line program: once started it installs a
// small API on the JavaScript global object and then waits to be called.
//
//	gnux.version               string - the gnux version
//	gnux.tools                 string[] - every tool name
//	gnux.run(line)             Promise<number> - run one shell line, resolve its exit status
//	gnux.cwd()                 string - the shell's working directory
//	gnux.onOutput = (fd, chunk) => …   called with 1 (stdout) or 2 (stderr) and a Uint8Array
//
// Lines run in one persistent gnux shell, so variables and cd carry over from
// call to call. The tools reach the filesystem through the Node-style `fs` and
// `process` objects Go's js/wasm runtime expects on the global object; the
// playground supplies in-memory ones, so nothing touches the visitor's disk.
//
// When the API is ready, the function globalThis.gnuxReady is called, if set.
//
// Build with `make web/wasm`.
package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"syscall/js"

	"github.com/braswelljr/gnux/cmd"
	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/shell"
	"github.com/braswelljr/gnux/internal/common"
)

// outputBudget caps how much one line may print. The browser runs Go on its
// only thread, so an endless writer (yesx with no pipe) would otherwise freeze
// the page; past the cap, writes fail like a closed pipe and the tool stops.
const outputBudget = 1 << 20

// output is the stdout/stderr pair handed to the shell, sharing one budget.
type output struct {
	api       js.Value
	remaining int
	truncated bool
}

// writer is one of the two streams of an output.
type writer struct {
	out *output
	fd  int
}

func (w writer) Write(p []byte) (int, error) {
	if w.out.remaining <= 0 {
		w.out.truncated = true
		return 0, syscall.EPIPE
	}
	n := min(len(p), w.out.remaining)
	w.out.remaining -= n
	w.out.emit(w.fd, p[:n])
	if n < len(p) {
		w.out.truncated = true
		return n, syscall.EPIPE
	}
	return n, nil
}

// emit hands a chunk to the page's onOutput callback.
func (o *output) emit(fd int, p []byte) {
	fn := o.api.Get("onOutput")
	if fn.Type() != js.TypeFunction {
		return
	}
	chunk := js.Global().Get("Uint8Array").New(len(p))
	js.CopyBytesToJS(chunk, p)
	fn.Invoke(fd, chunk)
}

func main() {
	api := js.Global().Get("Object").New()
	out := &output{api: api}
	sh := cmd.NewShell(
		// Interactive: a syntax error must not end the session.
		shell.Options{Interactive: true},
		core.Streams{In: strings.NewReader(""), Out: writer{out, 1}, Err: writer{out, 2}},
	)

	var tools []any
	for _, t := range cmd.Tools() {
		tools = append(tools, t.Name())
	}
	api.Set("version", common.Version)
	api.Set("tools", js.ValueOf(tools))

	api.Set("cwd", js.FuncOf(func(js.Value, []js.Value) any {
		wd, err := os.Getwd()
		if err != nil {
			return "/"
		}
		return wd
	}))

	// One line at a time: the shell is a single session with shared state.
	var mu sync.Mutex
	api.Set("run", js.FuncOf(func(_ js.Value, args []js.Value) any {
		line := ""
		if len(args) > 0 {
			line = args[0].String()
		}
		// Filesystem calls complete on the JavaScript event loop, so the work
		// must happen on a goroutine while this callback returns a Promise.
		executor := js.FuncOf(func(_ js.Value, settle []js.Value) any {
			resolve := settle[0]
			go func() {
				mu.Lock()
				defer mu.Unlock()
				out.remaining, out.truncated = outputBudget, false
				status := sh.Eval(line)
				if out.truncated {
					out.emit(2, fmt.Appendf(nil, "%s: output truncated after %d bytes\n", common.AppName, outputBudget))
				}
				resolve.Invoke(status)
			}()
			return nil
		})
		defer executor.Release()
		return js.Global().Get("Promise").New(executor)
	}))

	js.Global().Set(common.AppName, api)
	if ready := js.Global().Get("gnuxReady"); ready.Type() == js.TypeFunction {
		ready.Invoke(api)
	}

	// Stay alive to serve calls; the page ends the program by dropping it.
	select {}
}
