# The shell runs no external program implicitly

`gnux shell` runs its builtins and the registered tools, in process, and nothing else. A name that is neither is `command not found` with status 127, never a lookup on `PATH`. An external program starts only when a tool whose purpose is running a command is asked to run it: `envx` (and `timeoutx`, `nicex`, `nohupx`), `xargsx`, and `findx -exec`.

## Why

The shell exists so that one script behaves the same on every platform. Falling through to the system's programs would make `ls` mean gnux on one machine and `/bin/ls` on another, and a typo could start a program the author never meant to run. In a browser there are no external programs at all, so a script that relied on them would work locally and fail in the playground.

## What it looks like

The interpreter reaches tools only through its `Exec` function, so `core/shell` does not import `cmd` and cannot reach anything the caller did not give it. From [`core/shell/shell.go`](../../core/shell/shell.go):

```go
// Exec runs a tool. It is the only way the shell reaches the tools, so a
// caller can restrict, wrap or replace the command set.
Exec ExecFunc
```

and the end of `runCommand`:

```go
if sh.Exec != nil {
	if status, found := sh.Exec(name, args, s); found {
		return status
	}
}
sh.diag.Errorf("%s: command not found", name)
return 127
```

`cmd.NewShell` wires in `cmd.RunTool`, which knows only the registry. `os/exec` is imported outside tests only where running a program is the tool's stated purpose: `core/env`, `core/timeout`, `core/nice`, `core/nohup`, and [`core/internal/run`](../../core/internal/run/run.go), which `xargsx` and `findx -exec` share. That package asks the registry first, so `xargsx rm` and `findx -exec rm {} +` run gnux's own `rm` in-process and behave the same on a machine with no GNU tools; only a name that is not a tool is looked up on `PATH`.

Do: `envx git status` when a script really needs an external program, or `findx . -name '*.go' | xargsx gofmt -l` when the program is what a tool is told to run.

Do not: add a `PATH` fallback to `runCommand`, or import `os/exec` in `core/shell`.

## How it is checked

- `TestDiagnostics` in [`core/shell/shell_test.go`](../../core/shell/shell_test.go) asserts `"gnux: nosuch: command not found\n"`.
- [`cmd/shell_test.go`](../../cmd/shell_test.go) runs the shell end to end with the real tools.
- That `os/exec` stays confined to `core/env` is review.

## Related

- BRULE-6 in [business requirements](../requirements/business-requirements.md); FR-25, FR-29 in [functional requirements](../requirements/functional-requirements.md); NFR-18 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [There is one list of tools](there-is-one-list-of-tools.md)
- [The playground never touches the visitor's files](the-playground-never-touches-the-visitors-files.md)
