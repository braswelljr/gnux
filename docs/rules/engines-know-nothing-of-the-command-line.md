# Engines know nothing of the command line

Every tool is two packages: an engine in `core/<gnu>/` that owns all behavior and takes an `Options` struct, and an adapter in `cmd/<gnu>x/` that declares the flags and folds them into that struct. Nothing under `core/` imports cobra or pflag, and nothing under `cmd/` walks a filesystem.

## Why

The engines are the public library. A Go program that imports `core/rm` should get removal, not a command-line framework, and should be able to set options as struct fields rather than by building an argument list. Keeping flags out of the engine also keeps the behavior in one place: the shell, the WebAssembly build and library callers all reach the same engine, whatever parsed the arguments.

## What it looks like

The engine contract, as every tool implements it (see [`core/rm/model.go`](../../core/rm/model.go) and [`core/rm/rm.go`](../../core/rm/rm.go)):

```go
const Name = "rmx"
type Options struct{ ... }
func New(opts Options, s core.Streams) *Rm
func (r *Rm) Run(args []string) error
```

The adapter does only mapping. From [`cmd/rmx/rmx.go`](../../cmd/rmx/rmx.go):

```go
func buildOptions(fs *pflag.FlagSet) (rm.Options, error) {
	...
	force.Apply(fs, &o)
	if err := interactive.Apply(fs, &o); err != nil {
	...
	preserveroot.Apply(fs, &o)
```

Do: add a field to `Options`, map the flag to it in `buildOptions`, and implement the behavior in the engine.

Do not: read `fs.GetBool` inside `core/`, or put traversal, prompting or I/O in `cmd/<gnu>x/`.

`truex` and `falsex` are the one exception: they have no behavior to put in an engine.

## How it is checked

- `TestRegistryIsConsistent` in [`cmd/main_test.go`](../../cmd/main_test.go) checks what an adapter must expose.
- Each adapter's `TestBuildOptions` tests the mapping in isolation from the engine.
- That `core/` stays free of `github.com/spf13/...` imports is review; no automated check enforces it. `grep -rl spf13 core` finds nothing today.

## Related

- FR-30 in [functional requirements](../requirements/functional-requirements.md); NFR-25, NFR-39 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [An engine reads and writes only its streams](an-engine-reads-and-writes-only-its-streams.md)
- [The zero value is the safe default](the-zero-value-is-the-safe-default.md)
- [Workflows](../workflows/README.md), in particular adding a tool.
