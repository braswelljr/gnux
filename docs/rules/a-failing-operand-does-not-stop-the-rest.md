# A failing operand does not stop the rest

When a tool is given several operands and one of them fails, it reports that one and carries on with the others. The failure is remembered and turns into exit status 1 at the end, after every operand has been tried.

## Why

That is what GNU does, and scripts depend on it: `rm a b c` with `b` missing still removes `a` and `c`. Stopping at the first failure would leave work half done in a way the user did not ask for and might not notice, and it would make the result depend on argument order.

## What it looks like

Per-operand functions report and return; they do not return an error that aborts the loop. From [`core/rm/rm.go`](../../core/rm/rm.go):

```go
if r.mayPrompt() || r.Verbose {
	for _, p := range paths {
		r.removeArg(p)
	}
} else {
	r.runConcurrent(paths)
}
return r.diag.Result()
```

`removeArg` writes its diagnostic with `r.diag.Errorf` and returns; `Diag` counts it, and `Result()` turns any count into `core.ErrFailed` once all operands are done.

Do: report with `diag.Errorf`, `return` from the per-operand function, and return `diag.Result()` from `Run`.

Do not: `return err` from inside the operand loop of `Run`.

A usage error (a missing operand, a bad flag value) is different: it is detected before any operand is processed and stops the tool.

## How it is checked

The engine suites have a test for it in most tools, for example:

- `TestErrorsDoNotStopLaterOperands` in [`core/cat/cat_test.go`](../../core/cat/cat_test.go)
- "one failure does not stop the rest" in [`core/ln/ln_test.go`](../../core/ln/ln_test.go)
- "One bad operand does not stop the rest" in [`core/touch/touch_test.go`](../../core/touch/touch_test.go)
- the remaining operands in [`core/realpath/realpath_test.go`](../../core/realpath/realpath_test.go) and [`core/ls/ls_test.go`](../../core/ls/ls_test.go)

## Related

- FR-8 in [functional requirements](../requirements/functional-requirements.md); NFR-44 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [A failure is never reported as success](a-failure-is-never-reported-as-success.md)
- [Diagnostics are written in GNU's words](diagnostics-are-written-in-gnus-words.md)
