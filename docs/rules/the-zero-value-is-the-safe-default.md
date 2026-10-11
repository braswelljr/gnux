# The zero value is the safe default

An engine built from the zero value of its `Options` behaves exactly like the GNU tool run with no flags, and every safeguard is on. When a default is "on", the field is named for its negative, so that forgetting it can never switch a safeguard off.

## Why

The engines are a public Go library. A caller who writes `rm.Options{Recursive: true}` has thought about recursion and nothing else: not the root failsafe, not prompting. Go initializes every field they did not mention to its zero value, so the zero value has to be the behavior they would get from plain `rm -r`, including the refusal to recurse on `/`. A field named `PreserveRoot bool` would default to `false` and silently drop the failsafe for every library caller.

## What it looks like

From [`core/rm/model.go`](../../core/rm/model.go):

```go
// Options mirrors the GNU rm flag surface. The zero value is plain `rm`: no
// recursion, default prompting, and the '/' failsafe on.
type Options struct {
	...
	// NoPreserveRoot (--no-preserve-root) allows recursing on '/'. It is the
	// negative so the zero value keeps the failsafe on.
	NoPreserveRoot bool
}
```

The prompting mode follows the same idea: `PromptDefault` is `iota`, the zero `Mode`, and means "prompt for write-protected files on a terminal", which is what `rm` does with no flags.

The same naming appears wherever GNU's default is on: `NoClobber` (`core/cp`, `core/mv`), `NoCreate` and `NoDereference` (`core/touch`), `NoTargetDirectory` (`core/ln`, `core/mv`), `NoNewline` (`core/echo`), `NoOwner` and `NoGroup` (`core/ls`).

The engine checks the negative field, from [`core/rm/rm.go`](../../core/rm/rm.go):

```go
if r.Recursive && !r.NoPreserveRoot && core.IsRoot(path) {
	r.diag.Errorf("it is dangerous to operate recursively on '%s'", path)
	r.diag.Warnf("use --no-preserve-root to override this failsafe")
	return
}
```

Do: `NoPreserveRoot bool`, so `Options{}` keeps the failsafe.

Do not: `PreserveRoot bool` with the adapter setting it to `true`, which protects command-line users and nobody else.

## How it is checked

- The engine suites construct engines from literal `Options` values that name only the flags under test, so they exercise the zero value of everything else. [`core/rm/example_test.go`](../../core/rm/example_test.go) is the documented library use.
- Each adapter's `TestBuildOptions` has a `"defaults"` case asserting that no flags produce the zero `Options` (for example `{"defaults", nil, mv.Options{}}` in [`cmd/mvx/mvx_test.go`](../../cmd/mvx/mvx_test.go)).
- That a new field is named for its negative is review. No test exercises the root refusal directly, since it cannot be run safely.

## Related

- FR-31 in [functional requirements](../requirements/functional-requirements.md); NFR-36 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [Engines know nothing of the command line](engines-know-nothing-of-the-command-line.md)
- [Conflicts resolve toward the most cautious option](conflicts-resolve-toward-the-most-cautious-option.md)
