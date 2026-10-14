# Benchmarking and Robustness

How gnux is measured and how it is kept from leaking, hanging or crashing: the benchmark suite that runs the real tools and how to compare two runs of it, the tests that hold every streaming tool to a fixed amount of memory, check that no goroutine or open file outlives a command and that a filter answers a slow producer line by line, the fuzz targets for every parser of text gnux did not write, and the one place a tool's panic is contained. Everything here drives the tools through `cmd.RunTool` or `cmd.Main`, the path a real command line takes, so a result is about what a user runs and not about a function in isolation.

## Benchmarks

The suite is [`cmd/bench_test.go`](../../cmd/bench_test.go). Each benchmark is a command line, run in-process on generated input with the output discarded, so the figure is the tool's own work without the disk or the terminal.

| Benchmark           | What it runs                                                              | Reads as                    |
|---------------------|---------------------------------------------------------------------------|-----------------------------|
| `BenchmarkStream`   | Filters, counters, sorters and digests over 8 MiB of text on standard input | MB/s, B/op, allocs/op       |
| `BenchmarkGenerate` | Tools that produce output from nothing (`seq`, `echo`, `true`)             | ns/op                       |
| `BenchmarkTree`     | `find`, `grep -r`, `ls -R`, `du` over 2,000 files in 50 directories         | ns/op for the whole tree    |
| `BenchmarkShell`    | Whole pipelines in `gnux shell`: the tools plus the pipes between them      | ns/op for the script        |
| `BenchmarkDispatch` | What gnux spends before a tool starts, for each way of naming it (NFR-8)   | ns/op, allocs/op            |

```bash
make bench                              # everything, into bench/latest.txt
make bench BENCH='Stream/grep'          # one tool: BENCH is a -bench regexp
make bench BENCH='Tree|Shell' COUNT=6   # repeated, so a comparison has a spread
```

To judge a change, measure before and after on the same idle machine and compare:

```bash
git stash                 # or check out the base commit
make bench COUNT=6
make bench/save           # bench/latest.txt becomes bench/baseline.txt
git stash pop
make bench COUNT=6
make bench/compare        # benchstat: the change per benchmark, with its significance
```

`bench/` is ignored by Git: the numbers describe one machine. `make bench/compare` downloads `benchstat` the first time. A difference that `benchstat` marks `~` is noise, not a result; six runs are the least that give it something to work with.

### A pass or a fail

`make bench/check` turns the comparison into an exit status, with [`tools/benchcheck`](../../tools/benchcheck/main.go):

```bash
make bench COUNT=5 && make bench/save      # on the base commit
make bench COUNT=5 && make bench/check     # on the change: exit 1 on a regression
```

Timings depend on the machine and on what else it is doing, so no fixed number can be a threshold. The check compares two runs made on the same machine instead, and counts a benchmark as slower only when every one of its new runs is slower than every one of its old runs by more than `BENCH_TIME_TOLERANCE` percent (25 by default): the two ranges must not overlap. Allocations per operation hardly vary between runs, so their medians are compared against the tighter `BENCH_ALLOCS_TOLERANCE` (10 percent, and more than a handful). A benchmark that exists in only one of the two files is listed and never fails the check.

The Bench workflow (`.github/workflows/bench.yml`) does this on every pull request that touches Go code: it benchmarks the base commit and the pull request one after the other in the same job, five runs each, and fails when `benchcheck` does, with a 40 percent time tolerance because a shared runner is noisier than a desk. A slowdown smaller than that passes; the allocation check, which load does not disturb, catches most of what the timing check lets through.

To add a benchmark, add a command line to the matching list. A command that reads standard input belongs in `streamBenchmarks`; one that takes a directory uses `DIR` for the generated tree's root.

Two things the figures do not show. `BenchmarkStream/xargs -n 10 true` measures what it costs to start a tool 150,000 times, not `xargsx` itself. And process start-up (loading the binary, the Go runtime) is outside every in-process benchmark: measure it with a tool such as `hyperfine './bin/gnux true'`.

## What the suite found

The first runs of the suite and of the checks below changed the code, which is the reason they are kept:

- `gnux shell` built the root command, with every tool's flags, before reading a script: 318 µs. `cmd.Main` now runs it directly: 12 µs.
- Fifteen filters (`catx -n`, `nlx`, `cutx`, `trx`, `foldx`, `expandx`, `unexpandx`, `pastex`, `headx`, `uniqx`, `grepx`, `sedx`) held a line's output until their 64 KiB buffer filled. See "A filter answers each line" below.
- The memory test first measured garbage instead of live memory and blamed `xargsx` wrongly; it now collects before each sample.
- Running a tool through cobra's `Execute` scanned its arguments several times in search of a subcommand: 13 MB of garbage for a batch of 25,000 names from `xargsx`. `cmd.RunTool` now parses the flags and calls the tool itself (`runCommand` in `cmd/main.go`): `xargs true` over 8 MiB of names went from 119 ms and 847 MB allocated to 46 ms and 43 MB, and every dispatch from 52 allocations to 38.

## Memory: a streaming tool uses a fixed amount

`TestStreamingToolsUseBoundedMemory` ([`cmd/robust_test.go`](../../cmd/robust_test.go)) feeds each tool in the `streaming` list 4 MiB of generated text that is never held in memory whole, and samples the live heap (after a collection, so garbage does not count) while the tool reads. A tool may grow the heap by 1 MiB. The tools stay under half a mebibyte whatever the size of the input, and one that keeps its input grows the heap by about that size, so a small input separates the two and the test takes ten seconds under the race detector. `TestMemoryTestSeesAToolThatKeepsItsInput` is the control: `sortx` and `tacx` must hold their input, and the measurement has to show it, or a passing test would prove nothing.

A new filter, counter, digest or encoder goes in the `streaming` list. A tool that needs the whole input by nature does not; say so in its `Meta().Long`.

Go has no manual memory management, so "no memory leak" means three things here, each with its own test:

| What can leak               | Test                                  | How it would show                                   |
|-----------------------------|---------------------------------------|-----------------------------------------------------|
| Memory held while streaming | `TestStreamingToolsUseBoundedMemory`  | The heap grows with the input                       |
| Goroutines                  | `TestNoGoroutineIsLeftBehind`         | A pipeline stage still running after its script ended |
| Open files                  | `TestNoFileIsLeftOpen` (Unix)         | The process holds more descriptors after a command than before |

`TestNoGoroutineIsLeftBehind` runs shell pipelines, including ones whose reader stops early (`yes | head -n 2`) and parallel `xargsx -P`, and requires the goroutine count to return to where it was. `TestNoFileIsLeftOpen` ([`cmd/robust_unix_test.go`](../../cmd/robust_unix_test.go)) runs the tools that open many files, on their failure paths too (a missing operand among forty), and counts the entries of `/dev/fd` before and after. Both matter because a tool is one call in a long-lived host: a shell script, the playground, a program embedding the engines.

## A filter answers each line

A filter between a slow producer and its reader must pass each line on when it has it. Behind `tail -f`, a build log or a terminal, a filter that waits for its output buffer to fill looks hung.

`core.FlushIfIdle(out, in)` is the whole mechanism:

```go
func FlushIfIdle(out *bufio.Writer, in *bufio.Reader) {
	if in.Buffered() == 0 {
		_ = out.Flush()
	}
}
```

A filter calls it after each record it has written. When input is still buffered, the next read cannot block, so nothing is flushed. When the buffer has run dry, the next read will wait for the producer, so what has been written is passed on first. On bulk input a 64 KiB buffer runs dry once per 64 KiB, which is when the output would be written anyway: the cost is nil, and `BenchmarkStream` shows it. Engines that read raw blocks instead of lines (`catx`, `trx`, `headx`) flush once per read, which is the same rule.

`TestLineFiltersAnswerEachLine` writes one line to each filter in `lineFilters`, leaves the input open, and requires the output within three seconds. A new line-at-a-time filter goes in that list.

## Fuzzing: text gnux did not write

| Target                        | Package      | Input                                                       |
|-------------------------------|--------------|-------------------------------------------------------------|
| `FuzzParse`, `FuzzRun`        | `core/shell` | A script                                                    |
| `FuzzScript`                  | `core/sed`   | A sed script and its input, in both regular expression syntaxes |
| `FuzzPattern`                 | `core/grep`  | A pattern, an input and a combination of options            |
| `FuzzExpression`              | `core/find`  | A find expression, evaluated over a small tree              |
| `FuzzInput`                   | `core/xargs` | xargs input under every way of splitting and batching it    |

```bash
make fuzz                  # every target for 30 seconds
make fuzz FUZZTIME=5m      # longer, before a release
go test -run='^$' -fuzz='^FuzzScript$' -fuzztime=2m ./core/sed
```

The seeds run in every `go test`, and the Test workflow runs `make fuzz` on Linux. Each target keeps the fuzzer away from the rest of the machine: sed runs in sandbox mode, so no script reads or writes a file, and a script that branches is only parsed, because it may legitimately never end; a find expression that would write, delete, run a command or name a second starting point is skipped; xargs runs nothing, and checks instead that no command line goes over the `-s` limit it was given.

A failure is saved under the package's `testdata/fuzz/`. Commit that file with the fix: it becomes a seed.

## A panic stays in the tool

`contained` in [`cmd/main.go`](../../cmd/main.go) recovers a panic in the tool it runs, prints `<tool>: internal error: <value>` with the stack and a line asking for a report, and returns `ExitInternal` (2). A tool may be one stage of a shell pipeline, a command typed into the playground, or a call from an embedding program, and none of those should die of one tool's bug. `TestPanicIsContained` pins the behavior.

A `recover` only sees a panic on its own goroutine: one on a goroutine an engine started would end the process whatever the caller did. So every engine that starts goroutines brings their panics back with a [`core.Trap`](../../core/trap.go):

```go
var trap core.Trap
go func() {
	defer wg.Done()
	defer trap.Catch() // records the panic and its stack
	work()
}()
wg.Wait()
trap.Release() // raises it again here, as a *core.Panic
```

`Catch` is deferred after the defers that tell the engine the goroutine has finished, so it runs before them and the panic is recorded by the time the engine stops waiting. `Release` raises a `*core.Panic` carrying the stack of where the panic happened, which `contained` prints in place of its own.

| Engine      | Goroutines                         | What else keeps it from hanging                                   |
|-------------|------------------------------------|-------------------------------------------------------------------|
| `core/rm`   | One worker per CPU                 | The panic is caught per path, so the worker lives on to take the rest |
| `core/du`   | One per directory                  | The read slot is returned by a defer                              |
| `core/xargs`| One per running command under `-P` | The slot is released by a defer                                   |
| `core/shell`| One per pipeline stage             | The stage closes its pipes in a defer, so its neighbors end too   |

`TestWorkerPanicReachesTheCaller` (`core/rm`), `TestParallelPanicReachesTheCaller` (`core/xargs`) and `TestPipelineStagePanicReachesTheCaller` (`core/shell`) inject a panic and require it on the calling goroutine. An engine that starts a goroutine without a `Trap` reopens the hole: `core/timeout` starts two, each a single standard-library call.

## Requirements

NFR-7 (streaming), NFR-8 (start-up time), NFR-48 (parsers survive arbitrary input), NFR-49 (a panic stays in the tool), NFR-50 (nothing is left behind), NFR-51 (a filter answers each line) and NFR-52 (throughput is measured) in [non-functional requirements](../requirements/non-functional-requirements.md).
