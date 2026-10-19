# Business Requirements

Business requirements state why gnux exists and what it must achieve for the people who use,
embed, extend and maintain it. Each requirement has a stable identifier (`BR-n`), a rationale,
acceptance criteria and the check that verifies it, so that a proposed change can be judged
against an outcome rather than against taste.

**This document stands on its own.** It is the first thing to read about why the project exists.
What the software must do in detail is in
[functional-requirements](functional-requirements.md); how well it must do it is in
[non-functional-requirements](non-functional-requirements.md); the
[software-requirements-specification](software-requirements-specification.md) ties the three
together and carries the traceability matrix.

Requirement levels follow the usual convention: **must** is mandatory, **should** is expected
unless there is a stated reason not to, **may** is optional.

## Vision

The GNU core utilities' interface, everywhere. gnux is a portable, GNU-compatible
implementation that makes the GNU command-line interface available on every machine and inside
every Go program, without needing GNU coreutils installed. It complements GNU coreutils rather
than replacing them. A person who knows `rm`, `cp` or `ls` already knows `rmx`, `cpx` and `lsx`:
the flags, the behavior, the messages and the exit statuses are the same, on Linux, macOS,
Windows, in containers and in the browser alike. The same code is offered three ways
(as commands, as a Go library, and as a portable shell) and is documented and demonstrated on a
website with an in-browser playground.

## Problem statement

The GNU core utilities are the common language of file work, and their interface is not
available everywhere it is needed.

- **Windows does not ship them.** A script that calls `rm -rf build` or `cp -r assets dist`
  needs a compatibility layer (WSL, Git Bash, Cygwin) installed and on `PATH` before it runs.
- **macOS ships different ones.** The BSD tools share names with the GNU tools and differ in
  options and messages, so a command that works on a Linux runner can fail or behave
  differently on a developer's laptop.
- **Minimal containers ship few of them, or a reduced implementation.** A build step that
  works in one image breaks in another.
- **A program that needs their semantics has to spawn them.** A Go application that wants
  `rm -r` with its safeguards, or `cp -a` with its attribute handling, either shells out to a
  tool that may not exist or reimplements the behavior and gets the edge cases wrong.
- **Build and CI scripts depend on a system shell.** The script is only as portable as the
  shell and the utilities found on each machine.

gnux answers all five by bringing the GNU tools' interface to every platform: one statically
linked binary, the same engines exposed as Go
packages, and an interpreter that runs a script using nothing but that binary.

## Objectives and measures of success

| ID    | Objective                                                        | Measure of success                                                                      | Requirements        |
|-------|------------------------------------------------------------------|-----------------------------------------------------------------------------------------|---------------------|
| OBJ-1 | Be GNU-compatible for the tools it implements                    | An existing command works with only the tool name changed                               | BR-1, BR-8, BR-11   |
| OBJ-2 | Erase platform differences in everyday file work                 | The same command line gives the same result on every supported platform                 | BR-2, BR-5          |
| OBJ-3 | Cost nothing to adopt                                            | One command installs it; nothing else has to be installed or configured                 | BR-3, BR-10         |
| OBJ-4 | Serve programs as well as people                                 | A Go program calls an engine, a tool by name, or a script without starting a process    | BR-4                |
| OBJ-5 | Be easy to evaluate, learn and find                              | A visitor reads the reference and runs real commands in a browser, found through search | BR-6, BR-7          |
| OBJ-6 | Stay cheap to extend toward the rest of coreutils                | A new tool is two packages and one registry line; its documentation follows unaided     | BR-9, BR-12         |

## Stakeholders and their needs

| Who                         | What they need from gnux                                                                    | Served by                |
|-----------------------------|---------------------------------------------------------------------------------------------|--------------------------|
| Command-line users          | Familiar tools that behave identically on every machine they use                            | BR-1, BR-2, BR-8         |
| Windows and container users | The GNU interface where coreutils is not installed, without a compatibility layer           | BR-2, BR-3               |
| Script and CI authors       | One script that runs unchanged on every developer machine and runner                        | BR-5                     |
| Go developers               | File operations with GNU semantics as a library, without spawning processes                 | BR-4                     |
| Contributors                | A small, uniform structure in which adding a tool is routine                                | BR-9, BR-12              |
| Evaluators and learners     | Documentation and a way to try the tools without installing anything                        | BR-6, BR-7               |
| Packagers and redistributors | A permissive license, a static binary and a reproducible way to build it                   | BR-3, BR-10              |
| The maintainer              | A codebase that stays correct and cheap to maintain as it grows                             | BR-9, BR-12              |

## Scope

**In scope.**

- The 89 tools listed in the README: nearly all of GNU coreutils, and the four tools a build
  or CI script uses beside them, `grepx` (GNU grep), `sedx` (GNU sed), and `findx` and `xargsx`
  (GNU findutils). They are in scope because a script that leaves gnux for the system's `find`
  or `sed` is no longer portable; other GNU packages are not.
- The multi-call binary `gnux`, including dispatch on the name it was started under.
- The Go API: the engines under `core/`, and `cmd.Main`, `cmd.RunTool` and `cmd.NewShell`.
- `gnux shell`, the portable script interpreter.
- The generated reference in four formats, and the hand-written documents beside it.
- The website in `web/`: documentation, search, the playground and its WebAssembly build.
- Installation by `go install`, from source with `make`, by `install.sh`, and as a container
  image.

**Out of scope.**

- A general-purpose shell. `gnux shell` has the control flow a build script needs (`if`, `for`,
  `while`, `case`) and command substitution, and stops there: no functions, arithmetic, job
  control or here-documents.
- Locale-dependent behavior. Sorting, case and character classes follow the C locale with
  UTF-8 text.
- SELinux contexts and ACLs.
- Bit-for-bit reproduction of GNU `--help` text. The options and their meanings match; the
  wording of the help screen is gnux's own.
- Every option of every GNU tool. Options that are not implemented are named in
  [functional-requirements](functional-requirements.md), tool by tool.
- The rest of coreutils. The tools still to be written are listed in the README's roadmap and
  are not requirements of the current product.
- Package-manager distribution (Homebrew, Scoop, apt and the like). Prebuilt archives are
  published on GitHub releases by the Release workflow; nothing submits them to a package
  manager.

## The requirements

### BR-1: GNU-compatible everywhere

gnux must be GNU-compatible, on every platform it supports, for the tools it implements:
compatible with scripts and habits formed on the GNU tools.

- **Rationale.** The value of the project is that nobody has to learn it. A tool that behaves
  almost like `rm` is worse than no tool, because the difference is discovered in production.
- **Acceptance.** For an implemented tool, an existing command line works with only the tool
  name changed (or unchanged, when the binary is linked under the GNU name), and produces the
  same result, the same diagnostic wording and the same exit status, within the documented
  deviations.
- **Verified by.** The engine suites in `core/<tool>/*_test.go`, which assert on exact standard
  output and standard error; `cmd/main_test.go` for dispatch under both names. Comparison
  against a GNU binary is not automated: fidelity rests on tests written from GNU's documented
  behavior, and on review.

### BR-2: no platform differences in everyday file work

gnux must remove platform differences from everyday file work.

- **Rationale.** The problem being solved is that the same command means different things on
  different machines.
- **Acceptance.** The same command line gives the same result on Linux, macOS and Windows, given
  the same files. Where a platform lacks a concept (owners, inodes or unprivileged symbolic
  links on Windows) the tool degrades predictably and its reference page says so.
- **Verified by.** The Test workflow, which runs the whole suite on `ubuntu-latest`,
  `macos-latest` and `windows-latest`; the Build workflow, which cross-compiles for every
  supported platform.

### BR-3: adoptable in one step

gnux must be adoptable in one step.

- **Rationale.** A tool that needs a runtime, a service or configuration cannot bring the GNU
  interface to a machine that has neither.
- **Acceptance.** One command installs it. The binary is statically linked, has no runtime
  dependencies and reads no configuration file. Making each tool a command of its own is one
  further command, or none when using `install.sh` or `make install/links`.
- **Verified by.** The Build workflow (`CGO_ENABLED=0` on every target); the multi-call smoke
  test in the Test workflow; review of `install.sh`, the `Makefile` and the `Dockerfile`.

### BR-4: useful to programs as well as people

gnux must be useful to programs as well as people.

- **Rationale.** The semantics are the hard part, and a Go program that needs them should not
  have to spawn a process that may not exist on the machine.
- **Acceptance.** A Go application can call any tool's engine through a documented package
  under `core/`, run a tool by name, and run a whole script, all in-process, with its own
  streams.
- **Verified by.** The engine suites, which use only the public API; `core/rm/example_test.go`;
  `TestRunTool` and `TestNewShell` in `cmd/shell_test.go`.

### BR-5: portable automation without a system shell

gnux must make portable automation possible without depending on a system shell.

- **Rationale.** A build script is only as portable as the shell and utilities it finds.
- **Acceptance.** A `gnux shell` script runs unchanged on every supported platform, and the only
  thing the machine needs is the gnux binary.
- **Verified by.** `core/shell/shell_test.go` and `cmd/shell_test.go`, on all three operating
  systems in the Test workflow.

### BR-6: easy to evaluate and learn

gnux must be easy to evaluate and learn.

- **Rationale.** Nobody installs a file-deleting tool on trust.
- **Acceptance.** A visitor can read the reference for every tool and run real commands in a
  browser, against files that exist only in the page's memory.
- **Verified by.** The Web workflow, which tests the playground's filesystem against the real
  WebAssembly build and builds the site; `cmd/docs/docs_test.go` for the generated content.

### BR-7: discoverable

gnux must be discoverable.

- **Rationale.** A tool nobody can find helps nobody.
- **Acceptance.** The website is indexable, with a unique title and description on every page,
  canonical URLs, a sitemap, `robots.txt` and structured data.
- **Verified by.** The Web workflow builds the routes that produce them. Whether the output is
  correct for a crawler is checked by review; there is no automated SEO test. The production
  origin is still a placeholder in `web/src/config/site.ts` and must be confirmed before launch.

### BR-8: safe to trust with destructive operations

gnux must be safe to trust with destructive operations.

- **Rationale.** These tools delete and overwrite. One surprising loss ends adoption.
- **Acceptance.** Dangerous defaults match GNU's safeguards (the root failsafe, the refusal of
  `.` and `..`, prompting when asked, never clobbering when told not to). No data is lost through
  behavior that is specific to gnux, and where gnux cannot reproduce GNU's resolution of
  conflicting options it chooses the more cautious one.
- **Verified by.** `core/rm`, `core/cp`, `core/mv` and `core/ln` suites; the adapter suites that
  pin each precedence rule.

### BR-9: inexpensive to extend

gnux must stay inexpensive to extend toward the rest of coreutils.

- **Rationale.** The roadmap is several times the size of what is built. If the fortieth tool
  costs more than the fourth, the project stops.
- **Acceptance.** Adding a tool touches two packages and one registry line. The root command,
  dispatch by name, the reference in every format and the website page follow from the
  registration with no second list to update.
- **Verified by.** `TestRegistryIsConsistent` and `TestEveryToolAnswersHelpAndVersion` in
  `cmd/main_test.go`; `TestGenerateAll` and `TestGenerateMDX` in `cmd/docs/docs_test.go`.

### BR-10: free to use, modify and redistribute

gnux must be free to use, modify and redistribute.

- **Rationale.** It is meant to be bundled into images, toolchains and other programs.
- **Acceptance.** Released under the MIT license, with no copyleft dependency linked into the
  binary.
- **Verified by.** Review of `LICENSE` and `go.mod`. There is no automated license scan.

### BR-11: honest about its differences

gnux must state plainly where it differs from its GNU counterpart.

- **Rationale.** A compatibility claim is only credible when its limits are written down. An
  undocumented difference is a bug report waiting for a user to pay for it.
- **Acceptance.** Every deliberate deviation, and every platform limitation, is stated in the
  tool's reference page. Options of the GNU tool that are not implemented are listed in
  [functional-requirements](functional-requirements.md).
- **Verified by.** Review. The reference pages state behavior differences; they do not yet list
  unimplemented options, which is why the requirement register carries that list.

### BR-12: documentation that cannot drift

The documentation of what gnux does must be derived from the code, and checked.

- **Rationale.** Reference documentation written beside the code describes the previous
  release. People stop trusting it, and then stop reading it.
- **Acceptance.** The reference for every tool is generated from the same definitions the
  program runs on, is committed, and a change whose committed copy is stale fails CI.
- **Verified by.** The Docs workflow and `make docs/check`; `TestGenerateAllDeterministic`.

## Business rules

Rules the product holds to whatever a single change would prefer. Each is carried in detail by
a file in [rules](../rules/README.md).

| ID      | Rule                                                                                                              |
|---------|-------------------------------------------------------------------------------------------------------------------|
| BRULE-1 | A tool is named after its GNU counterpart with an `x` appended, and nothing else                                 |
| BRULE-2 | GNU's behavior is the specification. Where gnux differs, the difference is deliberate and documented              |
| BRULE-3 | When conflicting options cannot be resolved the way GNU resolves them, the most cautious option wins              |
| BRULE-4 | A tool never reports success for an operation that failed                                                         |
| BRULE-5 | gnux reads and writes only what the user named. It does not use the network, collect telemetry or keep state     |
| BRULE-6 | The shell never runs an external program implicitly                                                               |
| BRULE-7 | There is one list of tools. The program, the dispatcher, the reference and the website are built from it          |
| BRULE-8 | Nothing in the reference is written by hand                                                                       |
| BRULE-9 | The playground never touches the visitor's files                                                                  |
| BRULE-10 | gnux presents itself as an independent, GNU-compatible project that complements GNU coreutils, never as a replacement for them, as part of GNU, or as endorsed by the GNU Project or the Free Software Foundation |

## Assumptions

| ID  | Assumption                                                                                                         | If it stops holding                                                        |
|-----|--------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------|
| A-1 | Users know the GNU tools, or will learn from GNU's own documentation                                               | The reference would have to teach, not only specify                        |
| A-2 | Installing from source is acceptable, so the Go toolchain is available wherever gnux is installed                  | Prebuilt, published binaries become a requirement                          |
| A-3 | Text is UTF-8 and the C locale's ordering is acceptable                                                            | Locale support moves into scope                                            |
| A-4 | The per-tool commands can be created as symbolic links, or as copies of the binary where links are unavailable     | Another dispatch mechanism is needed                                       |
| A-5 | A browser with WebAssembly support is available to anyone using the playground                                     | The playground shows an error and the documentation remains usable         |
| A-6 | GNU's documented behavior is stable enough to be a specification                                                   | Each tool needs a statement of the GNU version it mirrors                  |

## Constraints

| ID  | Constraint                                                                                                                                 |
|-----|--------------------------------------------------------------------------------------------------------------------------------------------|
| C-1 | Written in Go, at the version pinned in `go.mod`, with the standard library plus cobra and pflag (command line), go-isatty (terminals) and golang.org/x/sys and x/crypto (system calls, BLAKE2b) |
| C-2 | No cgo anywhere. The binary must link statically and cross-compile for Linux, macOS, Windows and `js/wasm`                                  |
| C-3 | Engines must not depend on the command-line framework                                                                                       |
| C-4 | The command-line framework does not report the order in which options were given. A tool that needs it tracks it, or applies BRULE-3       |
| C-5 | `cd` in the shell changes the working directory of the whole process, because that is what the tools resolve relative paths against        |
| C-6 | In a browser there are no external programs, no real terminal and a single thread                                                           |
| C-7 | One maintainer. Anything that must be remembered rather than checked will eventually be forgotten                                           |
| C-8 | gnux is independent of, and not affiliated with or endorsed by, the GNU Project or the Free Software Foundation. It does not use the GNU logo or other GNU or FSF marks, and refers to GNU only to name the interface it is compatible with |

## Risks and mitigations

| ID  | Risk                                                                              | Mitigation                                                                                                   |
|-----|-----------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------|
| R-1 | A gnux tool behaves differently from GNU in a way that loses data                 | Safeguards on by default and tested; the most cautious option wins a conflict; deviations are documented     |
| R-2 | Behavior drifts between platforms                                                 | The suite runs on Linux, macOS and Windows on every push and pull request; platform code is build-tagged     |
| R-3 | A change breaks the WebAssembly build, and with it the playground                 | The Test workflow compiles `tools/wasm` for `js/wasm`; the Web workflow tests against the real binary        |
| R-4 | The reference drifts from the code                                                | It is generated from the registry, committed, and checked for freshness in CI                                |
| R-5 | The cost of a tool rises as the count grows                                       | One contract for every tool; registry invariants are tests; the reference follows registration               |
| R-6 | Users assume full GNU coverage and find a missing option in production            | Unimplemented options are listed per tool; an unknown option is a usage error, never silently ignored        |
| R-7 | A dependency introduces a vulnerability or an incompatible license                | Three direct dependencies; Dependabot keeps them current; no automated license scan exists yet               |
| R-8 | The website is launched with a placeholder origin, so canonical URLs are wrong    | `NEXT_PUBLIC_SITE_URL` must be set at build time; the fallback in `site.ts` is marked as a placeholder       |
| R-9 | A version cannot be identified in a bug report                                    | The version is injected at link time by `make`, the Dockerfile and CI. A plain `go install` reports `dev`    |

## Success metrics

These are the observable signs that the objectives are being met. None has a numeric target:
the project has not set any, and inventing them here would be worse than leaving them out.

| Metric                                                         | Where it is read                                                    |
|----------------------------------------------------------------|---------------------------------------------------------------------|
| Tools implemented, against the roadmap                         | `gnux --list` (22 today) and the README's roadmap                   |
| Supported platforms that build and pass the suite              | The Build and Test workflows                                        |
| Requirements with an automated check                           | The verification entries in the three requirement registers        |
| Documented deviations per tool                                 | The per-tool sections of the functional requirements                |
| Time for a contributor to add a tool                           | Not measured. [adding-a-tool](../workflows/adding-a-tool.md) is the path |
| Reference pages that match the code                            | The Docs workflow: all of them, or the change does not merge        |
