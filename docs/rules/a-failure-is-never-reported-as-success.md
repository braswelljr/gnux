# A failure is never reported as success

Every failure produces a diagnostic and a nonzero exit status, and the two never disagree. A tool that cannot do what was asked says so; it does not quietly do something else, and it does not exit 0. Each failure is reported exactly once.

## Why

The exit status is the only thing a script sees. `rmx x && deploy` must not deploy when `x` was not removed. The quieter forms of false success are the dangerous ones: a write error swallowed after the data was lost, or a platform that cannot touch a symbolic link's own times touching its target instead.

## What it looks like

The error convention in [`core/core.go`](../../core/core.go) separates "already reported" from "report this":

| Returned error     | Meaning                                            | Exit status |
|--------------------|----------------------------------------------------|-------------|
| `nil`              | success                                            | 0           |
| `core.ErrFailed`   | diagnostics already written                        | 1           |
| `*core.ExitError`  | a specific, silent status (`lsx` 2, `envx` 125-127) | `Code`      |
| `*core.UsageError` | printed by `cmd.Main` with the `--help` hint       | 1           |

`Diag.Result()` returns `ErrFailed` whenever any `Errorf` was counted, so a diagnostic always implies a nonzero status. `cmd.Main` does not print `ErrFailed`, so the failure is not reported twice.

Output errors count too. A closed pipe is the one quiet stop, as GNU's tools end on `SIGPIPE`; any other write error is reported. From [`core/cat/cat_test.go`](../../core/cat/cat_test.go):

```go
{"closed pipe is quiet", os.NewSyscallError("write", syscall.EPIPE), "", false},
{"other errors are reported", errors.New("disk on fire"), "catx: write error: Disk on fire\n", true},
```

A platform that cannot do the operation refuses rather than substituting another. From [`core/touch/lchtimes_other.go`](../../core/touch/lchtimes_other.go):

```go
// Off Linux the standard library offers no way to do that, so a symbolic link
// is refused - silently touching its target instead would change the wrong
// file.
```

Do: `d.Errorf(...)` and return `d.Result()`.

Do not: `_ = w.Write(...)` on the tool's output, or print a diagnostic and return `nil`.

## How it is checked

- `TestDiagnostics` in [`cmd/main_test.go`](../../cmd/main_test.go): "engine failure exits 1 without a usage hint".
- `TestWriteErrors` in `core/cat`, `core/head`, `core/tail` and `core/wc`.
- The engine suites assert on the returned error beside the diagnostics.
- `errcheck` in the Lint workflow, which ignores only `Close`, `os.Remove` cleanup and best-effort `fmt.Fprint*` diagnostic writes (see [`.golangci.yml`](../../.golangci.yml)).

## Related

- BRULE-4 in [business requirements](../requirements/business-requirements.md); FR-7, FR-33 in [functional requirements](../requirements/functional-requirements.md); NFR-12, NFR-14, NFR-44, NFR-45, NFR-46 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [A failing operand does not stop the rest](a-failing-operand-does-not-stop-the-rest.md)
- [Diagnostics are written in GNU's words](diagnostics-are-written-in-gnus-words.md)
