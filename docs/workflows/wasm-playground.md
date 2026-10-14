# WebAssembly Playground

How the website's playground runs the real gnux in a browser: `make web/wasm` compiles `tools/wasm` for `js/wasm`, the page boots it against an in-memory POSIX filesystem (`web/lib/playground/memfs.ts`) that it installs as the Node-style `fs`, `path` and `process` objects Go's runtime expects, and the program publishes a small JavaScript API (`gnux.run`, `gnux.cwd`, `gnux.onOutput`, `gnux.version`, `gnux.tools`). A line typed in the terminal goes through `useTerminal`, `GnuxSession.run`, the Go `run` callback and `shell.Eval` to `cmd.RunTool`, exactly the path a command line takes on a real machine. It covers the output budget that keeps a runaway command from freezing the page, how Reset starts a fresh instance, and the Node test suite in `web/test` that drives the real `gnux.wasm`.

## The pieces

| Piece                              | File                                     | Role                                                            |
|------------------------------------|------------------------------------------|-----------------------------------------------------------------|
| The Go program                     | `tools/wasm/main.go` (`//go:build js && wasm`) | Builds a gnux shell, publishes the API, waits forever     |
| Go's loader                        | `web/public/wasm_exec.js` (copied from `$GOROOT/lib/wasm`) | Defines `globalThis.Go`, bridges syscalls to `fs`/`process` |
| The in-memory filesystem           | `web/lib/playground/memfs.ts`            | `MemFS`: files, directories, links, permissions, limits         |
| The starting files                 | `web/lib/playground/seed.ts`             | `createSeededFS()`, `HOME = /home/guest`, `ENV`                 |
| Booting, environment-agnostic      | `web/lib/playground/runtime.ts`          | `bootGnux(options)` returns a `GnuxSession`                     |
| Booting, in a browser              | `web/lib/playground/browser.ts`          | `bootInBrowser()`: fetches the loader and the module            |
| React state                        | `web/src/hooks/use-gnux.ts`, `use-terminal.ts` | One instance per generation; scrollback, history, `run`   |
| The page                           | `web/src/components/playground/playground.tsx` | Terminal, side panel, examples, completion, `/` commands, full screen |
| Tests                              | `web/test/*.test.ts`, `web/test/harness.ts` | The real `gnux.wasm` under `node --test`                     |

## The path of one line

```mermaid
sequenceDiagram
    autonumber
    participant U as Visitor
    participant T as useTerminal.run
    participant S as GnuxSession.run (runtime.ts)
    participant G as gnux.run (tools/wasm/main.go)
    participant E as shell.Eval
    participant RT as cmd.RunTool
    participant FS as MemFS (globalThis.fs)
    U->>T: Enter "lsx -la"
    T->>T: busy? ignore; "clear"? clear scrollback
    T->>T: print command entry with session.cwd()
    T->>S: await session.run(line)
    S->>G: api.run(line)
    G-->>S: Promise (executor starts a goroutine)
    G->>G: mu.Lock(); reset output budget
    G->>E: sh.Eval(line)
    E->>RT: parse, expand, Exec(name, args, streams)
    RT->>FS: os.Lstat, os.ReadDir ... via syscall/js
    FS-->>RT: callbacks (synchronous)
    RT-->>G: writes to writer{out, 1|2}
    G-->>T: onOutput(fd, Uint8Array) per chunk
    E-->>G: status
    G->>G: if truncated, emit notice on fd 2
    G-->>S: resolve(status)
    S-->>T: number
    T->>T: "[exit status N]" if non-zero; setCwd(session.cwd())
```

## Step 1: building the module

```makefile
web/wasm:
	GOOS=js GOARCH=wasm go build -trimpath $(LDFLAGS) -o $(WEB_DIR)/public/gnux.wasm ./tools/wasm
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" $(WEB_DIR)/public/wasm_exec.js
```

- **Build tags decide the platform code.** `js` is neither `unix` nor `windows`, so
  `core/fs_other.go` (`!unix`) supplies `Writable` (owner write bit) and `DeviceID`
  (unsupported), and `core/pipe_other.go` (`!windows`) supplies `IsClosedPipe`. Every tool must
  compile here, which the Test workflow checks with `GOOS=js GOARCH=wasm go build -o /dev/null
  ./tools/wasm` (NFR-3).
- **The loader is copied from the same toolchain** that built the module. `wasm_exec.js` and the
  Go runtime must match, so neither is committed (`.gitignore` lists both) and both are rebuilt
  together.
- `LDFLAGS` injects the version, which the page shows in its banner.

## Step 2: the Go program (`tools/wasm/main.go`)

`main` does four things and then parks:

1. **Builds one shell** with `cmd.NewShell`, so its commands are the registered tools reached
   through `cmd.RunTool`, as everywhere else:

   ```go
   sh := cmd.NewShell(
   	// Interactive: a syntax error must not end the session.
   	shell.Options{Interactive: true},
   	core.Streams{In: strings.NewReader(""), Out: writer{out, 1}, Err: writer{out, 2}},
   )
   ```

   Standard input is always empty: a browser terminal has no stdin stream to hand a tool, so
   `catx` with no operand reads end of file at once.

2. **Publishes the API** on a new JavaScript object:

   | Member                  | Type                                  | Implementation                                        |
   |-------------------------|---------------------------------------|-------------------------------------------------------|
   | `version`               | `string`                              | `common.Version`                                      |
   | `tools`                 | `string[]`                            | the names from `cmd.Tools()`                          |
   | `cwd()`                 | `string`                              | `os.Getwd()`, or `/` on error                         |
   | `run(line)`             | `Promise<number>`                     | `sh.Eval(line)` on a goroutine                        |
   | `onOutput = (fd, chunk)`| set by the page                       | called with `1` or `2` and a `Uint8Array`             |

3. **Sets `globalThis.gnux`** to that object and calls `globalThis.gnuxReady(api)` if the page
   defined it.
4. **`select {}`**: the program stays alive to serve calls. The page ends it by dropping it.

### `run`: a Promise around `Eval`

```go
api.Set("run", js.FuncOf(func(_ js.Value, args []js.Value) any {
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
				out.emit(2, []byte(fmt.Sprintf("%s: output truncated after %d bytes\n", common.AppName, outputBudget)))
			}
			resolve.Invoke(status)
		}()
		return nil
	})
	defer executor.Release()
	return js.Global().Get("Promise").New(executor)
}))
```

Why it is shaped like this:

- **A goroutine, not the callback.** A `js.FuncOf` callback runs on the JavaScript event loop's
  turn; blocking it on a filesystem call or a timer (`sleepx`) would deadlock the Go scheduler
  against the loop that has to deliver the result. Returning a Promise and doing the work on a
  goroutine lets the loop keep turning.
- **`mu` serializes lines.** The shell is one session with shared state (variables, `$?`, the
  working directory), so two overlapping `run` calls must not interleave. The UI also refuses to
  start a line while one is running (`busy` in `useTerminal`).
- **`Eval`, not `Run`.** `Eval` keeps variables, `$?` and the working directory between calls and
  treats `exit` as ending only the line (FR-81, FR-85). See
  [shell-execution](shell-execution.md#step-3-run-reads-lines-eval-runs-one).
- **`Interactive: true`.** A syntax error reports status 2 and leaves the session usable.

### The output budget (NFR-11)

```go
const outputBudget = 1 << 20 // 1 MiB per line, stdout and stderr together

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
```

The browser runs Go on its only thread, so `yesx` with nothing downstream would never yield and
the tab would hang. Past the budget a write fails with `EPIPE`, the error every streaming tool
already treats as "the reader has gone, stop quietly" (see
[error-handling-and-exit-status](error-handling-and-exit-status.md#closed-pipes)). The tool
stops, and `run` appends `gnux: output truncated after 1048576 bytes` on stderr. The notice goes
through `emit` directly, so it is never itself cut off.

The budget limits the terminal streams only. A redirection writes to the in-memory filesystem,
which has its own limits (below).

## Step 3: the in-memory filesystem (`memfs.ts`)

Go's `js/wasm` runtime implements `os` and `syscall` by calling a Node-style `fs` object and a
`process` object on the global scope (see `$GOROOT/src/syscall/fs_js.go`). `MemFS` provides
both, so the tools see an ordinary POSIX filesystem and nothing reaches the visitor's disk
(NFR-19):

- **`nodeFs()`** is the callback surface: `open`, `read`, `write`, `stat`, `lstat`, `mkdir`,
  `readdir`, `rename`, `symlink`, `link`, `chmod`, `chown`, `utimes`, `truncate`, `fsync` and the
  rest. Each wraps a synchronous core method and calls the callback **synchronously**, outside
  the `try`, so an exception thrown by the callback is never mistaken for a filesystem error and
  reported twice.
- **`nodeProcess()`** gives `cwd`, `chdir`, `umask`, uid/gid (a single user, 1000) and an empty
  `env`. The shell's `cd` therefore moves `MemFS`'s working directory.
- **`nodePath()`** gives `resolve`, the only `path` function Go calls.
- **Errors are `FsError`s with a Node-style `code`** (`ENOENT`, `EACCES`, `ENOTEMPTY`, ...).
  Every code must be one Go's `errnoByCode` table knows, or the Go runtime panics; the
  `ErrorCode` union lists exactly the ones used. Go turns them into `syscall.Errno`, and
  `core.ErrText` turns those into GNU's wording, so diagnostics in the browser read exactly as
  on Linux.
- **Semantics it implements**: regular files, directories, symbolic links (followed in all but
  the last component; `ELOOP` after 40), hard links with link counts, permission bits checked for
  the one user, a umask (`0o022`), timestamps, `O_APPEND`/`O_TRUNC`/`O_EXCL`, and open files
  that outlive their last name.
- **Descriptors 0, 1, 2.** Reading fd 0 is always end of file. Writes to fd 1 and 2 go to the
  browser console line by line: only the Go runtime itself writes there (a panic, `println`),
  because the tools write through the shell's `writer`, never through fd 1 or 2.
- **Limits.** `maxBytes` (default 64 MiB of file data) and `maxInodes` (default 100,000).
  Exceeding either fails with `ENOSPC`, which reaches the tool as `No space left on device`.

`createSeededFS()` in `seed.ts` builds the starting tree as root (`asRoot`): `/home`, `/tmp`
(mode `1777`), and `/home/guest` owned by the user, then, as the user, `README.txt`,
`notes/`, `src/` (including a `build.gnux` script), `logs/app.log` and a symbolic link
`latest.log -> logs/app.log`, with the working directory set to `HOME`. `ENV` sets `HOME`,
`USER`, `LOGNAME`, `SHELL`, `TMPDIR`, `LANG` and `TERM`.

## Step 4: booting (`runtime.ts`, `browser.ts`)

`bootGnux({ fs, env, loadGoRuntime, instantiate })` is written so the same code runs in the
browser and under `node --test`: the caller supplies how to load `wasm_exec.js` and how to
instantiate the module.

1. Install `globalThis.fs` and `globalThis.path` from the `MemFS`.
2. If `globalThis.Go` is not defined yet, `loadGoRuntime()`. The loader installs a stub
   `process` when there is none; `bootGnux` removes that stub afterwards so it cannot confuse
   unrelated code on the page.
3. `new Go()`, with `argv = ['gnux']`, `env = { ...ENV }`, and an `exit` hook that records an
   early exit.
4. `instantiate(go.importObject)`.
5. **Swap `globalThis.process` for the `MemFS` one only for the duration of `go.run`.** Go reads
   `fs` and `process` once during start-up, which runs synchronously inside `go.run` until
   `main` parks in `select {}`. Swapping just for that window keeps the page's own `process`
   (or Node's, in tests) intact everywhere else. The globals are reassigned at this point
   because another boot may have replaced them while the module was downloading.
6. Read `globalThis.gnux` and delete it, then validate it with `readApi`: the object comes from
   another program, so `run` and `cwd` must be functions, `version` falls back to `'unknown'`
   and `tools` keeps only strings.
7. Return a `GnuxSession`: `version`, `tools`, `fs`, `run(line)` (a non-number status becomes
   1), `cwd()`, `onOutput(handler)`, and `dispose()`, which disconnects output and clears Go's
   pending timeouts (a `sleepx` in flight).

`bootInBrowser()` in `browser.ts` supplies the browser halves: a `<script src="/wasm_exec.js">`
added once per page, and `instantiate`, which compiles while downloading with
`WebAssembly.instantiateStreaming` and falls back to a buffer when the server's
`Content-Type` prevents streaming. The compiled `WebAssembly.Module` is kept, so `/refresh` starts a
fresh instance without downloading again. The only network requests are for these two files
(NFR-19).

## Step 5: the React side

- **`useGnux()`** boots one instance per `generation`. If the effect is cleaned up before the
  boot finishes (a fast `/refresh`, or React's development double mount), the late session is
  disposed instead of used. A failed boot shows its message (for example "This browser does not
  support WebAssembly." or "Could not download /gnux.wasm (HTTP 404)").
- **`reset()`** sets the state back to loading and increments `generation`. The effect's cleanup
  disposes the old session and the new effect boots a new `MemFS` and a new Go instance: every
  file comes back and all shell state is gone. The old instance is simply dropped, since a page
  cannot terminate a Go program.
- **`useTerminal(session, banner)`** owns the scrollback, history and working directory.
  `onOutput` chunks are decoded with one streaming `TextDecoder` per stream (so a multi-byte
  character split across chunks survives) and flushed into React state at most once per animation
  frame, because gnux can emit thousands of chunks before it yields. Scrollback is capped at 500
  entries of at most 200,000 characters each, trimmed at a line boundary with
  `[earlier output trimmed]`.
- **`run(line)`** ignores a line while one is running, handles `clear` itself (it belongs to the
  terminal, not the shell), prints the command with the current directory, awaits
  `session.run`, prints `[exit status N]` for a non-zero status, and refreshes the prompt's
  directory from `session.cwd()`. If the Go program itself throws, it prints
  `gnux stopped unexpectedly: ...` and suggests `/refresh`.
- **`/` commands** (`lib/playground/commands.ts`) belong to the page and never reach gnux.
  `Workspace` in `playground.tsx` reads a line with `parseSlash`; a `/` line is echoed into the
  scrollback and history by `useTerminal`'s `echo`, then handled: `/help` prints `helpText`
  (the playground commands, every tool in the running build with the description the docs give
  it, passed in from the page, and the keys), `/examples [N]`, `/docs [TOOL]` (a new tab),
  `/history`, `/clear`, `/fullscreen` and `/refresh` (which calls `reset()`). An unknown `/word`
  is an error, never a shell command. `/` lines are not put in `?cmd=`.
- **Suggestion.** With the caret at the end of the line, the terminal shows greyed the rest of
  the newest history entry that starts with what is typed, else what completion would add when
  it has a single answer (`suggest`). Right arrow, or Tab when completion has nothing to add,
  takes it. `completeSlash` completes `/` command names before the shell's completion runs.
- **Full screen** (`useFullscreen`) asks the browser for full screen on the workspace and also
  lays the terminal over the page with `position: fixed`, so it works where the Fullscreen API
  does not. Esc, the title-bar button or `/fullscreen` leaves it.
- **A command that never returns** cannot be interrupted, since a WebAssembly call holds the
  thread. While one runs, Esc or Ctrl+C in the prompt calls `reset()`. A loop is the one such
  command a visitor can write in the shell itself, so `tools/wasm` sets `Shell.LoopLimit`
  (`loopLimit`, 10000 turns): `while truex; do truex; done` ends with
  `gnux: loop stopped after 10000 iterations`.
- **Completion** (`complete.ts`) completes command names (the tools plus the builtins) in
  command position and file names from the `MemFS` elsewhere, escaping special characters with
  a backslash.
- **The home page's terminal** is a second, smaller host of the same session. Its animation is
  a picture, but once a visitor takes the keyboard (`HeroStage`, `components/home`),
  `useGnux(taken)` boots gnux exactly as the playground does and `HeroSession` runs each
  entered line through `useTerminal`. `useGnux(false)` downloads nothing, so a visitor who never
  takes the keyboard never fetches the module.
- **A command in the URL** (`?cmd=...`) is sanitized to one printable line of at most 500
  characters and placed in the prompt. It is never run on arrival, because anyone can write a
  link.

## Step 6: the tests (`web/test`)

`web/test/harness.ts` boots the real `public/gnux.wasm` in Node through the same `bootGnux`,
evaluating `wasm_exec.js` with `vm.runInThisContext` and compiling the module once for all
tests. `boot()` returns a `Harness` whose `run(line)` collects stdout and stderr per line.

| Suite                  | What it covers                                                                       | Requirement |
|------------------------|--------------------------------------------------------------------------------------|-------------|
| `wasm.test.ts`         | "publishes its API"; dispatch by GNU name; `cd` and variables carrying over; globbing; pipelines; the connectors; diagnostics and statuses; each tool against the virtual filesystem; "runaway output is cut off instead of hanging"; "filling the filesystem fails cleanly"; "sessions are independent" | FR-47, FR-84, FR-85, NFR-11, NFR-33 |
| `memfs.test.ts`        | descriptor offsets, `O_APPEND`/`O_TRUNC`/`O_EXCL`, links, permissions, umask, `ENOSPC`, the callback surface | NFR-19, NFR-33 |
| `complete.test.ts`     | command and file completion; ANSI escapes                                            | FR-48       |
| `rules.test.ts`        | house rules over the web source, including "no em dashes in anything we write"       | NFR-42      |

Two of those tests, verbatim:

```ts
test('runaway output is cut off instead of hanging', async () => {
  const r = await h.run('yesx');
  assert.match(r.stderr, /output truncated/);
  assert.ok(r.stdout.length <= 1 << 20);
});

test('filling the filesystem fails cleanly', async () => {
  const r = await h.run('yesx > big');
  assert.equal(r.status, 1);
  assert.match(r.stderr, /No space left on device/);
  ...
});
```

Run them locally with `make web/test`, which rebuilds the wasm first (the tests are meaningless
against a stale module), then `pnpm test` in `web/`. The Web workflow runs the same after
`make web/wasm`; see [ci-and-release](ci-and-release.md).

## Things that differ from a real machine

| Behavior                               | Why                                                                       |
|----------------------------------------|---------------------------------------------------------------------------|
| `envx PROGRAM` fails                   | There are no external programs in a browser                               |
| Standard input is always empty         | The terminal has no stdin stream to give a tool                           |
| `--one-file-system` does nothing        | `DeviceID` is unsupported off Unix (`core/fs_other.go`)                   |
| Ownership is one user                  | `MemFS` models a single uid/gid; seeded system directories belong to root |
| Output stops at 1 MiB per line         | The output budget (NFR-11)                                                |
| A line that waits without writing (`sleepx infinity`) never returns | Only a reset ends it: Esc or Ctrl+C          |

## Related

- User-facing flow: WF-6 in [srs-workflows](../requirements/srs-workflows.md).
- The shell the playground drives: [shell-execution](shell-execution.md).
- Requirements: section 10 of
  [functional-requirements](../requirements/functional-requirements.md), and NFR-3, NFR-11,
  NFR-19 and NFR-33 in [non-functional-requirements](../requirements/non-functional-requirements.md).
- Frontend rules: [rules](../rules/README.md).
