# An engine reads and writes only its streams

An engine reads input and writes output and diagnostics only through the `core.Streams` it was given in `New`. It never touches `os.Stdin`, `os.Stdout` or `os.Stderr`; the one place the process streams appear is `core.Stdio()`, which the entry point passes in.

## Why

The streams are what make an engine usable anywhere other than a terminal. The shell connects pipeline stages with `io.Pipe` and redirections with files; the WebAssembly build routes output to the page; tests capture stdout and stderr in buffers and compare them exactly; a library caller may want to silence a tool entirely. A single direct write to `os.Stdout` defeats all of them, and in a pipeline it sends output to the wrong place.

## What it looks like

From [`core/core.go`](../../core/core.go):

```go
// Streams bundles the three standard streams a tool reads and writes. Passing
// them explicitly keeps every engine testable against in-memory buffers, and
// lets an application embedding an engine capture or silence its output.
type Streams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

// Stdio returns the process's standard input, output and error.
func Stdio() Streams {
	return Streams{In: os.Stdin, Out: os.Stdout, Err: os.Stderr}
}
```

The entry point, [`main.go`](../../main.go), is the only caller that hands the process's streams in: `os.Exit(cmd.Main(os.Args, core.Stdio()))`. Diagnostics and prompts are built from the same streams: `core.NewDiag(Name, s)` and `core.NewPrompter(Name, s)`.

Do: `fmt.Fprintln(s.Out, line)`, or `r.diag.Printf(...)`.

Do not: `fmt.Println(line)` or `bufio.NewReader(os.Stdin)` in an engine.

## How it is checked

- The engine suites build engines on `bytes.Buffer` streams and assert on exact stdout and stderr, so output that bypasses the streams is missing from the buffer and the test fails.
- `core.Streams` is passed by value with no default, so an engine cannot be built without one.
- That no file under `core/<gnu>/` names `os.Stdin`, `os.Stdout` or `os.Stderr` is review; today only `core/core.go` does.

## Related

- FR-32, FR-34 in [functional requirements](../requirements/functional-requirements.md); NFR-30 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [Engines know nothing of the command line](engines-know-nothing-of-the-command-line.md)
- [Tests drive engines, not terminals](tests-drive-engines-not-terminals.md)
