# Non-Functional Requirements

Non-functional requirements define how well gnux must behave, as distinct from what it must do:
portability, performance, reliability and data safety, security and privacy, usability,
accessibility, maintainability, testability, the observability of failures, discoverability and
licensing. They bind every tool, the shell, the library and the website. Each has a stable
identifier (`NFR-n`), a measure where one is possible, and the check that verifies it.

**This document stands on its own.** What the software must do is in
[functional-requirements](functional-requirements.md); why is in
[business-requirements](business-requirements.md).

Requirement levels: **must** is mandatory, **should** is expected unless there is a stated
reason not to, **may** is optional.

## How to read the verification entries

The entry under each requirement names what checks it, and the wording is deliberate:

| Wording              | Means                                                                                     |
|----------------------|-------------------------------------------------------------------------------------------|
| A test or workflow   | Something fails automatically when the requirement is broken                              |
| Review               | A person has to look. Nothing fails on its own                                            |
| Not verified         | Nothing checks it today, and the entry says what would                                    |

`NFR-1` to `NFR-35` keep the numbers they had in the first draft of this register. `NFR-36`
onward were added since. The document is ordered by quality attribute, so the numbers do not
run in order down the page.

Three gaps are worth knowing before reading further. Start-up time (NFR-8) and accessibility
(NFR-24) have no automated check. Formatting of the Go code (NFR-28) is not checked in CI. And
the suite runs on Linux and Windows only in CI: a developer's own machine exercises one
platform, so a change that breaks another is found by the Test workflow, not before it.

## 1. Compatibility and portability

### NFR-1: supported platforms

gnux must run on Linux (amd64, arm64 and riscv64), macOS and FreeBSD (amd64 and arm64) and
Windows (amd64 and arm64).

- **Measure.** Nine targets: `linux/amd64`, `linux/arm64`, `linux/riscv64`, `darwin/amd64`,
  `darwin/arm64`, `freebsd/amd64`, `freebsd/arm64`, `windows/amd64`, `windows/arm64`.
- **Verified by.** The Build workflow compiles all nine, and the Release workflow
  (`.goreleaser.yaml`) archives all nine. The Test workflow runs the suite on
  `ubuntu-latest`, `macos-latest` and `windows-latest`; the architecture of each is whatever
  GitHub's runner is, so not every target's tests are executed.

### NFR-2: static, no cgo

gnux must build without cgo and link statically, so the binary has no runtime dependencies.

- **Measure.** `CGO_ENABLED=0` succeeds for every target.
- **Verified by.** The Build workflow sets it for every target; the Dockerfile sets it too.

### NFR-3: WebAssembly

Every tool must also compile to WebAssembly (`js/wasm`).

- **Rationale.** The playground runs the real tools. One tool that stops compiling for the
  browser removes all of them from it.
- **Verified by.** The "Build for WebAssembly" step of the Test workflow; the Web workflow,
  which builds `gnux.wasm` and runs tests against it.

### NFR-4: isolated platform code

Platform-specific code must be isolated in build-tagged files and must have a fallback that
compiles everywhere.

- **Measure.** No `runtime.GOOS` branch outside test files. Every build-tagged file has a
  counterpart whose constraint covers the rest.
- **Verified by.** The Build workflow and the WebAssembly build fail when a fallback is
  missing. The absence of `runtime.GOOS` branching is checked by review; at the time of writing
  it appears only in `core/ls/ls_test.go`.

### NFR-5: predictable degradation

Where a platform lacks a concept (owners, inodes or unprivileged symbolic links on Windows),
the tool must degrade predictably and say so in its documentation.

- **Verified by.** Review of each tool's Permissions note. Platform-specific tests in
  `core/ls`, `core/mkdir` and `core/env` cover the behavior on the platforms they run on.

### NFR-6: the same messages everywhere

Common error messages must read the same on every platform.

- **Measure.** Not found, permission denied, already exists and directory not empty each have
  one fixed text.
- **Verified by.** `TestErrText` in `core/core_test.go`.

## 2. Performance

### NFR-7: streaming

Tools must process input as a stream; memory use must not grow with file size unless the
operation requires it.

- **Measure.** `catx`, `headx`, `tailx` and `wcx` read through fixed-size buffers. `headx` stops
  reading once it has enough; `tailx` seeks to the end of a regular file and reads backward.
  Every engine buffers its streams through `core.NewReader` and `core.NewWriter`, which share
  one size, `core.BufferSize` (64 KiB).
- **Verified by.** `TestLargeInputs` (`core/head`, `core/tail`),
  `TestLargeInputCrossesBuffers` (`core/cat`) and `TestChunkBoundaries` (`core/wc`) prove
  correctness across buffer boundaries. `TestStreamingToolsUseBoundedMemory` in
  `cmd/robust_test.go` measures memory: every streaming tool reads 4 MiB while the live heap
  may grow by 1 MiB, and `TestMemoryTestSeesAToolThatKeepsItsInput` checks that the
  measurement does catch a tool that keeps its input.

### NFR-8: start-up time

Tools must start in a time imperceptible in interactive use.

- **Measure.** Naming a tool, by `argv[0]` or as the first argument, builds that tool's command
  only, never the root command that carries all of them. `Lookup` reads an index built once per
  process.
  `gnux shell` is run directly too.
- **Verified by.** `BenchmarkDispatch` in `cmd/main_test.go` measures `cmd.Main` for a trivial
  tool under each way of naming it, and `BenchmarkShell/true` the shell
  (`make bench BENCH='Dispatch|Shell/true'`). They are measurements, not checks: nothing fails
  when they get slower.

### NFR-51: a filter answers each line

A tool that turns each input line into output must pass a line on when it has it, without
waiting for more input.

- **Rationale.** Behind a slow producer (`tail -f`, a build log, a terminal) a filter that
  waits for its output buffer to fill looks hung.
- **Measure.** Filters call `core.FlushIfIdle` after each record: the output is flushed when
  no more input is buffered, which costs nothing on bulk input.
- **Verified by.** `TestLineFiltersAnswerEachLine` in `cmd/robust_test.go`.

### NFR-52: throughput is measured

The speed of the tools must be measurable with one command, and a change that makes a tool
markedly slower, or makes it allocate markedly more, must fail a check.

- **Measure.** `make bench` runs `cmd/bench_test.go`: the real tools on generated text, a
  directory tree and shell pipelines, reporting MB/s and allocations. `make bench/check`
  compares two runs made on the same machine with `tools/benchcheck`: a benchmark is slower
  when all of its new runs are slower than all of its old ones by more than the tolerance, and
  its allocations are compared by their medians.
- **Verified by.** The Bench workflow, which measures the base commit and the pull request in
  one job and fails on a regression beyond 40 percent in time or 10 percent in allocations;
  `tools/benchcheck/main_test.go`. A slowdown inside the tolerance is not caught. See
  [benchmarking-and-robustness](../workflows/benchmarking-and-robustness.md).

### NFR-9: concurrency where it is safe

Independent operations should run concurrently where order and prompting do not forbid it.

- **Measure.** `rmx` removes independent top-level arguments with one worker per CPU, bounded by
  the number of arguments, when it cannot prompt and is not verbose. It is the only tool that
  does so today.
- **Verified by.** `TestConcurrentBulkRemoval` in `core/rm/rm_test.go`, under the race
  detector.

### NFR-10: ordered output stays ordered

Output that GNU guarantees to be ordered must stay ordered, concurrency notwithstanding.

- **Measure.** `rmx -v` reports removals in argument order, and children before their
  directory.
- **Verified by.** `TestVerboseOutput` and `TestInjectedRemoveFn` in `core/rm/rm_test.go`.

### NFR-11: a responsive playground

The playground must stay responsive: a command that produces unbounded output must be stopped,
not freeze the page.

- **Measure.** One line may write at most 1,048,576 bytes (1 MiB) across standard output and
  standard error. Past that, writes fail like a closed pipe, the tool stops, and a notice is
  written to standard error.
- **Verified by.** "runaway output is cut off instead of hanging" in `web/test/wasm.test.ts`.
  A command that waits without writing (`sleepx infinity`) is not stopped by the budget: it is
  ended by resetting the session.

## 3. Reliability and data safety

### NFR-12: no false success

A tool must never report success for an operation that failed.

- **Measure.** Any reported error makes the exit status nonzero.
- **Verified by.** The engine suites assert on the returned error beside the diagnostics.

### NFR-13: atomic replacement

Replacing a file should be atomic where the platform allows it.

- **Measure.** `lnx -f` creates the new link under a temporary name and renames it into place.
  `mvx` within one filesystem is a single rename. `cpx` overwrites in place, as GNU `cp` does.
- **Verified by.** `TestForceReplaces` (`core/ln`), `TestRename` (`core/mv`).

### NFR-14: closed pipes

Writing to a closed pipe must end the tool quietly, as it does for the GNU tools.

- **Verified by.** `TestWriteErrors` (`core/cat`, `core/head`, `core/tail`, `core/wc`);
  `TestRunStopsQuietlyOnARealClosedPipe` and `TestRunReportsOtherWriteErrors` (`core/yes`).

### NFR-49: a panic stays in the tool

A bug that makes a tool panic must not end the process that ran it.

- **Rationale.** A tool may be one stage of a shell pipeline, a command in the playground or a
  call from an embedding program.
- **Measure.** `cmd.Main` and `cmd.RunTool` recover the panic, print `<tool>: internal error`
  with the stack, and return status 2. The engines that start goroutines (`core/rm`,
  `core/du`, `core/xargs`, `core/shell`) carry a panic on one of them back to the calling
  goroutine with a `core.Trap`, and keep their channels and pipes moving so that nothing waits
  for the goroutine that died.
- **Verified by.** `TestPanicIsContained` and `TestCarriedPanicShowsItsOwnStack` in
  `cmd/main_test.go`; `core/trap_test.go`; `TestWorkerPanicReachesTheCaller` (`core/rm`),
  `TestParallelPanicReachesTheCaller` (`core/xargs`) and
  `TestPipelineStagePanicReachesTheCaller` (`core/shell`).

### NFR-50: nothing is left behind

A command must leave no goroutine running and no file open when it returns.

- **Rationale.** A host that runs many commands in one process would otherwise collect them
  until it ran out.
- **Verified by.** `TestNoGoroutineIsLeftBehind` in `cmd/robust_test.go` (shell pipelines,
  readers that stop early, `xargsx -P`) and `TestNoFileIsLeftOpen` in
  `cmd/robust_unix_test.go` (the file-handling tools, failure paths included; Unix only).

### NFR-15: no data races

Concurrent code must be free of data races.

- **Verified by.** `go test -race ./...` in the Test workflow and in `make test`.

### NFR-16: no lasting process-wide effects

Library use must not alter process-wide state beyond the call, other than the effects the
caller asked for.

- **Measure.** `Shell.Run` restores the working directory it found. `Shell.Eval` keeps it by
  design, for hosts that want a continuous session.
- **Known limit.** `cd` is process-wide while a script runs, so two scripts cannot run
  concurrently in one process without affecting each other.
- **Verified by.** `TestCd` and `TestEvalKeepsState` in `core/shell/shell_test.go`.

### NFR-36: safeguards hold in the zero value

Every safeguard must be on when an engine is constructed from the zero value of its options.

- **Rationale.** A library caller who writes `rm.Options{Recursive: true}` has not thought about
  the root failsafe and must still have it.
- **Verified by.** The engine suites, which start from the zero value; review of each new
  `Options` field against
  [the zero value is the safe default](../rules/the-zero-value-is-the-safe-default.md).

## 4. Security and privacy

### NFR-17: nothing the user did not name

gnux must not access the network, collect telemetry, or read or write anything the user did not
name.

- **Measure.** No package under `core/`, `cmd/` or `internal/` imports a network package. No
  configuration file is read. `envx` runs exactly the program it is given.
- **Verified by.** Review. No automated check enforces the import rule.

### NFR-18: no implicit external programs

The shell must not execute external programs implicitly; they are reachable only through
`envx`.

- **Measure.** A command name that is not a builtin or a registered tool is `command not
  found`, status 127.
- **Verified by.** `TestDiagnostics` in `core/shell/shell_test.go`.

### NFR-19: the playground stays in the page

The playground must not touch the visitor's files; everything it does stays in the page's
memory.

- **Measure.** The WebAssembly program's filesystem is the in-memory one supplied by the page.
  The only network requests are for the site's own `gnux.wasm` and `wasm_exec.js`.
- **Verified by.** `web/test/memfs.test.ts` and `web/test/wasm.test.ts` exercise the filesystem;
  the absence of other storage or network use is checked by review.

### NFR-20: few, current dependencies

Dependencies must be kept to a minimum and kept current.

- **Measure.** Three direct Go dependencies: cobra, pflag and go-isatty.
- **Verified by.** Dependabot (`.github/dependabot.yml`) proposes updates for Go modules weekly
  and for GitHub Actions daily. It is not configured for the website's packages.

### NFR-21: unprivileged container

The container image must run as an unprivileged user.

- **Measure.** The image creates user `gnux` (uid 10001) and switches to it.
- **Verified by.** Review of the `Dockerfile`. No workflow builds the image.

### NFR-37: least-privilege automation

Every CI workflow must run with read-only repository permissions unless it needs more.

- **Measure.** Each of the six workflows declares `permissions: contents: read` at the top level.
  The one exception is the Release workflow's publishing job, which alone is granted
  `contents: write` to create the release.
- **Verified by.** Review of `.github/workflows/`.

### NFR-38: baseline response headers

The website should send baseline security headers on every route.

- **Measure.** `X-Content-Type-Options: nosniff`, `X-Frame-Options: SAMEORIGIN`, a
  `Referrer-Policy` and a `Permissions-Policy`, set in `web/next.config.ts`. There is no
  Content-Security-Policy, deliberately: the theme script is inline and the playground compiles
  WebAssembly, so a useful policy needs a per-request nonce.
- **Verified by.** Review. Nothing asserts on the headers.

## 5. Usability

### NFR-22: no manual required

A user of the GNU tools must be able to use gnux without reading its documentation.

- **Verified by.** Review, against the option tables in
  [functional-requirements](functional-requirements.md).

### NFR-23: messages that say what failed

Messages must say what failed and on which file.

- **Measure.** Every operand-level diagnostic names the operand in single quotes and gives the
  system's reason.
- **Verified by.** The engine suites compare diagnostics exactly.

## 6. Accessibility

### NFR-24: an accessible website

The website must be usable with a keyboard and a screen reader, in light and dark themes, on
phone-sized screens.

- **Measure.** Semantic markup, an accessible name on every control, no meaning carried by
  color alone, and a layout that works at phone width. The rules that carry this are
  [the markup says what the thing is](../rules/the-markup-says-what-the-thing-is.md) and
  [the shortest path, and only one of them](../rules/the-shortest-path-and-only-one-of-them.md).
- **Verified by.** Not verified automatically. The Web workflow lints, type-checks and builds;
  it runs no accessibility audit and no browser test.

## 7. Maintainability

### NFR-25: one structure for every tool

Every tool must follow the same two-package structure: an engine and a thin adapter.

- **Measure.** `core/<gnu>/` and `cmd/<gnu>x/` for each tool. `truex` and `falsex` are the one
  exception: they have no engine.
- **Verified by.** Review. `TestRegistryIsConsistent` checks what an adapter must expose.

### NFR-26: one list of tools

There must be exactly one list of tools, from which the program, the dispatcher and the
documentation are built.

- **Verified by.** `TestRegistryIsConsistent`, `TestRootListAndVersion`, `TestGenerateAll` and
  `TestGenerateMDX` all read `cmd.Tools()`.

### NFR-27: documented exports

Exported symbols must be documented; comments must explain why, not restate the code.

- **Verified by.** Review. The linter does not enforce it: revive's `exported` rule is not
  enabled in `.golangci.yml`.

### NFR-28: lint and format

The code must pass the configured linters and be formatted by the standard formatters.

- **Measure.** `golangci-lint` v2 with the repository's configuration reports nothing. On the
  website, `eslint` and `prettier --check` report nothing.
- **Verified by.** The Lint workflow for the Go linters and Go formatting (`gofmt -s` and
  `goimports` are configured as golangci-lint formatters, which `golangci-lint run` checks); the
  Web workflow for the website's lint and format check. `make format` applies both formatters,
  and the lefthook pre-commit hook applies and checks both before a commit.

### NFR-29: reproducible generation

Generated files must be reproducible: generating twice from the same source yields identical
output.

- **Measure.** No timestamps; the man page date is pinned; cobra's generation footer is
  disabled.
- **Verified by.** `TestGenerateAllDeterministic` for the four reference formats. The website
  content has no determinism test of its own; the Docs workflow would fail if it varied.

### NFR-39: engines do not depend on the command-line framework

Nothing under `core/` may import cobra or pflag.

- **Rationale.** The engines are the public library. A caller who imports `core/rm` should not
  pull a command-line framework into their program.
- **Verified by.** Review. No automated check enforces it.

### NFR-40: the generator does not ship in the binary

The documentation generator and its dependencies must not link into the `gnux` binary.

- **Measure.** Only `tools/gendocs` imports `cmd/docs`, which is what pulls in `cobra/doc` and
  `go-md2man`.
- **Verified by.** Review. No automated check enforces it.

### NFR-41: one place for the toolchain version

The Go version must be stated in one place.

- **Measure.** `go.mod`. Every workflow reads it with `go-version-file: go.mod`.
- **Verified by.** Review of `.github/workflows/`. The README, the guide and `AGENTS.md` repeat
  the version for readers and must be changed with it; nothing checks that they agree.

### NFR-42: house style

Prose, comments and messages must use US spelling and no em dashes.

- **Verified by.** `misspell` with `locale: US` in the Lint workflow, for spelling in Go
  sources. Nothing checks for em dashes, and nothing checks the markdown; both are review.

## 8. Testability

### NFR-30: testable without a terminal

Every engine must be testable against temporary directories and in-memory streams, with no
terminal.

- **Verified by.** The engine suites, which use `t.TempDir()`, buffers and injected functions.

### NFR-31: the race detector is always on

Tests must run under the race detector.

- **Verified by.** `go test -race` in the Test workflow and in `make test`.

### NFR-32: every mapping and precedence rule is tested

Every option-to-behavior mapping and every precedence rule must be covered by a test.

- **Verified by.** `TestBuildOptions` in each `cmd/<tool>x` package. Completeness is by review:
  nothing fails when a new option is registered without a row in the table.

### NFR-33: the playground's filesystem is tested against the real build

The playground's filesystem must be tested against the real WebAssembly build.

- **Verified by.** `web/test/wasm.test.ts`, run by `make web/test` and the Web workflow after
  `make web/wasm`.

### NFR-48: parsers survive arbitrary input

No script text may make the shell's lexer, parser or expansion panic, and no sed script, grep
pattern, find expression or xargs input may make its tool panic.

- **Rationale.** `gnux shell` reads scripts it did not write, including whatever a visitor
  types into the playground.
- **Measure.** `FuzzParse` and `FuzzExpand` (`core/shell/fuzz_test.go`) find no failure; an
  accepted line always yields non-empty pipelines and commands, and every word at least one
  argument. `FuzzExpand` globs through `Shell.Glob` against an in-memory file list: a pattern
  such as `/*/*/*/*` against a real Linux runner's disk walks `/proc` and stalls the fuzzer.
  `FuzzScript` (`core/sed`), `FuzzPattern` (`core/grep`), `FuzzExpression` (`core/find`) and
  `FuzzInput` (`core/xargs`) run the engines on generated input, each kept from touching the
  rest of the machine.
- **Verified by.** The seeds run in every `go test`; the Test workflow fuzzes each target for
  30 seconds on Linux (`make fuzz`).

### NFR-43: a new tool is covered on registration

Registering a tool must be enough to put it under the registry and documentation tests.

- **Measure.** The tests iterate over `cmd.Tools()`; none lists tools by name.
- **Verified by.** `cmd/main_test.go` and `cmd/docs/docs_test.go`.

## 9. Observability of failures

A command-line tool has two instruments: what it writes to standard error and the status it
exits with. These requirements keep both truthful.

### NFR-44: every failure is visible twice

Every failure must produce a diagnostic on standard error and a nonzero exit status.

- **Measure.** The two never disagree: a diagnostic counted as an error makes the status
  nonzero, and a nonzero status always follows a diagnostic, except for the tools whose only
  output is the status (`falsex`) and the relayed status of a command run by `envx`.
- **Verified by.** The engine suites; `TestDiagnostics` in `cmd/main_test.go`.

### NFR-45: a failure is reported once

A failure must be reported exactly once.

- **Measure.** An engine that has written its diagnostics returns `core.ErrFailed`, and the
  caller exits with status 1 without printing it. Only a usage error is printed by the caller.
- **Verified by.** `TestDiagnostics` ("engine failure exits 1 without a usage hint").

### NFR-46: statuses that mean something

A tool's exit statuses must be documented and stable.

- **Measure.** The table in section 5 of the
  [software-requirements-specification](software-requirements-specification.md).
- **Verified by.** `cmd/main_test.go`, `cmd/shell_test.go`, `core/env` and `core/ls` suites.

## 10. Search and discoverability

### NFR-34: content without scripts

Pages must be rendered so that their content is available to crawlers without running scripts.

- **Measure.** Documentation pages are generated at build time from static parameters.
- **Verified by.** The Web workflow builds them. That the served HTML carries the content is
  checked by review.

### NFR-35: machine-readable summaries

The site should provide machine-readable summaries of the documentation for language-model
tools.

- **Measure.** `/llms.txt` (an index) and `/llms-full.txt` (every page).
- **Verified by.** The Web workflow builds both routes. Their content is not asserted on.

## 11. Licensing and compliance

### NFR-47: a permissive license throughout

gnux must be released under the MIT license, and nothing linked into the binary may carry a
copyleft license.

- **Measure.** `LICENSE` is MIT. The modules linked into the binary are cobra, pflag,
  go-isatty and their own dependencies.
- **Verified by.** Review. There is no automated license scan.
