# Tests drive engines, not terminals

Each tool is tested by building its engine directly, against a `t.TempDir()` and in-memory streams, with prompts, clocks and the environment injected as functions, and asserting on exact stdout, stderr and the returned error. Tests use the standard library only, and the race detector is always on.

## Why

A test that needs a real terminal, the real clock or the developer's environment cannot run in CI and gives different answers on different machines. Engines already take their streams as parameters and expose their dependencies as exported fields, so a test can answer a prompt, freeze time or record every removal without a pseudo-terminal. The race detector is always on because the engines are concurrent (`rmx` fans out one worker per CPU, the shell runs pipeline stages as goroutines) and a race that only shows up under load is the kind nobody reproduces later.

## What it looks like

Injection points are exported fields on the engine. From [`core/rm/model.go`](../../core/rm/model.go):

```go
// PromptFn, when non-nil, is consulted instead of reading from standard
// input. It is the injection point used by tests to answer confirmation
// prompts.
PromptFn func(question string) bool

// RemoveFn, when non-nil, replaces os.Remove for the actual unlink. Tests
// use it to record removals without touching the filesystem; setting it
// also disables the os.RemoveAll fast path so every entry flows through it.
RemoveFn func(path string) error
```

The shell does the same with `Getenv` and `HomeDir` ([`core/shell/shell.go`](../../core/shell/shell.go)), and `core.Prompter` has `Fn`.

The layers are tested separately:

| Suite                                    | What it covers                                       |
|------------------------------------------|------------------------------------------------------|
| `core/<gnu>/<gnu>_test.go`               | the engine, black-box, against a temp dir            |
| `cmd/<gnu>x/<gnu>x_test.go`              | `buildOptions`: every flag mapping and precedence    |
| `core/shell/shell_test.go`               | the interpreter, against a fake command set          |
| `cmd/main_test.go`, `cmd/shell_test.go`  | dispatch, exit statuses, the shell with real tools   |
| `cmd/docs/docs_test.go`                  | the generator                                        |

Do: `t.TempDir()`, a `bytes.Buffer` for each stream, and `PromptFn` returning the answer.

Do not: write to the developer's home directory, call `time.Now()` in an assertion, or pipe answers into `os.Stdin`.

## How it is checked

- `go test -race ./...` in the [Test workflow](../../.github/workflows/test.yml), on ubuntu, macOS and Windows, and in `make test`.
- `TestConcurrentBulkRemoval` in [`core/rm/rm_test.go`](../../core/rm/rm_test.go) exercises the concurrent path under the race detector.
- That tests stay on the standard library is review: the only direct requirements in `go.mod` are cobra, pflag and go-isatty.

## Related

- FR-34 in [functional requirements](../requirements/functional-requirements.md); NFR-15, NFR-30, NFR-31, NFR-32 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [An engine reads and writes only its streams](an-engine-reads-and-writes-only-its-streams.md)
- [A memfs error carries a code Go knows](a-memfs-error-carries-a-code-go-knows.md), for the website's equivalent: tests against the real build.
