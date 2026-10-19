# AGENTS.md

This file provides guidance to agents when working with code in this repository.

## Project Overview

**gnux** (`github.com/braswelljr/gnux`) is a portable, GNU-compatible implementation of the GNU
core utilities in Go. It brings the GNU command-line interface (same flags, messages, exit
statuses) to every platform without needing GNU coreutils installed, and complements rather than
replaces GNU coreutils; it is not affiliated with or endorsed by the GNU Project or the FSF. It is
usable three ways -

- **as commands**: one multi-call binary, `gnux`, carrying every tool, each also a command of its
  own named after the GNU tool with an `x` appended (`rm` → `rmx`, `cp` → `cpx`), obtained by
  linking the binary to that name;
- **as a Go library**: each tool's engine is an importable package under `core/`;
- **as a portable shell**: `gnux shell` interprets a small POSIX-sh subset whose commands are the
  tools, run in-process, so one script behaves the same on every platform.

A website in `web/` (Fumadocs documentation + a playground running the tools as WebAssembly)
sits on top of the same code.

Every tool mirrors its GNU counterpart's flag surface, semantics, diagnostics and exit statuses.

## Architecture at a glance

Each tool is two packages:

- **`core/<gnu>/`** - the **engine** (`core/rm`, `core/cp`, …). Owns all behavior: filesystem
  work, prompting, diagnostics, concurrency. It knows nothing about cobra or flags; it takes an
  `Options` struct and a `core.Streams`. This is the public library API.
- **`cmd/<gnu>x/`** - the **adapter** (`cmd/rmx`, `cmd/cpx`, …). Declares the flags, folds the
  parsed values into the engine's `Options`, and carries the tool's documentation. No traversal
  or I/O logic lives here.

Shared pieces:

- **`core/`** (root package) - plumbing every engine uses: `Streams`, `Diag`, `Prompter`, the
  error conventions, `ErrText`/`Describe`, and build-tagged filesystem helpers.
- **`internal/tool`** - `tool.Spec`, the description of a tool, and `Spec.Command`, which turns
  it into a cobra command (adding `--help`/`--version`).
- **`cmd/registry.go`** - `Tools()`, the single list of specs. The root command, the argv[0]
  dispatcher and the doc generator are all built from it.

## Project Structure

```bash
gnux/
  main.go                  # os.Exit(cmd.Main(os.Args, core.Stdio())); //go:generate hook for docs
  cmd/
    main.go                # Main(argv, streams) int - argv[0] dispatch + error→exit-status mapping
    root.go                # NewRoot(streams) - gnux root command, one subcommand per tool
    registry.go            # Tools() / Commands() / Lookup(name) - THE list of tools
    shell.go               # `gnux shell` command (ShellSpec) + NewShell for library use
    docs/                  # Reference generator (cobra/doc: md/man/yaml/rest); tool-only dep
    <gnu>x/                # One adapter package per tool: Spec, Meta, Register, buildOptions, run
    rmx/<flag>/            # rmx alone keeps one sub-package per flag (force, interactive, …)
  core/
    core.go                # Streams, Diag, Prompter, ErrFailed/ExitError/UsageError, ErrText, …
    fs_unix.go             # Writable()/DeviceID() via syscall (build tag unix)
    fs_other.go            # Writable()/DeviceID() fallback (build tag !unix: Windows, wasm)
    <gnu>/                 # One engine package per tool: Name, Options, New, Run (+ tests)
    shell/                 # The gnux shell interpreter: parse.go, expand.go, shell.go
  internal/
    common/                # AppName ("gnux") + link-time Version
    docmeta/               # zero-dep doc metadata (Meta, Example) for tools and flags
    tool/                  # tool.Spec / tool.Topic, Spec.Command, --help/--version handling
  tools/gendocs/           # `go run ./tools/gendocs [dir] [format]` - reference generator
  tools/wasm/              # js/wasm entry point for the playground (build tag js && wasm)
  tools/changelog/         # `go run ./tools/changelog vX.Y.Z` - a release's CHANGELOG.md section
  tools/benchcheck/        # `go run ./tools/benchcheck OLD NEW` - fail on a benchmark regression
  CHANGELOG.md             # Release history (Keep a Changelog); published as /docs/changelog
  web/                     # Next.js + Fumadocs site and playground (own go.mod: not part of ./...)
    content/docs/          # Generated MDX (committed) - never edit by hand
    public/                # gnux.wasm + wasm_exec.js land here (`make web/wasm`; not committed)
  documentation/           # The published documentation:
    guides/                # Hand-written usage docs: getting started, cookbook, scripting, library, ...
    <format>/              # Generated reference (committed, deterministic) - never edit by hand:
                           #   {index.*, guide.*, <tool>/index.*, <tool>/flags/<flag>.*}
  docs/                    # Hand-written contributor documentation
    README.md              # Map of docs/ and documentation/
    requirements/          # Hand-written SRS + business / functional / non-functional registers
    workflows/             # Hand-written: how each process runs inside the code
    rules/                 # Hand-written: the cross-cutting rules, one file each (indexed below)
  action.yml               # setup-gnux: the composite GitHub Action (uses: braswelljr/gnux@v0)
  .goreleaser.yaml         # What a release is: archives, deb/rpm/apk, checksums, Homebrew/Scoop/winget/snap/AUR
  .github/workflows/       # test.yml, build.yml, lint.yml, docs.yml, web.yml, bench.yml, action.yml, release.yml
```

## Tools

`archx`, `b2sumx`, `base32x`, `base64x`, `basenamex`, `catx`, `chgrpx`, `chmodx`, `chownx`,
`cksumx`, `commx`, `cpx`, `csplitx`, `cutx`, `datex`, `ddx`, `dfx`, `dirnamex`, `dirx`, `dux`,
`echox`, `envx`, `expandx`, `exprx`, `factorx`, `falsex`, `findx`, `fmtx`, `foldx`, `grepx`,
`groupsx`, `headx`, `hostnamex`, `idx`, `installx`, `joinx`, `killx`, `linkx`, `lnx`, `lognamex`,
`lsx`, `md5sumx`, `mkdirx`, `mktempx`, `mvx`, `nicex`, `nlx`, `nohupx`, `nprocx`, `odx`, `pastex`,
`printenvx`, `printfx`, `pwdx`, `readlinkx`, `realpathx`, `rmdirx`, `rmx`, `sedx`, `seqx`,
`sha1sumx`, `sha224sumx`, `sha256sumx`, `sha384sumx`, `sha512sumx`, `shufx`, `sleepx`, `sortx`,
`splitx`, `statx`, `sumx`, `tacx`, `tailx`, `teex`, `testx`, `timeoutx`, `touchx`, `truex`,
`truncatex`, `trx`, `tsortx`, `ttyx`, `unamex`, `unexpandx`, `uniqx`, `unlinkx`, `uptimex`,
`usersx`, `vdirx`, `wcx`, `whoamix`, `whox`, `xargsx`, `yesx`. The README's roadmap lists what is not implemented yet.

`grepx`, `sedx`, `findx` and `xargsx` mirror GNU grep, GNU sed and GNU findutils, not coreutils:
they are the tools a build script reaches for next. `xargsx` and `findx -exec` run a command
named after a gnux tool in-process (`tool.RunTool`, wired to `cmd.RunTool`) and anything else
as a program on `PATH`, through `core/internal/run`.

## Development Commands

```bash
make build         # go build -o ./bin/gnux .   (the binary lives in bin/ and is run from there)
make links         # build, then symlink every tool name in ./bin to gnux
make install       # go install (with version ldflags)
make install/links # install, then symlink every tool name in GOBIN
make test          # go test -race ./...
make lint          # golangci-lint run
make format        # golangci-lint fmt (gofmt -s + goimports, in .golangci.yml), then Prettier in web/
make hooks         # lefthook install: the Git hooks in lefthook.yml
make docs          # regenerate documentation/ (all formats) and web/content/docs (also: go generate ./...)
make ci            # vet + lint + test + docs freshness check
make bench         # benchmark the real tools into bench/latest.txt (BENCH=regexp, COUNT=n)
make bench/save    # keep the latest results as the baseline; make bench/compare runs benchstat
make bench/check   # fail if bench/latest.txt regressed against bench/baseline.txt (tools/benchcheck)
make fuzz          # fuzz every parser for FUZZTIME each (default 30s)
make conformance   # compare every case in cmd/testdata/conformance with the GNU tools (Linux)
make conformance/docker # the same, inside a Debian container with current GNU releases
make web/wasm      # compile gnux to web/public/gnux.wasm (+ wasm_exec.js)
make web/dev       # wasm + docs content, then `pnpm dev` in web/
make web/test      # test the playground's in-memory fs against the real wasm
make web/build     # production build of the website
make release/check # validate .goreleaser.yaml (needs goreleaser v2)
make release/snapshot # every release archive and manifest into dist/, nothing published
make completions   # bash/zsh/fish/PowerShell completion scripts into completions/

# Always build into bin/ and run ./bin/gnux (or the links make links puts beside it). A bare
# `go build` at the root drops a `gnux` binary there: .gitignore keeps it out of a commit, and
# nothing in the Makefile, the hooks or CI builds or runs one outside bin/.

# Version is injected at link time:
go build -ldflags "-X github.com/braswelljr/gnux/internal/common.Version=$(git describe --tags --always)" -o bin/gnux .
```

## Requirements

- **Go 1.26+** (pinned in `go.mod`; CI reads it from there) and `git`.
- **golangci-lint v2** for `make lint` and `make format`; **lefthook** for the Git hooks
  (`make hooks`): on commit, staged Go files are formatted and their packages linted, and staged
  web files go through Prettier and ESLint; on push, `make docs/check` fails if the generated
  docs (`documentation/` and `web/content/docs/`) do not match the code.
- **Node.js 22.18+ and pnpm**, only for `web/` (pinned in `web/package.json`).
- No cgo anywhere: the binary must stay statically linked and cross-compile for linux, darwin,
  freebsd, windows **and js/wasm** (the playground). Releases cover linux amd64/arm64/riscv64,
  darwin and freebsd amd64/arm64, and windows amd64/arm64. Anything platform-specific needs a fallback that
  compiles everywhere - prefer the `unix` / `!unix` build tags over listing operating systems.

These are mirrored for users in the README's Requirements section and the guide's
`Requirements` section (`cmd/docs/guide.go`); change all three together.

## Tech Stack

- **Go 1.26** - module path `github.com/braswelljr/gnux`
- **Cobra / pflag** - CLI framework; `cobra/doc` for reference generation
- **mattn/go-isatty** - terminal detection
- **golang.org/x/sys** - Windows and BSD system calls the standard library lacks (statfs,
  sysctl, priority classes, sessions); **golang.org/x/crypto** - BLAKE2b for `b2sumx`
- **syscall** - permission, device and stat details; always build-tagged
- Tests use the standard library only (`testing`, `t.TempDir()`, injected funcs)

## How a tool is invoked

`cmd.Main` looks at `argv[0]` (directory and `.exe` stripped, lower-cased):

- if it names a tool - by its own name (`rmx`) or the GNU name (`rm`) - that tool's command runs
  directly with the remaining arguments;
- otherwise the first argument picks the tool (`gnux rmx …`, or `gnux rm …`) and it runs just as
  directly, as does `gnux shell`; the root command, which carries every tool, is built only
  for what is left (`gnux --list`, `gnux help`, an unknown name).

`gnux --list` prints every tool name; `make links`, `install.sh` and the Dockerfile use it to
create the per-tool links.

`cmd.RunTool(name, args, streams)` is the same dispatch as a function: it runs a tool by either
name and returns its exit status. The shell and the wasm build both go through it.

## The gnux shell (`core/shell`, `cmd/shell.go`)

`gnux shell [-eu] [-c STRING | SCRIPT] [ARG]...` - interactive when reading a terminal. It is a
gnux command, not a tool: `ShellSpec()` has no GNU name, is in `Commands()` but not `Tools()`,
is not reachable through `argv[0]` and is not in `--list`. Options stop at `SCRIPT`
(`NoInterspersed`); what follows is the script's arguments.

- **Language** (keep it small - it is a portable subset, not a bash clone): pipelines, `&&`,
  `||`, `;`, `!`, redirection `<` `>` `>>` `2>`, `NAME=value`, `$NAME` `${NAME}`
  `${NAME:-default}` `$?`, the arguments `$1`... `$#` `$0` `$@` `$*`, command substitution
  `$(...)`, `if`/`elif`/`else`, `for`, `while`, `case`, single/double quotes, backslash before a
  special character only (so `C:\tools` survives unquoted), globbing, `~`, `#` comments.
  Builtins: `cd`, `exit`, `help`, `set` (`-e`, `-u`, `-- ARG...`), `shift`, `break`,
  `continue`, `[` (the test tool). **Not in the language, on purpose:** functions, arithmetic,
  `until`, here-documents, backquotes, `( )` / `{ }` grouping, background jobs, and external
  programs (those go through `envx`). Adding any of them is a decision, not a patch.
- **Deliberate differences from sh:** a variable's value is never re-split or re-globbed, and an
  unquoted `$(...)` is split into lines, never words; `NAME=value cmd` sets the variable for the
  rest of the script, not just for `cmd`; nothing spans lines except an `if`, a loop, a `case`
  or a `$(...)` (not a quoted string, a trailing `|` / `&&` / `||`, or a backslash).
- The interpreter reaches tools only through its `Exec` func (`shell.ExecFunc`), so `core/shell`
  does not import `cmd`; `cmd.NewShell` wires `RunTool` in.
- `parse.go` is a recursive-descent parser over an on-demand lexer; `$(` is parsed in place by a
  second parser on the same source. A `*syntaxError` with `incomplete` set means "more lines
  could finish this", which is how `Run` knows to keep reading and what `shell.Incomplete`
  reports to a line-based host.
- Streams are passed down every call (`runList`, `runPipeline`, `runCommand`), never read from
  the `Shell`, so a redirected loop reaches every command inside it.
- Pipeline stages run as goroutines joined by `io.Pipe`. Expansion, assignments and opening
  redirections all happen *before* any stage starts (shell state is not goroutine-safe, and sh
  guarantees redirection targets exist first). A stage whose reader has gone sees `EPIPE`, which
  the tools already treat as a quiet stop - that is what ends `yesx | headx`. A pipeline may
  hold **one** compound command (the parser refuses two): its body runs on its stage's
  goroutine while the caller waits, so shell state still has one user at a time, and the state
  is saved before and restored after (it is a subshell). `$(...)` saves and restores the same
  state (`save` / `restore`) and puts the working directory back.
- `-e` exempts a status that is tested: before `&&` / `||`, after `!`, and in the condition of
  an `if` or `while` (`sh.testing`). `-u` makes an unset variable an expansion failure: the
  command is not run and the script stops with status 1.
- `Run` executes a whole script and restores the process working directory afterwards (`cd` is
  process-wide because tools resolve relative paths against it). `Eval` runs one command and
  keeps all state, for REPL-style hosts such as the playground, which also sets `LoopLimit`
  because nothing can interrupt a script in a browser tab.
- Exit status: last command's; 2 on a syntax error (which aborts a non-interactive script);
  127 for an unknown command; 1 for an unset variable under `-u`.

## The tool contract

### Engine - `core/<gnu>/`

```go
package rm

const Name = "rmx"                             // command name; prefix of every diagnostic

type Options struct{ … }                       // mirrors the GNU flags; zero value = no flags
func New(opts Options, s core.Streams) *Rm     // build the engine
func (r *Rm) Run(args []string) error          // run against the operands
```

- **`Options` zero value must be the tool's default behavior** - library callers write
  `rm.Options{Recursive: true}` and get everything else as plain `rm`. Name a field for the
  negative when the default is "on" (`NoPreserveRoot`, not `PreserveRoot`).
- Read and write only through the `core.Streams` given to `New` - never `os.Stdin`/`os.Stdout`.
  Buffer them with `core.NewReader` / `core.NewWriter` (one shared size, `core.BufferSize`), not
  `bufio.NewReader` / `bufio.NewWriter`, and flush the writer before returning
  (`core.FlushOutput`). Write a per-line number with `core.AppendInt`, not `fmt.Fprintf`.
  A filter calls `core.FlushIfIdle(out, in)` after each record it writes, so that it answers a
  slow producer line by line, and is listed in `streaming` and `lineFilters` in
  `cmd/robust_test.go`.
- Write diagnostics through `core.NewDiag(Name, s)` in GNU's exact wording
  (`rmx: cannot remove 'x': No such file or directory`); use `core.ErrText(err)` for the
  capitalized, platform-normalized syscall text and `core.Describe(info)` for file-type nouns.
- A failing operand does not stop the remaining operands.
- An engine that starts a goroutine defers `trap.Catch()` in it and calls `trap.Release()`
  once it has ended (`core.Trap`): a panic on a goroutine of its own would otherwise end the
  whole process, not just the tool.
- Expose injection points as exported fields for anything that needs a terminal, a clock or the
  real environment (`PromptFn`, `RemoveFn`, `Now`, `Getenv`, …).

### Error convention

| Returned error     | Meaning                                             | Exit status          |
|--------------------|-----------------------------------------------------|----------------------|
| `nil`              | success                                             | 0                    |
| `core.ErrFailed`   | ran, something failed; diagnostics already written  | 1                    |
| `*core.ExitError`  | a specific, silent status (`lsx` 2, `envx` 125–127) | `Code`               |
| `*core.UsageError` | command-line mistake; build with `core.Usagef(…)`   | `Spec.UsageExit` (1) |

`cmd.Main` prints a usage error (and any flag-parsing error from cobra) as
`<tool>: <message>` followed by `Try '<tool> --help' for more information.` Engines never print
that hint themselves.

### Adapter - `cmd/<gnu>x/`

```go
func Spec() tool.Spec                 // Meta, GNU name, Flags, Run (+ RawArgs/NoInterspersed/UsageExit)
func Meta() docmeta.Meta              // Name, Use, Short, Long, Permissions, UseCases, Examples
func Register(fs *pflag.FlagSet)      // define the flags
func buildOptions(fs) <gnu>.Options   // fold parsed values into Options (unexported, table-tested)
```

- `--help` and `--version` are added by `Spec.Command`; never define them. `--help` renders
  through `internal/tool/help.go` (`Spec.WriteHelp`), and `gnux help TOOL [OPTION]`
  (`cmd/help.go`) prints the full manual (`WriteManual`) or one option (`WriteFlagHelp`), all from
  `Meta`. Fill `Meta.SeeAlso`, and `Meta.ExitStatus` when the tool's statuses differ from GNU's
  0/1 default; both also appear in the generated reference. `--help` has no
  shorthand, so `-h` is free for tools that use it (`lsx -h`, `touchx -h`).
- `RawArgs: true` disables flag parsing for tools whose arguments are not GNU-style options
  (`echox`, `truex`, `falsex`, `yesx`); a lone `--help`/`--version` is still honored.
- `NoInterspersed: true` stops option parsing at the first operand (`envx`).
- Options are parsed by `tool.ParseFlags` as glibc's getopt parses them: unambiguous prefixes of
  long options, `is ambiguous; possibilities:`, getopt's wording for every mistake. A long name
  gnux invents for an option GNU spells with a letter only goes in `tool.OwnNames(fs, ...)` at
  the end of `Register`: it then answers only when written in full and never changes what a GNU
  abbreviation means.
- `truex` and `falsex` are the one exception to the two-package rule: they have no engine.
- cobra/pflag do not expose the order flags appeared in, so GNU's "last one wins" between
  conflicting flags is approximated by a fixed precedence - always toward the **most cautious**
  option - and documented in the tool's `Meta().Long`.

### `rmx` keeps per-flag packages

`cmd/rmx/<flag>/` (force, interactive, recursive, directory, verbose, onefilesystem,
preserveroot) each expose `Register`, `Apply(fs, *rm.Options)` and `Meta()`, and are listed as
`Topics` on the rmx spec, which gives each flag its own reference page. `interactive.Apply`
returns an error for an invalid `--interactive` value. Other tools register their flags in the
adapter package directly; add `Topics` to a spec only when per-flag pages are worth writing.

`rmx` interactive precedence: `force` is applied first (baseline `PromptNever`), then
`interactive`. Conflicts resolve toward the most cautious: `-i` (always) > `--interactive=WHEN`
> `-I` (once) > `-f` (never). `-f` still suppresses nonexistent-file errors regardless.

## Adding a tool

1. Create the engine in `core/<gnu>/` (`Name`, `Options`, `New`, `Run`) with a black-box
   `<gnu>_test.go`. Start from `core/pwd` (small) or `core/rm` (full-featured).
2. Create the adapter in `cmd/<gnu>x/` (`Spec`, `Meta`, `Register`, `buildOptions`, `run`) with a
   `buildOptions` table test. Start from `cmd/pwdx`.
3. Add `<gnu>x.Spec()` to `Tools()` in `cmd/registry.go` (kept alphabetical).
4. Run `make docs` and commit the regenerated `documentation/` and `web/content/docs/`.
5. Add the tool to the tables in `README.md` and the list above, give it a section in
   `docs/requirements/functional-requirements.md`, and add it to
   `documentation/guides/adoption.md`.

The step-by-step version, with a skeleton engine and adapter, is
`docs/workflows/adding-a-tool.md`.

`cmd/main_test.go` enforces the registry invariants (name = GNU name + `x`, `Use` starts with
the name, docs complete, `--help`/`--version` work), and `cmd/docs` checks that every tool gets
a page in every format - a new tool is covered by both as soon as it is registered.

## The rm engine (`core/rm`)

- `Run` returns `core.ErrFailed` if any removal failed. No operands is a usage error, unless `-f`.
- Refuses `.` / `..` under `-r` (without it they are ordinary directories); refuses recursive `/` unless `NoPreserveRoot`.
- `os.Lstat` throughout - symlinks are removed, never followed.
- Non-recursive directory → `Is a directory`; `-d` on a non-empty dir → `Directory not empty`.
- Recursive removal is **post-order**: with `-i`, prompts `descend into directory?`, then each
  child, then `remove directory?`.
- In the default mode (no `-f`/`-i`) on a terminal, a non-writable entry triggers
  `remove write-protected <type> '<path>'?`. Writability is `access(2)` on Unix and the owner
  write-bit on Windows (`core/fs_*.go`).
- **Concurrency:** non-interactive, non-verbose removal fans out one worker per CPU across the
  top-level arguments (`runConcurrent`); `core.Diag` is mutex-guarded. Prompting or `-v` runs
  sequentially - prompts share one stdin, and `-v` output must stay in argument order.
  Single-directory recursion uses `os.RemoveAll` whenever no per-entry decision is needed
  (`fastPathOK`): no prompting, no `-v`, no `--one-file-system`, no injected `RemoveFn`.
- Injection points: `PromptFn func(string) bool` and `RemoveFn func(string) error` (setting
  `RemoveFn` also disables the fast path so every entry flows through it).

## Documentation generation

`tools/gendocs` (also `make docs` and `go generate ./...`) builds the reference into `documentation/` via
`cmd/docs`, in **markdown, man, yaml and reST**. Each format directory is self-contained and is
rebuilt from scratch on every run:

- `<format>/index.*` - gnux itself; links to every tool page.
- `<format>/guide.*` - requirements, installation + usage guide (synthetic command; prose in
  `guide.go`). Keep its REQUIREMENTS paragraph and the README's Requirements section in step
  with `go.mod` and the build matrix.
- `<format>/<tool>/index.*` - one page per tool: description, permissions, use cases, the flag
  listing, and per-shell (`bash` / `powershell`) examples.
- `<format>/<tool>/flags/<flag>.*` - one page per `Topic`, for tools that declare them.
- `<format>/shell/index.*` - the gnux shell.

Four folders are hand-written and edited like code: `documentation/guides/` (usage), and in
`docs/`, `requirements/` (the SRS and the `BR-n` / `FR-n` / `NFR-n` registers, plus the `WF-n`
user workflows), `workflows/` (how processes run inside the code) and `rules/`. `docs/README.md`
maps them. The guides sit in `documentation/` beside the generated formats, which the generator
rebuilds one `documentation/<format>/` at a time, so it never touches them. Only the guides are
published on the website: `gendocs ... mdx` writes them as a section of
the same name (`HandwrittenPages` in `cmd/docs/handwritten.go`, folders listed in
`handwrittenFolders` in `tools/gendocs/main.go`). Requirements, workflows and rules are for
contributors and stay in the repository; do not add them to the website. A page's title is its
first `# ` heading and its description its first paragraph, so every file starts with one
`# Title` and a one-paragraph summary and uses standard markdown only. Relative links to `.md`
files in a published folder become site links; any other repository link becomes a GitHub link.

When a change adds, removes or alters behavior a requirement describes, update the register in
the same change; cite the IDs in commits and PRs where it helps. When it changes how a process
runs (dispatch, the shell, the generator, CI), update the matching `docs/workflows/` page; when
it changes a usage example the guides show, update the guide.

`gendocs <dir> mdx` (run by `make docs` / `go generate` with `web/content/docs`) writes the
website's content instead: `index.mdx`, `guide.mdx`, `library.mdx`, `shell.mdx`,
`tools/<tool>/index.mdx`, one `tools/<tool>/<flag>.mdx` per option (a `Topic`'s prose where there
is one, otherwise the flag's usage, plus the tool's examples that pass the flag; `mdx_flags.go`)
and the `meta.json` files that order the sidebar (`mdx.go`). The sidebar has no collapsible
sections: the guides come first, listed page by page under a `---Guides---` heading, then the
tools under `---Tools---`; `fumadocs.css` sets those headings small, in capitals and the lit
color. Pages carry
`title`/`description` frontmatter and plain markdown only; `mdxEscape` escapes `{`, `}` and `<`
in prose so MDX does not read them as JSX. The guide is defined once as titled sections
(`guideSections`) and rendered as prose for the cobra formats and as headings for MDX.

Everything is driven by `cmd.Tools()` and each spec's `docmeta.Meta` - there is no second list to
update. Tool and flag markdown is hand-rendered (`markdown.go`) to get multi-section,
multi-language code blocks; man/yaml/reST reuse `cobra/doc` from a synthetic command that folds
permissions and use cases into its description. In markdown only, `wrapCode` wraps flag and
command references in backticks (ambiguous English tool names like `true`, `head`, `cat` are
left alone; the x-names always wrap). Output is **deterministic** - the cobra auto-gen tag is
disabled and the man date is pinned (`docs.ManDate`) - so committed docs don't churn.
`cobra/doc` pulls in `go-md2man`, but only `tools/gendocs` imports `cmd/docs`, so it never links
into the `gnux` binary.

## Rules

The cross-cutting rules, one file each in [`docs/rules/`](docs/rules/README.md), with the
reasoning, examples and the check that enforces each. The statements are repeated here so that
nothing depends on following a link.

### Backend

- **GNU is the specification, and every difference is documented** - match GNU's flags, output and statuses, and write any deliberate difference into the tool's `Meta().Long` or `Permissions`. ([file](docs/rules/gnu-is-the-specification-and-differences-are-documented.md))
- **The zero value is the safe default** - `Options{}` behaves like the tool with no flags and keeps every safeguard on, so a default-on field is named for its negative (`NoPreserveRoot`). ([file](docs/rules/the-zero-value-is-the-safe-default.md))
- **Engines know nothing of the command line** - behavior lives in `core/<gnu>/` behind `Options`, flags live in `cmd/<gnu>x/`, and nothing under `core/` imports cobra or pflag. ([file](docs/rules/engines-know-nothing-of-the-command-line.md))
- **An engine reads and writes only its streams** - use the `core.Streams` given to `New`, never `os.Stdin`, `os.Stdout` or `os.Stderr`; only `main.go` passes `core.Stdio()`. ([file](docs/rules/an-engine-reads-and-writes-only-its-streams.md))
- **There is one list of tools** - `Tools()` in `cmd/registry.go` feeds the binary, `--list`, the docs, the wasm build and the playground, and nothing keeps a second copy. ([file](docs/rules/there-is-one-list-of-tools.md))
- **Diagnostics are written in GNU's words** - write through `core.Diag` as `<tool>: <message>`, quote the operand, and use `core.ErrText` so the text is identical on every platform. ([file](docs/rules/diagnostics-are-written-in-gnus-words.md))
- **A failing operand does not stop the rest** - report the failure, carry on with the remaining operands, and return `diag.Result()` at the end. ([file](docs/rules/a-failing-operand-does-not-stop-the-rest.md))
- **Conflicts resolve toward the most cautious option** - when GNU's last-one-wins cannot be reproduced, `buildOptions` picks the cautious flag in any order, tests both orders and documents it. ([file](docs/rules/conflicts-resolve-toward-the-most-cautious-option.md))
- **A failure is never reported as success** - every failure gets a diagnostic and a nonzero status exactly once, and a tool refuses rather than silently doing something else. ([file](docs/rules/a-failure-is-never-reported-as-success.md))
- **Platform code lives behind build tags, with a fallback that compiles everywhere** - use `unix` / `!unix` files that also build for js/wasm, with no `runtime.GOOS` branches and no cgo. ([file](docs/rules/platform-code-lives-behind-build-tags.md))
- **The shell runs no external program implicitly** - `gnux shell` reaches only builtins and registered tools through `Exec`, anything else is `command not found` (127), and external programs go through `envx`. ([file](docs/rules/the-shell-runs-no-external-program-implicitly.md))
- **The reference documents itself, and it is checked** - `docs/` and `web/content/docs/` are generated deterministically from the registry and `Meta()`, never edited by hand, and the Docs workflow rejects stale copies. ([file](docs/rules/the-reference-documents-itself-and-it-is-checked.md))
- **Tests drive engines, not terminals** - build engines on `t.TempDir()`, buffers and injected funcs (`PromptFn`, `RemoveFn`, `Getenv`), with the standard library only and `-race` always on. ([file](docs/rules/tests-drive-engines-not-terminals.md))

### Frontend

- **No em dashes, and US spelling** - everywhere in the repository, using the punctuation that names the relationship instead. ([file](docs/rules/no-em-dashes.md))
- **A public page describes itself** - every page builds its metadata with `createMetadata(siteConfig, {...})`, never a `Metadata` literal or a hardcoded origin. ([file](docs/rules/a-public-page-describes-itself.md))
- **The markup says what the thing is** - semantic elements and landmarks before ARIA, an accessible name on every control, one `h1` with no skipped levels, and no meaning by color alone. ([file](docs/rules/the-markup-says-what-the-thing-is.md))
- **The shortest path, and only one of them** - one obvious route to each thing in the fewest steps, one primary action per view, no duplicate controls, labels that say what will happen and a reason beside anything disabled. ([file](docs/rules/the-shortest-path-and-only-one-of-them.md))
- **Build the UI from atoms upward** - compose from `src/components/ui` first; pages compose organisms and do not style from scratch. ([file](docs/rules/build-the-ui-from-atoms-upward.md))
- **A dark step mirrors its light one** - prefer the `fd-*` tokens, and otherwise pair each light step with its mirror (`text-red-700 dark:text-red-300`). ([file](docs/rules/a-dark-step-mirrors-its-light-one.md))
- **Type sizes come from the scale** - use `text-xs` through `text-3xl`, never an arbitrary `text-[13px]`. ([file](docs/rules/type-sizes-come-from-the-scale.md))
- **An icon-only button takes an icon size** - use `size="icon"` or `"icon-sm"`, an `aria-label`, and an `aria-hidden` icon. ([file](docs/rules/an-icon-only-button-takes-an-icon-size.md))
- **Icons come from the installed set** - `lucide-react` and nothing else. ([file](docs/rules/icons-come-from-the-four-installed-sets.md))
- **State a reader can arrive at lives in the URL** - shareable state goes through `nuqs`, like the playground's `?cmd=`, and is never run on arrival. ([file](docs/rules/state-a-reader-can-arrive-at-lives-in-the-url.md))
- **Data from outside is unknown until checked** - the wasm API and URL parameters are narrowed with `lib/guards.ts`, never cast. ([file](docs/rules/data-from-outside-is-unknown-until-checked.md))
- **An array fallback is a type check, never a nullish default** - write `Array.isArray(v) ? v : []` or `arrayOf(v, guard)`, never `v ?? []`. ([file](docs/rules/an-array-fallback-is-a-type-check-never.md))
- **Copying goes through useCopyToClipboard** - only the hook touches `navigator.clipboard`, and the control says whether the copy worked. ([file](docs/rules/copying-goes-through-usecopytoclipboard.md))
- **The playground never touches the visitor's files** - the real gnux runs on an in-memory filesystem the page supplies, with no disk access, uploads or external programs. ([file](docs/rules/the-playground-never-touches-the-visitors-files.md))
- **A memfs error carries a code Go knows** - every error is an `FsError` whose code is in Go's `errnoByCode` table, and the real wasm in `pnpm test` is the judge. ([file](docs/rules/a-memfs-error-carries-a-code-go-knows.md))

### Working on the repository

- **Agents hand over the git commands, and the maintainer runs them** - never `git commit`, `git push`, `git tag`, a force-push, or `gh release create` / `delete`, even when asked to release; prepare and check the change, then end with the exact commands and the commit or tag message to paste. ([file](docs/rules/agents-hand-over-git-commands.md))

## Code Style

- Doc comments on exported symbols; comment the *why*, not the obvious.
- US spelling (misspell runs with `locale: US`).
- Wrap errors with `fmt.Errorf("…: %w", err)`.
- Filenames are snake_case (`fs_unix.go`).
- Platform variance goes in build-tagged files (`*_unix.go` / `*_windows.go`), never runtime
  `runtime.GOOS` branching.
- Everything under `core/` is public API: keep it free of cobra/pflag, and don't break exported
  names casually.

## Linting

`.golangci.yml` (golangci-lint **v2** schema): default set + `misspell`, `unconvert`, `revive`,
`gocritic`. Complexity linters (`funlen`, `cyclop`, `maintidx`, `gocognit`) are disabled - the
engines' dispatch uses intentional `switch` chains. `errcheck` ignores `*.Close()`, `os.Remove`
cleanup, and best-effort `fmt.Fprint*` diagnostic writes. `_test.go` files are exempt from
`gocritic`/`revive`.

## Testing

- Race detector always on: `go test -race ./...`.
- `core/<gnu>/<gnu>_test.go` - the primary suite for each tool (black-box `<gnu>_test`): drives
  the engine against `t.TempDir()` and in-memory streams, asserting on exact stdout/stderr.
  Inject prompts, clocks and the environment rather than driving a real terminal.
- `core/rm/example_test.go` - a runnable `Example` documenting library use.
- `cmd/<gnu>x/<gnu>x_test.go` - `buildOptions`: every flag mapping and precedence rule.
- `core/shell/shell_test.go` - the interpreter against a fake command set: quoting, expansion,
  globbing, connectors, pipelines, redirection, `cd`, `-e`, syntax errors, `Eval`.
  `core/shell/control_test.go` - `if`, `for`, `while`, `case`, `$(...)`, script arguments, `-u`,
  commands that span lines.
- `cmd/shell_test.go` - `gnux shell` end to end with the real tools, `RunTool`, `NewShell`.
- `cmd/main_test.go` - registry invariants, argv[0] and subcommand dispatch (by tool name and by
  GNU name), exit statuses and usage-error formatting.
- `cmd/robust_test.go`, `cmd/robust_unix_test.go` - the real tools held to their limits:
  streaming tools in a fixed amount of memory, no goroutine or open file left behind, a filter
  answering each line while its input stays open.
- `cmd/bench_test.go` - the benchmark suite (`make bench`);
  `core/{sed,grep,find,xargs,printf,seq,tr,cut}/fuzz_test.go` and `core/shell/fuzz_test.go` - the
  fuzz targets (`make fuzz`). All of it is described in
  `docs/workflows/benchmarking-and-robustness.md`.
- `cmd/conformance_test.go` (build tag `conformance`) - every line of
  `cmd/testdata/conformance/*.txt` run through GNU's tools and through gnux under the GNU names,
  output, errors and status compared (`make conformance`, `docs/workflows/gnu-conformance.md`).
  A behavior change that GNU shares gets a case there.
- `internal/tool/getopt_test.go` - options parsed as getopt does: abbreviations, `OwnNames`,
  glibc's wording for option errors.
- `cmd/docs/docs_test.go` - `wrapCode`/`fenceLang`, an end-to-end `GenerateAll` (a page per tool
  per format, markdown links resolve, man stays plain, output is deterministic) and
  `GenerateMDX` (a page per tool, frontmatter present, no unescaped MDX characters).
- `web/` - `pnpm test` drives the real `gnux.wasm` in Node against the playground's in-memory
  filesystem.

## CI

| File        | What it does                                                                            |
|-------------|-----------------------------------------------------------------------------------------|
| `test.yml`  | `go vet` + `go test -race` on ubuntu/macos/windows, a multi-call smoke test, wasm build, 30s fuzzing of every parser (`make fuzz`), and the GNU conformance harness in a Debian container |
| `build.yml` | cross-compiles `gnux` for every release platform and uploads the binaries               |
| `lint.yml`  | `golangci-lint` (pinned v2 release)                                                     |
| `docs.yml`  | regenerates `documentation/` and `web/content/docs/`; fails if the committed copies are stale |
| `web.yml`   | builds the wasm, then lints, type-checks, tests and builds the website                  |
| `bench.yml` | on a pull request: benchmarks the base commit and the change in one job and fails on a regression (`tools/benchcheck`) |
| `action.yml` | runs the setup-gnux action (`action.yml` at the root) on linux/macos/windows runners and uses the tools |
| `release.yml` | on a `vX.Y.Z` tag: re-runs vet/test/docs checks, requires the tag's `CHANGELOG.md` section, then GoReleaser (`.goreleaser.yaml`) builds per-platform archives with completions and `.deb`/`.rpm`/`.apk` packages, publishes a GitHub release with an SBOM per archive, a cosign-signed `checksums.txt`, a provenance attestation and that section as its notes, publishes to each of Homebrew, Scoop, winget, the Snap Store and the AUR whose secret is set, moves the `vX` major tag, and (job `image`) pushes the Alpine and `scratch` Docker images to GHCR |

An agent prepares a release (changelog section, `make docs`, `make ci`, `make release/check`) and hands the maintainer the commit, push and tag commands; it never runs them ([rule](docs/rules/agents-hand-over-git-commands.md)).

Every notable change goes into `CHANGELOG.md` under `## [Unreleased]` in the same change
(Keep a Changelog: Added, Changed, Fixed, Removed). It is published as `/docs/changelog`. To
release, turn that section into `## [X.Y.Z] - YYYY-MM-DD`, open a new empty Unreleased above it,
update the link definitions, commit, then `git tag -a vX.Y.Z && git push origin vX.Y.Z`; a tag
with a hyphen is a prerelease. `docs/workflows/publishing-a-release.md` is the full procedure,
with the one-time setup of each package manager. The release's notes are that section (`go run
./tools/changelog vX.Y.Z` prints them), and a release tag without one fails before it builds. Every workflow is `permissions: contents: read`; only the release's `release` job gets
`contents: write` (plus `id-token`/`attestations: write` for the attestation and the keyless
signature), and its `image` job `packages: write`. Without link flags, `common.Version` falls back to the build-info module
version, so `go install ...@vX.Y.Z` reports the tag.

All read the Go toolchain from `go.mod` via `go-version-file: go.mod` - keep it there rather
than hardcoding a version. Each workflow states the requirement it enforces in a comment at the
top; when a requirement changes, update the workflow, the README and the guide together.

## Important Files

- `main.go` - thin entry point; `//go:generate` hook.
- `cmd/main.go` - `Main`: tool selection by `argv[0]`, error → exit status.
- `cmd/registry.go` - `Tools()`: the list everything else is built from.
- `cmd/root.go` - `NewRoot`: the `gnux` command and its subcommands.
- `internal/tool/tool.go` - `Spec`, `Topic`, `Spec.Command`.
- `core/core.go` - `Streams`, `Diag`, `Prompter`, the error types.
- `core/rm/`, `cmd/rmx/` - the full-featured reference tool; `core/pwd/`, `cmd/pwdx/` - the
  minimal one.
- `core/shell/`, `cmd/shell.go` - the gnux shell.
- `tools/wasm/main.go` - the playground's WebAssembly entry point and its JS API.
- `cmd/docs/` - reference generator (`docs.go`, `markdown.go`, `guide.go`, `mdx.go`).
- `.golangci.yml` - linter configuration (v2 schema).
