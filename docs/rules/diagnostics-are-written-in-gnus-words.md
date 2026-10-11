# Diagnostics are written in GNU's words

Every diagnostic goes through `core.Diag`, is prefixed with the tool's name, names the operand in single quotes and uses GNU's exact wording. The system error text comes from `core.ErrText`, which gives the same capitalized message on every platform, and file-type nouns come from `core.Describe`.

## Why

Scripts and people match on these messages. `rmx: cannot remove 'x': No such file or directory` has to read the same as GNU's message, and the same on Linux, macOS and Windows, where Go's raw error strings differ (`The system cannot find the file specified.` on Windows). Writing every message through one type also lets the engine count errors for its exit status and keeps concurrent writes from interleaving.

## What it looks like

From [`core/core.go`](../../core/core.go):

```go
// ErrText renders an error the way GNU tools do: the bare, capitalized syscall
// message (e.g. "Permission denied", "Directory not empty") with the path and
// operation stripped. The common cases are normalized so the text is identical
// on every platform.
func ErrText(err error) string {
	switch {
	...
	case errors.Is(err, fs.ErrNotExist):
		return "No such file or directory"
	case errors.Is(err, fs.ErrPermission):
		return "Permission denied"
```

An engine uses it like this, from [`core/rm/rm.go`](../../core/rm/rm.go):

```go
r.diag.Errorf("cannot remove '%s': %s", path, core.ErrText(err))
```

`Diag.Errorf` writes `<prog>: <message>` to the error stream under a mutex and counts the error; `Diag.Warnf` writes without counting.

Do: `d.Errorf("cannot stat '%s': %s", path, core.ErrText(err))`.

Do not: `fmt.Fprintln(s.Err, err)`, which prints Go's wording, the operation and a platform-specific message.

Usage errors are the exception to writing diagnostics yourself: an engine returns `core.Usagef(...)` and `cmd.Main` prints it with the `Try '<tool> --help'` hint.

## How it is checked

- `TestErrText` in [`core/core_test.go`](../../core/core_test.go) pins the normalized messages.
- The engine suites compare stderr exactly, for example `"lnx: failed to create symbolic link '...': File exists\n"` in [`core/ln/ln_test.go`](../../core/ln/ln_test.go). They run on Linux, macOS and Windows in the Test workflow.
- `TestDiagnostics` in [`cmd/main_test.go`](../../cmd/main_test.go) checks the prefix and the usage hint.

## Related

- FR-5, FR-6 in [functional requirements](../requirements/functional-requirements.md); NFR-6, NFR-23 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [GNU is the specification, and every difference is documented](gnu-is-the-specification-and-differences-are-documented.md)
- [A failure is never reported as success](a-failure-is-never-reported-as-success.md)
