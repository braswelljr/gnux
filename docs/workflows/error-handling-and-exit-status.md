# Error Handling and Exit Status

How a failure travels from the point it happens to the process exit status: engines report each failed operation through `core.Diag` in GNU's wording, using `core.ErrText` and `core.Describe` to keep the text identical on every platform, and return one of four values (`nil`, `core.ErrFailed`, a `*core.ExitError` or a `*core.UsageError` built with `core.Usagef`); `cmd.Main` maps that value to a status and, for a usage error only, prints the message with the `Try '<tool> --help'` hint. It also covers how a closed output pipe (`EPIPE`) is treated as a quiet stop rather than a failure, and lists every special status a tool or the shell uses, taken from the code.

## The contract

Every engine's `Run`, and every adapter's `run`, returns exactly one of (FR-33):

| Returned            | Built with                         | Meaning                                                 | Printed by `cmd.Main`                       | Status            |
|---------------------|------------------------------------|---------------------------------------------------------|---------------------------------------------|-------------------|
| `nil`               |                                    | success                                                 | nothing                                     | 0                 |
| `core.ErrFailed`    | `diag.Result()`                    | it ran, something failed, diagnostics already written   | nothing                                     | 1                 |
| `*core.ExitError`   | `&core.ExitError{Code: n}`         | a specific status that carries meaning                  | nothing                                     | `Code`            |
| `*core.UsageError`  | `core.Usagef(format, ...)`         | a command-line mistake                                  | `<tool>: <msg>` and the hint                | `Spec.UsageExit` (default 1) |
| any other error     | cobra or pflag                     | a flag-parsing failure (unknown flag, bad value)        | `<tool>: <msg>` and the hint                | `Spec.UsageExit`  |

The split exists so that each failure is reported exactly once (NFR-45) and is visible twice:
as text on standard error and as a status a script can test (NFR-44).

- **Diagnostics are written where the failure happens**, because only the engine knows which
  operand failed and why, and because a failing operand must not stop the rest (FR-8). By the
  time `Run` returns there may be ten messages; the return value only summarizes them.
- **`ErrFailed` carries no text** so that nothing above the engine can print it a second time.
- **`ExitError` is silent** for the same reason, and carries the number for tools whose status
  means more than pass/fail.
- **A usage error is the one case printed by `cmd.Main`**, because only it knows the name the
  tool was invoked as and the hint GNU appends, and because cobra's own parse errors arrive the
  same way.

## The pieces in `core/core.go`

### `ErrFailed`

```go
var ErrFailed = errors.New("command failed")
```

Returned through `(*Diag).Result()`, which is `ErrFailed` when `Errorf` was called at least once.
Callers test it with `errors.Is(err, core.ErrFailed)`.

### `ExitError`

```go
type ExitError struct{ Code int }
func (e *ExitError) Error() string { return fmt.Sprintf("exit status %d", e.Code) }
```

The `Error` text exists to satisfy the interface; it is never printed by gnux.

### `UsageError` and `Usagef`

```go
type UsageError struct{ Msg string }
func Usagef(format string, a ...any) error { return &UsageError{Msg: fmt.Sprintf(format, a...)} }
```

An engine returns `core.Usagef("missing operand")` and never prints the hint itself: an engine
called from a library has no `--help` to point at, and `cmd.Main` adds it for the command line.

### `Diag`

`core.NewDiag(Name, s)` writes `<Name>: <message>` and counts errors. Its methods take a mutex,
so concurrent workers can share one (`core/rm`'s fan-out does).

| Method   | Writes to | Counts | Use                                                     |
|----------|-----------|--------|---------------------------------------------------------|
| `Errorf` | stderr    | yes    | a failed operation                                      |
| `Warnf`  | stderr    | no     | a note that is not itself a failure, or a second line of one |
| `Printf` | stdout    | no     | regular output such as `-v` reports                     |
| `Failed`, `Result` |  |        | whether anything was counted; `ErrFailed` or `nil`      |

### `ErrText`

`ErrText(err)` renders a Go error the way GNU prints a system error: the bare, capitalized
message, path and operation stripped, and the common cases normalized so the text is the same on
Linux, macOS, Windows and in the browser (FR-5, NFR-6):

```go
switch {
case errors.Is(err, syscall.ENOTEMPTY):
	return "Directory not empty" // checked before fs.ErrExist, which Go also matches ENOTEMPTY against
case errors.Is(err, fs.ErrNotExist):
	return "No such file or directory"
case errors.Is(err, fs.ErrPermission):
	return "Permission denied"
case errors.Is(err, fs.ErrExist):
	return "File exists"
}
// otherwise unwrap *fs.PathError / *os.LinkError and capitalize the first rune
```

The order matters: Go's `syscall.Errno.Is` reports `ENOTEMPTY` as matching `fs.ErrExist`, so
without the first case a refused directory removal would read `File exists`. `core/mv` adds one
more correction of its own (`errText`) for the Windows `ERROR_DIR_NOT_EMPTY` code.
`TestErrText` in `core/core_test.go` pins every case.

### `Describe`

`Describe(info)` returns GNU's noun for a file: `symbolic link`, `directory`, `fifo`, `socket`,
`character special file`, `block special file`, `regular empty file`, `regular file`, or `file`.
It is what makes `rmx -i` ask `remove regular empty file 'f'?` exactly as GNU does.

## How `cmd.Main` maps the result

`exitStatus` in `cmd/main.go`:

```go
if err == nil {
	return 0
}
var exit *core.ExitError
switch {
case errors.Is(err, core.ErrFailed):
	return 1
case errors.As(err, &exit):
	return exit.Code
}
name := common.AppName
if ran != nil {
	name = tool.DiagName(ran)
}
fmt.Fprintf(s.Err, "%s: %v\n", name, err)
fmt.Fprintf(s.Err, "Try '%s --help' for more information.\n", name)
return tool.UsageExit(ran)
```

- `ran` is the cobra command that actually executed, so `gnux rmx --bogus` is reported as `rmx`.
- `tool.DiagName(ran)` and `tool.UsageExit(ran)` read the annotations `Spec.Command` recorded
  (`gnux/diag-name`, `gnux/usage-exit`). For the root command, which has none, they fall back to
  `ran.Name()` (`gnux`) and 1.
- Both `errors.Is` and `errors.As` see through wrapping, so an adapter may wrap an engine error
  with `fmt.Errorf("...: %w", err)` without changing the status.

How the full path reaches here is in [command-line-to-tool-run](command-line-to-tool-run.md).

### Real output

```text
$ rmx
rmx: missing operand
Try 'rmx --help' for more information.                    # 1: UsageError from the engine
$ rmx --interactive=sometimes x
rmx: invalid argument "sometimes" for '--interactive'
Valid arguments are:
  - 'never'
  - 'once'
  - 'always'
Try 'rmx --help' for more information.                    # 1: UsageError from the adapter
$ gnux rmx --no-such-flag
rmx: unrecognized option '--no-such-flag'
Try 'rmx --help' for more information.                    # 1: pflag error (FR-73)
$ rmx missing
rmx: cannot remove 'missing': No such file or directory   # 1: ErrFailed, no hint
$ lsx missing
lsx: cannot access 'missing': No such file or directory   # 2: ExitError (serious trouble)
$ gnux envx nosuchprog
envx: 'nosuchprog': No such file or directory             # 127: ExitError
$ gnux envx --bogus
envx: unrecognized option '--bogus'
Try 'envx --help' for more information.                   # 125: envx UsageExit
```

## Every special status

Collected by searching the code for `ExitError{` and `UsageExit:`. Every tool not listed exits 0
on success, 1 on any failure, and 1 for a usage error (FR-7, NFR-46).

| Command      | Status | When                                                                 | Where                                              |
|--------------|--------|----------------------------------------------------------------------|----------------------------------------------------|
| `lsx`        | 1      | minor trouble: an entry found while listing cannot be examined (`ErrFailed`) | `core/ls/ls.go`, severity `minor`          |
| `lsx`        | 2      | serious trouble: a command-line operand cannot be accessed, a directory cannot be opened or read at the top level, a directory loop | `core/ls/ls.go`, severity `serious`, `&core.ExitError{Code: l.status}` |
| `lsx`        | 2      | a usage error, including an unknown flag                             | `cmd/lsx/lsx.go`, `UsageExit: 2`                   |
| `envx`       | 125    | envx itself failed: cannot unset a variable, cannot change directory (`-C`), a write error, a malformed `-S` string, a failure relaying streams | `core/env/env.go` (`ExitCanceled`), `cmd/envx/envx.go` |
| `envx`       | 125    | a usage error, including an unknown flag                             | `cmd/envx/envx.go`, `UsageExit: env.ExitCanceled`  |
| `envx`       | 126    | the command was found but could not be run                           | `core/env/env.go`, `cannotRun` (`ExitCannotInvoke`) |
| `envx`       | 127    | the command was not found                                            | `core/env/env.go`, `cannotRun` (`ExitNotFound`)    |
| `envx`       | the command's | otherwise; on Unix a command killed by a signal is `128 + signal` | `core/env/lookup_unix.go`, `exitStatus`           |
| `falsex`     | 1      | always, as a status, with nothing printed                            | `cmd/falsex/falsex.go`, `&core.ExitError{Code: 1}` |
| `truex`      | 0      | always                                                               | `cmd/truex/truex.go`                               |
| `gnux shell` | last   | the status of the last command run                                   | `core/shell/shell.go`, `Run` returns `&core.ExitError{Code: sh.status}` |
| `gnux shell` | 2      | a syntax error (and a non-interactive script stops)                  | `core/shell/shell.go`, `runLine`                   |
| `gnux shell` | 2      | `exit` with a non-numeric argument                                   | `core/shell/shell.go`, `exit`                      |
| `gnux shell` | 2      | a usage error (`gnux shell a b`)                                     | `cmd/shell.go`, `UsageExit: 2`                     |
| `gnux shell` | 127    | an unknown command inside the script                                 | `core/shell/shell.go`, `runCommand`                |
| `gnux shell` | 127    | the script file cannot be opened                                     | `cmd/shell.go`, `&core.ExitError{Code: 127}`       |
| `gnux shell` | 1      | a redirection could not be opened (that command is not run)          | `core/shell/shell.go`, `runCommand`                |
| `gnux`       | 1      | an unknown tool (`gnux nosuchtool`)                                  | `cmd/root.go`, `core.Usagef("unknown tool '%s'", ...)` |

Note that `falsex --help` and `falsex --version` exit 0: they are handled by `Spec.Command`
before `falsex`'s `run` is reached.

Inside the shell, a tool's status is whatever `cmd.RunTool` returned for it, so `lsx`'s 2 and
`envx`'s 125 to 127 survive into `$?` and into the script's own exit status.

## Closed pipes

When the reader of a tool's output goes away (`yesx | headx -n 2`, or a shell pipeline whose
last stage has finished), the next write fails. GNU tools die quietly of `SIGPIPE`; gnux has no
signals in-process, so the write error is recognized and treated as a reason to stop, not a
failure (NFR-14):

```go
// core/pipe_other.go (//go:build !windows)
func IsClosedPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe)
}
```

`core/pipe_windows.go` also accepts `ERROR_BROKEN_PIPE` (109) and `ERROR_NO_DATA` (232), the two
codes Windows uses for a vanished pipe reader, neither of which Go maps to `EPIPE`.

Three places produce such an error without an OS pipe:

| Source                                  | What it returns                       | Why                                                    |
|-----------------------------------------|---------------------------------------|--------------------------------------------------------|
| A real OS pipe                          | `EPIPE` (Unix), 109 or 232 (Windows)  | the reader process exited                              |
| `pipeWriter` in `core/shell/shell.go`   | `io.ErrClosedPipe` translated to `syscall.EPIPE` | a later pipeline stage closed its read end  |
| `writer` in `tools/wasm/main.go`        | `syscall.EPIPE`                       | the playground's 1 MiB output budget is spent (NFR-11) |

Making all three look like `EPIPE` means each tool needs one check, not three.

How each streaming tool handles it, from the code:

| Tool              | Check                                                   | On a closed pipe        | On any other write error            |
|-------------------|---------------------------------------------------------|-------------------------|-------------------------------------|
| `catx`, `headx`, `tailx`, `wcx` | a local `closedPipe`: `EPIPE` or `io.ErrClosedPipe` | stop, no message   | `write error: <ErrText>`, status 1  |
| `echox`           | `core.IsClosedPipe`                                     | no message              | `write error: <ErrText>`, status 1  |
| `yesx`            | `io.ErrClosedPipe` or `isBrokenPipe` (build-tagged; the Windows file also accepts 109 and 232) | stop, status 0 | `standard output: <ErrText>`, status 1 |
| `envx` (printing the environment) | none                                    | `write error: ...`, status 125 | the same                  |
| `lsx`             | none: the buffered writer's flush error is discarded    | output stops silently   | output stops silently               |

Two consequences worth knowing. The local `closedPipe` helpers in `catx`, `headx`, `tailx` and
`wcx` do not include the two Windows codes that `core.IsClosedPipe` and `yesx` handle, so on a
real Windows pipe those four could report a write error where `yesx` stops quietly; inside the
gnux shell and the playground they always see `EPIPE` and stop. And `lsx` never reports a write
error at all. The `TestWriteErrors` cases in `core/cat`, `core/head`, `core/tail` and `core/wc`
pin both halves of the rule ("closed pipe is quiet", "other errors are reported"), and
`TestRunStopsQuietlyOnARealClosedPipe` in `core/yes` uses a real OS pipe.

The shell's real output for the classic case:

```text
$ gnux shell -c 'yesx | headx -n 2; echox $?'
y
y
0
```

## Writing a new failure path

1. Report it where it happens: `d.Errorf("cannot remove '%s': %s", path, core.ErrText(err))`.
   Use GNU's exact wording; quote operands as GNU does.
2. Continue with the next operand.
3. At the end, `return d.Result()`.
4. If the GNU tool uses a status other than 1 for this case, return `&core.ExitError{Code: n}`
   instead, after writing the diagnostic, and add the row to the table above and to FR-7 in
   [functional-requirements](../requirements/functional-requirements.md).
5. For a malformed command line, return `core.Usagef(...)` before doing anything. If GNU uses a
   status other than 1 for usage errors, set `UsageExit` on the `Spec`.
6. For output, treat a closed pipe as a stop, not an error.

## Tests that pin this workflow

| Test                                                   | What it proves                                           | Requirement |
|--------------------------------------------------------|----------------------------------------------------------|-------------|
| `TestDiagnostics` (`cmd/main_test.go`)                 | engine failure exits 1 with no hint; usage errors name the tool, add the hint, exit 1; unknown tool | FR-6, FR-73, FR-75, NFR-45 |
| `TestErrText` (`core/core_test.go`)                    | platform-normalized system text, `ENOTEMPTY` first        | FR-5, NFR-6 |
| engine suites (`core/<gnu>/*_test.go`)                 | exact stderr and the returned error                      | FR-5, NFR-12, NFR-44 |
| `core/ls`, `core/env`, `cmd/envx` suites               | the 2 and 125 to 127 statuses                             | FR-7, NFR-46 |
| `TestSyntaxErrors`, `TestDiagnostics` (`core/shell`)   | 2 and 127 in the shell                                   | FR-27       |
| `TestShellCommand` (`cmd/shell_test.go`)               | a failing tool's status propagates; `gnux shell` usage errors | FR-23, FR-27 |
| `TestWriteErrors` (`core/cat`, `core/head`, `core/tail`, `core/wc`) | closed pipes are quiet, other write errors reported | NFR-14 |

## Related

- [command-line-to-tool-run](command-line-to-tool-run.md), where the mapping is applied.
- [engine-run](engine-run.md), where diagnostics are written.
- [shell-execution](shell-execution.md#closing-pipes-and-epipe), where pipeline `EPIPE` comes from.
- FR-5 to FR-8 and FR-33 in [functional-requirements](../requirements/functional-requirements.md);
  NFR-6, NFR-14, NFR-44 to NFR-46 in
  [non-functional-requirements](../requirements/non-functional-requirements.md).
- The [rules](../rules/README.md).
