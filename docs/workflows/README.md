# Workflows

How gnux's technical processes actually run inside the code, one document per process: from a command line to a tool run, inside an engine, through the portable shell, in the browser playground, when adding a tool, when generating the documentation, in CI and at release, and how every failure becomes an exit status. Each document follows the real source step by step, names the files and functions involved, explains why each step is built the way it is, shows real output where it can, and cites the requirement identifiers each step enforces. The user-facing counterparts of these processes are in [srs-workflows](../requirements/srs-workflows.md).

## The workflows

| Workflow                                                              | In one line                                                                                       |
|-----------------------------------------------------------------------|---------------------------------------------------------------------------------------------------|
| [command-line-to-tool-run](command-line-to-tool-run.md)               | `argv` to a tool run: `argv[0]` normalization, the registry, the root command, `Spec.Command`, `--help`/`--version`, and the error-to-status mapping |
| [engine-run](engine-run.md)                                           | Inside an engine, with `core/rm` as the example: `Options`, `New`, `Run`, diagnostics, prompting, concurrency, post-order recursion; how `core/mv` and `core/cp` differ |
| [shell-execution](shell-execution.md)                                 | `gnux shell` from source text to exit status: lexing, parsing, expansion, redirections, pipelines as goroutines, `EPIPE`, `-e`, `Run` versus `Eval` |
| [wasm-playground](wasm-playground.md)                                 | The real gnux in the browser: the `js/wasm` API, the in-memory filesystem, booting, the output budget, `/` commands and `/refresh`, and the Node tests |
| [adding-a-tool](adding-a-tool.md)                                     | A compiling engine and adapter skeleton, registration, the test that catches each mistake, and a checklist |
| [documentation-generation](documentation-generation.md)               | The registry to four reference formats and the website: `gendocs`, `cmd/docs`, determinism, `wrapCode`, `mdxEscape`, link rewriting, the Docs check |
| [ci-and-release](ci-and-release.md)                                   | Every GitHub workflow: trigger, steps, requirements enforced, local reproduction, common failures; the tag-to-release flow; where the version comes from |
| [publishing-a-release](publishing-a-release.md)                       | A version to every place it is installed from: the one-time setup of Homebrew, Scoop, winget, the Snap Store and the AUR, cutting and checking a release, installing one particular version, and recovering from a bad one |
| [error-handling-and-exit-status](error-handling-and-exit-status.md)   | `ErrFailed`, `ExitError`, `UsageError`, `ErrText`, `Describe`, closed pipes, and a table of every special status |
| [gnu-conformance](gnu-conformance.md)                                 | The harness that runs each case through the GNU tools and through gnux and compares output, errors and status |
| [terminal-visuals](terminal-visuals.md)                               | gnux's terminal view: the shared themes, bars, trees, icons and git status, how `--pretty` and the `GNUX_*` variables are resolved, how a tool adopts it, and the plan for every tool |
| [benchmarking-and-robustness](benchmarking-and-robustness.md)         | `make bench` and comparing two runs; the memory, goroutine and open-file tests; `core.FlushIfIdle`; the fuzz targets; where a panic is contained |

## Where to start

| If you want to                                  | Read                                                                                     |
|-------------------------------------------------|------------------------------------------------------------------------------------------|
| Understand the path every command takes         | [command-line-to-tool-run](command-line-to-tool-run.md), then [error-handling-and-exit-status](error-handling-and-exit-status.md) |
| Write or change an engine                       | [engine-run](engine-run.md), then [adding-a-tool](adding-a-tool.md)                       |
| Add a tool                                      | [adding-a-tool](adding-a-tool.md)                                                        |
| Change the shell language                       | [shell-execution](shell-execution.md)                                                    |
| Work on the website or the playground           | [wasm-playground](wasm-playground.md) and [documentation-generation](documentation-generation.md) |
| Fix a red CI run                                | [ci-and-release](ci-and-release.md)                                                      |
| Cut a release, or set up a package manager      | [publishing-a-release](publishing-a-release.md)                                          |
| Give a tool colors, bars, a tree or icons        | [terminal-visuals](terminal-visuals.md)                                                  |
| Make something faster, or check a change did not make it slower | [benchmarking-and-robustness](benchmarking-and-robustness.md)            |

## How to read them

Each document has the same shape:

1. **A summary paragraph** under the title. It is also the page's description on the website, so
   it says the whole thing in prose.
2. **A diagram** where the flow has enough moving parts to need one (a mermaid sequence diagram
   or flowchart).
3. **Numbered steps** in the order the code runs them. Each step names the file and function,
   quotes the few lines that carry the behavior, and says *why* it is built that way. Excerpts
   are short and current at the time of writing; the source is the authority when they differ.
4. **Real output** from a build of the current tree, marked with `$` prompts. Statuses that are
   not visible in the output are noted in a trailing comment.
5. **A table of the tests** that pin the workflow, with the requirement each one verifies.
6. **Related documents.**

Requirement identifiers (`FR-n`, `NFR-n`) refer to
[functional-requirements](../requirements/functional-requirements.md) and
[non-functional-requirements](../requirements/non-functional-requirements.md). A step that
"enforces FR-8" is the code that makes that requirement true, and the test named beside it is
what fails if it stops being true.

These documents describe *how*; they do not set rules. The rules the code follows are in
[rules](../rules/README.md), and task-oriented instructions are in [guides](../../documentation/guides/README.md).
When a change alters a process described here, update the document in the same change, as with
the requirement registers.

## Conventions

- Paths are relative to the repository root (`cmd/main.go`, `core/rm/rm.go`).
- Standard markdown only, with a single `# Title` and a prose summary first, because the
  documentation generator publishes this folder on the website and takes the title and
  description from them (see [documentation-generation](documentation-generation.md)).
- US spelling and no em dashes: the Lint workflow's `misspell` and the website's house-rule test
  check the same conventions elsewhere.
