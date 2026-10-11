# Software Requirements Specification

The specification of gnux as a whole, arranged after IEEE 830: what the product is and where
its boundaries fall, the interfaces it presents (command line, Go API, WebAssembly API, streams
and exit statuses), its features cross-referenced to the functional requirements, its data and
error conventions, its quality attributes cross-referenced to the non-functional requirements,
and a traceability matrix from each business requirement to what verifies it.

**This document stands on its own** as the first point of contact for what gnux is and what it
must interface with. The registers it draws on are
[business-requirements](business-requirements.md),
[functional-requirements](functional-requirements.md) and
[non-functional-requirements](non-functional-requirements.md); the flows are in
[srs-workflows](srs-workflows.md).

## 1. Introduction

### 1.1 Purpose

This document specifies the requirements for gnux, a portable, GNU-compatible implementation
of the GNU core utilities' interface in Go. It is written for the maintainer, contributors, people embedding gnux in other programs, and
anyone deciding whether to rely on it. It is the reference for deciding whether a change
belongs in the project and whether a release is ready.

### 1.2 Scope

gnux is one multi-call binary carrying 22 tools, each named after its GNU counterpart with an
`x` appended. The same code is offered as commands, as a Go library and as a portable shell,
and is documented and demonstrated on a website with a WebAssembly playground. What is in and
out of scope is listed in [business-requirements](business-requirements.md).

### 1.3 Definitions

The glossary is section 8.

### 1.4 References

| Reference                                   | Used for                                                       |
|---------------------------------------------|----------------------------------------------------------------|
| GNU coreutils manual                        | The behavior, options, diagnostics and statuses each tool mirrors |
| POSIX Shell Command Language                | The subset `gnux shell` implements                             |
| IEEE Std 830-1998                           | The arrangement of this document                               |
| `AGENTS.md` at the repository root          | The architecture and the tool contract                         |
| [rules](../rules/README.md)                 | The cross-cutting rules the code follows                       |
| [workflows](../workflows/README.md)         | How each technical process runs                                |

### 1.5 Overview

Section 2 describes the product. Section 3 specifies its external interfaces. Section 4 lists
its features against the functional requirements. Section 5 states the data and error
conventions. Section 6 lists the quality attributes against the non-functional requirements.
Section 7 is the traceability matrix, and section 8 the glossary.

## 2. Overall description

### 2.1 Product perspective

gnux is self-contained. It brings the interface of tools that ship with some operating systems
to every platform, complements them rather than replacing them, and depends on none of them. Its parts, and how they sit on each other:

```text
                  main.go                      tools/wasm/main.go     tools/gendocs
                     |                                 |                    |
                     v                                 v                    v
   cmd.Main  ->  cmd.RunTool / cmd.NewRoot       cmd.NewShell          cmd/docs
                     |                                 |                    |
                     v                                 v                    |
          cmd/registry.go: Tools()  <------------------+--------------------+
                     |
                     v
          cmd/<gnu>x: Spec, Meta, Register, buildOptions      (adapters: flags and documentation)
                     |
                     v
          core/<gnu>: Options, New, Run                       (engines: all behavior)
                     |
                     v
          core: Streams, Diag, Prompter, errors, build-tagged filesystem helpers
```

- **Engines** (`core/<gnu>/`) own all behavior and are the public library.
- **Adapters** (`cmd/<gnu>x/`) declare flags, fold them into the engine's options, and carry
  the tool's documentation.
- **The registry** (`cmd/registry.go`) is the single list of tools. The root command, dispatch
  by program name, the generated reference and the website's pages are all built from it.
- **The shell** (`core/shell`, `cmd/shell.go`) interprets scripts whose commands are the tools.
- **The generator** (`cmd/docs`, `tools/gendocs`) writes the reference and the website content.
- **The website** (`web/`) is a separate Node project that presents the generated content and
  runs the WebAssembly build of gnux in a playground.

### 2.2 Product functions

| Function                                  | Summary                                                                     |
|-------------------------------------------|-----------------------------------------------------------------------------|
| File and directory management             | `cpx`, `mvx`, `rmx`, `rmdirx`, `mkdirx`, `lnx`, `touchx`                    |
| Listing and path inspection               | `lsx`, `pwdx`, `realpathx`, `basenamex`, `dirnamex`                         |
| Text output and selection                 | `catx`, `headx`, `tailx`, `wcx`, `echox`, `yesx`                            |
| Process and scripting helpers             | `envx`, `sleepx`, `truex`, `falsex`                                         |
| Multi-call dispatch                       | One binary; a tool is chosen by subcommand or by the name it was started under |
| Portable scripting                        | `gnux shell`: a POSIX sh subset whose commands are the tools, run in-process |
| Embedding                                 | Engines, tools by name, and scripts, callable from Go without a process     |
| Reference documentation                   | Generated from the registry in markdown, man, YAML, reST and website content |
| Evaluation in a browser                   | A playground running the real tools against an in-memory filesystem         |

### 2.3 User classes

| Class                       | Characteristics                                                    | Interface used                         |
|-----------------------------|--------------------------------------------------------------------|----------------------------------------|
| Command-line user           | Knows the GNU tools; expects them to behave identically            | The commands                           |
| Windows or container user   | Lacks the GNU tools; wants them without a compatibility layer      | The commands, under either name        |
| Script and CI author        | Wants one script for every developer machine and runner            | `gnux shell`                           |
| Go developer                | Wants GNU file semantics as a library                              | `core/<gnu>`, `cmd.RunTool`, `cmd.NewShell` |
| Contributor                 | Adds or changes a tool                                             | The tool contract, the rules, CI       |
| Evaluator or learner        | Has not installed anything                                         | The website and the playground         |
| Packager                    | Builds and redistributes the binary                                | The Makefile, the Dockerfile, the license |

### 2.4 Operating environment

| To                         | The environment needs                                                                          |
|----------------------------|------------------------------------------------------------------------------------------------|
| Run gnux                   | Linux or macOS on amd64 or arm64, or Windows on amd64. Nothing else                            |
| Use the per-tool commands  | Symbolic links and the install directory on `PATH`; on Windows, a copy of `gnux.exe` under the tool's name works instead |
| Install or build           | Go at the version in `go.mod` (1.25 or newer) and `git`. `make`, `bash` and Docker are optional |
| Use gnux as a library      | A Go module on Go 1.25 or newer                                                                |
| Contribute                 | The above, plus golangci-lint v2, and lefthook for the Git hooks                               |
| Work on the website        | The above, plus Node.js and pnpm at the versions stated in `web/package.json`                  |
| Use the playground         | A browser with WebAssembly support                                                             |

### 2.5 Design and implementation constraints

- **Language and dependencies.** Go, with the standard library plus cobra and pflag (command
  line) and go-isatty (terminal detection). Engines must not depend on cobra or pflag.
- **No cgo.** The binary must link statically and cross-compile, including for `js/wasm`.
- **Option order.** The command-line framework does not report the order in which options were
  given. A tool that needs GNU's "last one wins" either tracks order itself (`lsx`, `headx`,
  `tailx`) or resolves the conflict toward the most cautious option (FR-9).
- **Locale.** Sorting, case and width follow the C locale with UTF-8 text.
- **Working directory.** `cd` in the shell changes the working directory of the process,
  because that is what the tools resolve relative paths against. A library caller running
  scripts concurrently must account for it.
- **Browser limits.** In the playground there are no external programs, no real terminal and a
  single thread. A tool that waits forever is ended by resetting the session.

### 2.6 Assumptions and dependencies

The assumptions are listed in [business-requirements](business-requirements.md). The software
dependencies:

| Dependency                    | Role                                  | Linked into the binary |
|-------------------------------|---------------------------------------|------------------------|
| `github.com/spf13/cobra`      | Command structure and help            | Yes                    |
| `github.com/spf13/pflag`      | GNU-style option parsing              | Yes                    |
| `github.com/mattn/go-isatty`  | Terminal detection                    | Yes                    |
| `cobra/doc` and `go-md2man`   | Reference generation                  | No: only `tools/gendocs` imports them |
| Next.js, Fumadocs, Tailwind   | The website                           | No: a separate project in `web/`      |

## 3. External interfaces

### 3.1 Command line

```text
gnux [OPTION]... [TOOL [ARGUMENT]...]
TOOL [OPTION]... [OPERAND]...
gnux shell [OPTION]... [SCRIPT]
```

- **Selecting a tool.** If the name gnux was started under (directory and `.exe` stripped,
  lower-cased) is a tool's name or the GNU name it mirrors, that tool runs with the remaining
  arguments. Otherwise the root command runs and the first argument selects the tool, by either
  name.
- **Options of `gnux` itself.** `--help`, `--list` (every tool name, one per line) and
  `--version`.
- **Options of every tool.** `--help` and `--version`, plus the tool's own, listed in
  [functional-requirements](functional-requirements.md).
- **Options of `gnux shell`.** `-c STRING`, `--command=STRING`: run the commands in STRING.
  `-e`, `--errexit`: exit at the first command that fails.
- **Syntax.** Short options combine, long options take `--name=value`, and `--` ends option
  parsing. `echox`, `truex`, `falsex` and `yesx` parse their own arguments. `envx` stops parsing
  at the first operand.

### 3.2 Go API

The module is `github.com/braswelljr/gnux`. Everything under `core/` is public API.

| Package            | Symbols                                                                                              |
|--------------------|------------------------------------------------------------------------------------------------------|
| `core`             | `Streams`, `Stdio()`, `Discard()`, `IsTerminal`; `Diag` and `NewDiag`; `Prompter` and `NewPrompter`; `ErrFailed`, `ExitError`, `UsageError`, `Usagef`; `ErrText`, `Describe`, `IsRoot`, `Plural`; `Writable`, `DeviceID` |
| `core/<gnu>`       | `Name`, `Options` (where the tool has options), a constructor `New`, and `Run(args []string) error`  |
| `core/shell`       | `Shell`, `New(opts, exec, streams)`, `Options`, `ExecFunc`, `Shell.Run`, `Shell.RunString`, `Shell.Eval`, `Shell.Set` |
| `cmd`              | `Main(argv, streams) int`, `RunTool(name, args, streams) (status int, ok bool)`, `NewShell(opts, streams)`, `Tools()`, `Commands()`, `Lookup(name)`, `NewRoot(streams)`, `ShellSpec()` |

Three levels of use, from the smallest:

```go
// An engine: rmx -rf build dist
err := rm.New(rm.Options{Recursive: true, Force: true}, core.Stdio()).Run([]string{"build", "dist"})

// A tool by name, exactly as the command line would run it
status, ok := cmd.RunTool("rmx", []string{"-rf", "build"}, core.Stdio())

// A script, in-process
sh := cmd.NewShell(shell.Options{ErrExit: true}, core.Stdio())
err = sh.RunString("mkdirx -p dist && cpx -r assets dist/")
```

The contract every engine keeps is in section 5.

### 3.3 WebAssembly JavaScript API

`tools/wasm` is built for `js/wasm` and is not a command-line program. Once started it installs
an object on the JavaScript global and waits to be called.

| Member                     | Type                          | Meaning                                                         |
|----------------------------|-------------------------------|-----------------------------------------------------------------|
| `gnux.version`             | string                        | The gnux version                                                |
| `gnux.tools`               | array of strings              | Every tool name                                                 |
| `gnux.run(line)`           | returns a promise of a number | Run one shell line; resolves with its exit status               |
| `gnux.cwd()`               | returns a string              | The shell's working directory                                   |
| `gnux.onOutput`            | assignable function           | Called with 1 (standard output) or 2 (standard error) and a `Uint8Array` |
| `globalThis.gnuxReady`     | optional function             | Called with the API object once it is ready                     |

- Lines run in one persistent, interactive shell: variables and `cd` carry over, and a syntax
  error does not end the session.
- One line runs at a time.
- One line may write at most 1,048,576 bytes; past that, writes fail like a closed pipe and a
  truncation notice is written to standard error.
- The program reaches the filesystem through the Node-style `fs` and `process` objects that
  Go's `js/wasm` runtime expects on the global object. The host supplies them.

### 3.4 Files and streams

- **Standard input.** Read by `catx`, `headx`, `tailx` and `wcx` when no file operand, or `-`,
  is given; by prompts; and by `gnux shell` when no script is named.
- **Standard output.** A tool's results and `--verbose` reports.
- **Standard error.** Diagnostics, prompts, and the shell's interactive prompt.
- **Files.** A tool reads and writes only the operands it is given. gnux reads no configuration
  file and writes no state of its own.
- **Environment.** Read where a tool's behavior depends on it: `envx` (the whole environment),
  `pwdx -L` (`PWD`), the shell (variable lookup, the home directory), and `lsx` on a terminal.
- **Redirection in the shell** creates files with mode 0644 before the umask.

### 3.5 Exit statuses

| Status   | Meaning                                                                                        |
|----------|------------------------------------------------------------------------------------------------|
| 0        | Success                                                                                        |
| 1        | An operation failed and was reported; or a command-line mistake, for most tools; or `falsex`   |
| 2        | `lsx`: serious trouble, or a bad command line. `gnux shell`: a syntax error or a bad command line |
| 125      | `envx` itself failed, including a bad command line                                             |
| 126      | `envx`: the command was found but could not be run                                             |
| 127      | `envx`: the command was not found. `gnux shell`: an unknown command, or a script that cannot be opened |
| 128 + n  | `envx`: the command was killed by signal n                                                     |
| other    | `envx` and `gnux shell` relay the status of the last command run; `exit N` in a script sets it |

## 4. System features

Each feature lists the functional requirements that specify it. The requirements themselves,
with their rationale, acceptance criteria and verification, are in
[functional-requirements](functional-requirements.md).

| Feature                                   | Requirements                                            |
|-------------------------------------------|---------------------------------------------------------|
| 4.1 Behavior common to every tool         | FR-1 to FR-11, FR-73, FR-74                             |
| 4.2 The 22 tools                          | FR-51 to FR-72                                          |
| 4.3 Safeguards on destructive operations  | FR-12 to FR-16                                          |
| 4.4 Invocation and multi-call dispatch    | FR-17 to FR-22, FR-75, FR-76                            |
| 4.5 The portable shell                    | FR-23 to FR-29, FR-77 to FR-79                          |
| 4.6 The Go library                        | FR-30 to FR-35, FR-80, FR-81                            |
| 4.7 Installation and distribution         | FR-36 to FR-38                                          |
| 4.8 Documentation generation              | FR-39 to FR-44, FR-82, FR-83                            |
| 4.9 The website                           | FR-45, FR-46                                            |
| 4.10 The playground                       | FR-47, FR-48, FR-84, FR-85                              |
| 4.11 Search and sharing                   | FR-49, FR-50, FR-86                                     |

The tools, with the requirement that specifies each:

| Tool        | GNU        | Requirement | Tool        | GNU        | Requirement |
|-------------|------------|-------------|-------------|------------|-------------|
| `basenamex` | `basename` | FR-51       | `mvx`       | `mv`       | FR-62       |
| `catx`      | `cat`      | FR-52       | `pwdx`      | `pwd`      | FR-63       |
| `cpx`       | `cp`       | FR-53       | `realpathx` | `realpath` | FR-64       |
| `dirnamex`  | `dirname`  | FR-54       | `rmdirx`    | `rmdir`    | FR-65       |
| `echox`     | `echo`     | FR-55       | `rmx`       | `rm`       | FR-66       |
| `envx`      | `env`      | FR-56       | `sleepx`    | `sleep`    | FR-67       |
| `falsex`    | `false`    | FR-57       | `tailx`     | `tail`     | FR-68       |
| `headx`     | `head`     | FR-58       | `touchx`    | `touch`    | FR-69       |
| `lnx`       | `ln`       | FR-59       | `truex`     | `true`     | FR-70       |
| `lsx`       | `ls`       | FR-60       | `wcx`       | `wc`       | FR-71       |
| `mkdirx`    | `mkdir`    | FR-61       | `yesx`      | `yes`      | FR-72       |

## 5. Data and error conventions

### 5.1 The engine contract

```go
package rm

const Name = "rmx"                          // command name; prefix of every diagnostic

type Options struct{ /* mirrors the GNU flags; the zero value is the tool with no flags */ }
func New(opts Options, s core.Streams) *Rm  // build the engine
func (r *Rm) Run(args []string) error       // run against the operands
```

- The zero value of `Options` is the tool's default behavior, with every safeguard on.
- An engine reads and writes only through the `core.Streams` it was given.
- Diagnostics are written through `core.NewDiag(Name, s)`, in GNU's wording.
- A failing operand does not stop the remaining operands.
- Anything that needs a terminal, a clock or the environment is an exported field the caller
  can replace.

### 5.2 Returned errors

| Returned error     | Meaning                                                       | Exit status                            |
|--------------------|---------------------------------------------------------------|----------------------------------------|
| `nil`              | Success                                                       | 0                                      |
| `core.ErrFailed`   | It ran, something failed, and the diagnostics are written     | 1                                      |
| `*core.ExitError`  | A specific, silent status                                     | Its `Code`                             |
| `*core.UsageError` | A command-line mistake; built with `core.Usagef`              | The tool's usage status: 1 unless stated |

The usage status is 2 for `lsx` and `gnux shell`, and 125 for `envx`. Any other error, such as
one from option parsing, is treated as a usage error.

### 5.3 Diagnostics

- **Form.** `<tool>: <message>` and a newline, on standard error. A gnux command that is not a
  tool is prefixed `gnux <command>` when the command line reports its usage error.
- **System errors** are rendered by `core.ErrText`: the bare, capitalized message with the path
  and operation removed. The common cases have one fixed text on every platform: `No such file
  or directory`, `Permission denied`, `File exists`, `Directory not empty`.
- **File types** in prompts and messages come from `core.Describe`: `regular file`, `regular
  empty file`, `directory`, `symbolic link`, `fifo`, `socket`, `character special file`, `block
  special file`.
- **Usage errors** are followed by `Try '<tool> --help' for more information.`, printed by
  `cmd.Main` and never by an engine.
- **Operands** are quoted in single quotes.

### 5.4 Prompts

A prompt is `<tool>: <question> ` on standard error. The answer is read from standard input and
is affirmative when it begins with `y` or `Y`; anything else, and end of input, is no.

### 5.5 Version

`gnux --version` prints `gnux VERSION`. A tool prints `TOOL (gnux) VERSION`. The version is the
output of `git describe --tags --always --dirty` injected at link time, or `dev` when the binary
was built without that flag.

## 6. Quality attributes

Each attribute lists the non-functional requirements that specify it, in
[non-functional-requirements](non-functional-requirements.md).

| Attribute                         | Requirements                                  |
|-----------------------------------|-----------------------------------------------|
| Compatibility and portability     | NFR-1 to NFR-6                                |
| Performance                       | NFR-7 to NFR-11                               |
| Reliability and data safety       | NFR-12 to NFR-16, NFR-36                      |
| Security and privacy              | NFR-17 to NFR-21, NFR-37, NFR-38              |
| Usability                         | NFR-22, NFR-23                                |
| Accessibility                     | NFR-24                                        |
| Maintainability                   | NFR-25 to NFR-29, NFR-39 to NFR-42            |
| Testability                       | NFR-30 to NFR-33, NFR-43                      |
| Observability of failures         | NFR-44 to NFR-46                              |
| Search and discoverability        | NFR-34, NFR-35                                |
| Licensing and compliance          | NFR-47                                        |

## 7. Traceability matrix

From each business requirement, through the requirements that realize it, to what verifies
them. "Review" means a person checks it and nothing fails automatically.

| Business requirement              | Functional                                           | Non-functional                              | Verification                                                                                   |
|-----------------------------------|------------------------------------------------------|---------------------------------------------|------------------------------------------------------------------------------------------------|
| BR-1 GNU-compatible everywhere    | FR-1 to FR-11, FR-51 to FR-74                        | NFR-6, NFR-22, NFR-23, NFR-44 to NFR-46     | `core/<tool>/*_test.go`, `cmd/<tool>x/*_test.go`, `cmd/main_test.go`                           |
| BR-2 no platform differences      | FR-29, FR-76                                         | NFR-1 to NFR-6                              | Test workflow on three operating systems; Build workflow; `TestErrText`                        |
| BR-3 adoptable in one step        | FR-17 to FR-22, FR-36 to FR-38                       | NFR-2, NFR-20, NFR-21                       | Build workflow; smoke test in the Test workflow; review of the installers                      |
| BR-4 useful to programs           | FR-30 to FR-35, FR-80, FR-81                         | NFR-16, NFR-30, NFR-36, NFR-39              | Engine suites; `core/rm/example_test.go`; `cmd/shell_test.go`                                  |
| BR-5 portable automation          | FR-23 to FR-29, FR-77 to FR-79                       | NFR-18                                      | `core/shell/shell_test.go`; `cmd/shell_test.go`                                                |
| BR-6 easy to evaluate and learn   | FR-39, FR-40, FR-44 to FR-48, FR-84, FR-85           | NFR-11, NFR-19, NFR-24, NFR-33              | Web workflow; `web/test/*.test.ts`; accessibility by review only                               |
| BR-7 discoverable                 | FR-49, FR-50, FR-86                                  | NFR-34, NFR-35                              | Web workflow builds the routes; content by review                                              |
| BR-8 safe with destructive work   | FR-9, FR-12 to FR-16, FR-73                          | NFR-12 to NFR-14, NFR-36                    | `core/rm`, `core/cp`, `core/mv`, `core/ln` suites; adapter precedence tests                    |
| BR-9 inexpensive to extend        | FR-41, FR-82, FR-83                                  | NFR-25 to NFR-29, NFR-40 to NFR-43          | `TestRegistryIsConsistent`; `cmd/docs/docs_test.go`; Lint workflow                             |
| BR-10 free to redistribute        | None                                                 | NFR-47                                      | Review of `LICENSE` and `go.mod`                                                               |
| BR-11 honest about differences    | FR-10, and the Deviations entry of FR-51 to FR-72    | NFR-5                                       | Review                                                                                         |
| BR-12 documentation cannot drift  | FR-41 to FR-43, FR-46, FR-82                         | NFR-29                                      | Docs workflow; `make docs/check`; `TestGenerateAllDeterministic`                               |

Requirements with no automated verification, gathered in one place so they are not mistaken for
covered:

| Requirement                      | What is missing                                                                  |
|----------------------------------|----------------------------------------------------------------------------------|
| NFR-8 start-up time              | No benchmark or timing check                                                     |
| NFR-24 accessibility             | No audit and no browser test                                                     |
| NFR-17, NFR-39, NFR-40           | No check of import rules (network packages, cobra under `core/`, the generator)  |
| NFR-27 documented exports        | The linter rule that would enforce it is not enabled                             |
| NFR-42, em dashes and markdown   | No check                                                                         |
| NFR-21, FR-36, FR-37             | The container image, `install.sh` and the link-creating targets are not run in CI |
| NFR-47, BR-10                    | No license scan                                                                  |
| FR-10, BR-11                     | Nothing compares a tool's options with GNU's                                     |
| FR-12, root half                 | The refusal to recurse on `/` has no test                                        |
| FR-14, `cpx` half                | The two `cpx` refusals have no test                                              |
| FR-38                            | Tests check that a version is printed, not which; `go install` builds report `dev` |
| FR-40, FR-43, FR-44              | Documentation content, by review                                                 |
| FR-45, FR-48, FR-49, FR-50, FR-86, NFR-34, NFR-35, NFR-38 | The Web workflow builds them; nothing asserts on what they produce |

## 8. Glossary

| Term             | Meaning                                                                                                   |
|------------------|-----------------------------------------------------------------------------------------------------------|
| Adapter          | The `cmd/<gnu>x` package of a tool: its flags, their mapping to engine options, and its documentation     |
| Engine           | The `core/<gnu>` package of a tool: all of its behavior, with no knowledge of flags                       |
| GNU name         | The name of the GNU counterpart (`rm`). Accepted as an alias wherever the gnux name is                    |
| Multi-call binary | One executable that behaves as different programs depending on the name it was started under            |
| Operand          | A command-line argument that is not an option: a file, a directory, a string                              |
| Registry         | `cmd.Tools()` in `cmd/registry.go`: the single list of tool specifications                                |
| Spec             | `tool.Spec`: a tool's metadata, GNU name, flag registration and entry point                               |
| Topic            | A per-flag documentation page declared on a spec (`rmx` has one per flag)                                 |
| Streams          | `core.Streams`: the standard input, output and error an engine reads and writes                           |
| Usage error      | A command-line mistake: a missing operand, an unknown option, an invalid value                            |
| Diagnostic       | A line on standard error of the form `<tool>: <message>`                                                  |
| Injection point  | An exported engine field that replaces a terminal, a clock, the environment or a system call in tests     |
| Raw arguments    | A tool that receives its arguments unparsed (`echox`, `truex`, `falsex`, `yesx`)                          |
| Playground       | The website page that runs the WebAssembly build of gnux against an in-memory filesystem                  |
| Reference        | The generated documentation of gnux, the guide, every tool and the shell                                  |
| Tool             | A gnux command that mirrors a GNU program. `gnux shell` is a command but not a tool                       |
