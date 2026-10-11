# gnux

[![Test](https://github.com/braswelljr/gnux/actions/workflows/test.yml/badge.svg)](https://github.com/braswelljr/gnux/actions/workflows/test.yml)
[![Build](https://github.com/braswelljr/gnux/actions/workflows/build.yml/badge.svg)](https://github.com/braswelljr/gnux/actions/workflows/build.yml)
[![Lint](https://github.com/braswelljr/gnux/actions/workflows/lint.yml/badge.svg)](https://github.com/braswelljr/gnux/actions/workflows/lint.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/braswelljr/gnux.svg)](https://pkg.go.dev/github.com/braswelljr/gnux)

The GNU core utilities' interface, everywhere: one Go binary, a library and a portable shell.
**gnux** is a portable, GNU-compatible implementation of the core utilities that brings their
command-line interface (the same flags, semantics, diagnostics and exit statuses) to Linux,
macOS, Windows, containers, the browser and Go programs, without needing GNU coreutils
installed. Every tool is named after its GNU counterpart with an `x` appended (`rm` is `rmx`,
`cp` is `cpx`, `ls` is `lsx`) and behaves like it.

> gnux is an independent project. It is not affiliated with or endorsed by the GNU Project or
> the Free Software Foundation; it does not replace GNU coreutils, it makes their interface
> available wherever you work.

```bash
rmx -rf build/          # each tool is a command of its own…
gnux rmx -rf build/     # …and a subcommand of the one binary
```

```go
// …and an importable engine…
err := rm.New(rm.Options{Recursive: true, Force: true}, core.Stdio()).Run([]string{"build"})
```

```bash
# …and a command in a script that runs the same on Linux, macOS and Windows.
gnux shell -c 'mkdirx -p dist && cpx -r assets dist/ && lsx -l dist'
```

Try it in the browser: the [playground](web/) runs the real tools, compiled to WebAssembly,
against an in-memory filesystem.

## Tools

| Tool        | GNU        | What it does                                       |
|-------------|------------|----------------------------------------------------|
| `basenamex` | `basename` | Strip directory and suffix from file names         |
| `catx`      | `cat`      | Concatenate files and print them                   |
| `cpx`       | `cp`       | Copy files and directories                         |
| `dirnamex`  | `dirname`  | Strip the last component from file names           |
| `echox`     | `echo`     | Print a line of text                               |
| `envx`      | `env`      | Run a program in a modified environment            |
| `falsex`    | `false`    | Do nothing, unsuccessfully                         |
| `headx`     | `head`     | Print the first part of files                      |
| `lnx`       | `ln`       | Make hard and symbolic links                       |
| `lsx`       | `ls`       | List directory contents                            |
| `mkdirx`    | `mkdir`    | Create directories                                 |
| `mvx`       | `mv`       | Move (rename) files                                |
| `pwdx`      | `pwd`      | Print the current working directory                |
| `realpathx` | `realpath` | Print the resolved absolute path                   |
| `rmx`       | `rm`       | Remove files and directories                       |
| `rmdirx`    | `rmdir`    | Remove empty directories                           |
| `sleepx`    | `sleep`    | Pause for a given time                             |
| `tailx`     | `tail`     | Print the last part of files, optionally following |
| `touchx`    | `touch`    | Change file timestamps, creating files as needed   |
| `truex`     | `true`     | Do nothing, successfully                           |
| `wcx`       | `wc`       | Count lines, words and bytes                       |
| `yesx`      | `yes`      | Print a string repeatedly                          |

`gnux --list` prints the tool names; `gnux --help` lists them with a summary. The rest of
coreutils is on the [roadmap](#roadmap).

## Requirements

**To run gnux**

- Linux or macOS on amd64 or arm64, or Windows on amd64 (the platforms CI builds). The binary
  is statically linked (no cgo) and has no runtime dependencies - nothing else needs to be
  installed.
- For the per-tool commands (`rmx`, `cpx`, …): a filesystem that supports symbolic links, and
  the install directory on your `PATH`. On Windows, creating symlinks needs Developer Mode or
  an elevated shell; copying `gnux.exe` to `rmx.exe` works just as well.

**To install or build from source**

- [Go](https://go.dev/dl/) 1.25 or newer (the version is pinned in `go.mod`).
- `git`, for `go install` and for the version string stamped into source builds.
- Optional: `make` for the Makefile targets, `bash` for `install.sh`, Docker for the image.

**To work on the website** (`web/`: documentation and playground)

- Node.js 22.18 or newer and [pnpm](https://pnpm.io/), plus Go to compile the tools to WebAssembly.

**To use gnux as a library**

- A Go module on Go 1.25 or newer. The engines depend only on the standard library and
  `github.com/mattn/go-isatty`; cobra is compiled only into the command-line packages under
  `cmd/`.

**To contribute**

- Everything above, plus [golangci-lint](https://golangci-lint.run/) v2 for `make lint` and
  `make format`, and [lefthook](https://lefthook.dev/) for the Git hooks (`make hooks` installs
  them: format and lint on commit, a docs freshness check on push). `make ci` must pass, which
  includes regenerating `documentation/`.

## Installation

### Prebuilt binaries

Every tagged version is published as a [GitHub release](https://github.com/braswelljr/gnux/releases)
with an archive per platform (`gnux-<version>-<os>-<arch>.tar.gz`, `.zip` for Windows) and a
`checksums.txt`:

```bash
v=v0.1.0
curl -LO "https://github.com/braswelljr/gnux/releases/download/$v/gnux-$v-linux-amd64.tar.gz"
curl -LO "https://github.com/braswelljr/gnux/releases/download/$v/checksums.txt"
sha256sum --check --ignore-missing checksums.txt
tar -xzf "gnux-$v-linux-amd64.tar.gz" && sudo mv "gnux-$v-linux-amd64/gnux" /usr/local/bin/
```

### With `go install`

```bash
go install github.com/braswelljr/gnux@latest
```

This installs the `gnux` binary. To also get every tool as its own command, link the tool
names to it (gnux looks at the name it was started under):

```bash
bin="$(dirname "$(command -v gnux)")"
for t in $(gnux --list); do ln -sf gnux "$bin/$t"; done
```

### Install script

```bash
./install.sh                 # latest via `go install`, plus the per-tool links
./install.sh --from-source   # build from the current checkout
./install.sh --no-links      # the gnux binary only
```

### From source

```bash
git clone https://github.com/braswelljr/gnux.git
cd gnux
make install/links   # install gnux and link every tool next to it
# or: make links && ./bin/lsx -la
```

### Docker

```bash
docker build -t gnux .
docker run --rm -v "$(pwd):/work" gnux lsx -la
docker run --rm -v "$(pwd):/work" --entrypoint rmx gnux -rv build/
# or via compose:
docker compose run --rm gnux --version
```

## Usage

### As commands

Run a tool by its own name once linked, or through `gnux`:

```bash
lsx -la                           # list everything, long format
cpx -r src/ backup/               # copy a tree
mvx -i draft.md final.md          # rename, asking before overwriting
rmx -rf /tmp/cache                # unattended recursive delete
mkdirx -p a/b/c                   # create a path of directories
tailx -f app.log                  # follow a log
wcx -l *.go                       # count lines
envx -i PATH=/usr/bin mytool      # run with a clean environment

gnux lsx -la                      # the same tools as subcommands
```

Both forms also answer to the GNU name - `gnux rm …` works, and a link called `rm` behaves as
`rmx` - so scripts written for the GNU tools run unchanged where GNU coreutils is not
installed (Windows, minimal containers).

Every tool accepts `--help` and `--version`. Options follow GNU conventions: short flags
combine (`-rvf`), long flags take `--name=value`, and `--` ends option parsing.

Tools exit `0` on success and `1` when an operation fails, with a diagnostic on standard error
prefixed by the tool name (`rmx: cannot remove 'x': No such file or directory`). A
command-line mistake adds a pointer to `--help`. Tools whose status carries more meaning keep
it: `lsx` exits `2` on serious trouble and `envx` uses `125`–`127`.

### As a portable shell

`gnux shell` is a small interpreter whose commands are the gnux tools, run inside the gnux
process. Nothing in it depends on the system's own shell, so one script runs unchanged on every
platform - the only thing a machine needs is the gnux binary.

```bash
gnux shell                       # interactive session
gnux shell -e build.gnux         # run a script, stopping at the first failure
gnux shell -c 'catx *.log | wcx -l'
```

```bash
# build.gnux
OUT=dist
rm -rf $OUT && mkdir -p $OUT/logs        # tools answer to their GNU names too
cp -r assets $OUT/
ls -1 $OUT | wc -l > $OUT/count.txt
cat $OUT/count.txt || echo "nothing built"
```

It understands a deliberately small subset of POSIX sh: pipelines, `&&` / `||` / `;`,
redirection (`<`, `>`, `>>`), `NAME=value` variables with `$NAME`, `${NAME}` and `$?`, single
and double quotes, globbing, `~` and `#` comments, with the builtins `cd`, `exit` and `help`.
There are no loops or functions, and external programs are reached through `envx PROGRAM …`.

### As a library

Each tool's engine is an importable package under [`core/`](core/). An engine takes the
options the flags would set plus the streams to read and write, and returns an error:

```bash
go get github.com/braswelljr/gnux
```

```go
package main

import (
	"errors"
	"log"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/rm"
)

func main() {
	// rmx -rf build dist
	engine := rm.New(rm.Options{Recursive: true, Force: true}, core.Stdio())

	err := engine.Run([]string{"build", "dist"})
	if errors.Is(err, core.ErrFailed) {
		// Some paths could not be removed; the diagnostics already went to stderr.
		log.Fatal("cleanup incomplete")
	}
}
```

- `core.Stdio()` wires an engine to the process's streams; `core.Discard()` silences it; or
  build a `core.Streams` from any `io.Reader`/`io.Writer` to capture output in a buffer.
- Errors follow one convention: `core.ErrFailed` (ran, but something failed and was reported),
  `*core.UsageError` (bad arguments) and `*core.ExitError` (a specific exit status).
- Engines expose injection points (`PromptFn`, `RemoveFn`, clocks, …) so the behavior that
  normally needs a terminal or a real filesystem can be driven from code.
- To run a tool by name, call `cmd.RunTool("rmx", args, streams)`; to run a whole portable
  script in-process, use `cmd.NewShell(opts, streams).RunString(script)`.
- To embed the entire command line, call `cmd.Main(argv, streams)`: it returns the exit status
  rather than exiting.

API documentation: [pkg.go.dev/github.com/braswelljr/gnux](https://pkg.go.dev/github.com/braswelljr/gnux).

## Workflows

**Everyday use.** Install once, link the tool names, and use `rmx`, `cpx`, `lsx` and the rest
exactly as you would their GNU counterparts - including on Windows, where GNU coreutils is not
usually installed.

**Cross-platform scripting.** Put the steps of a build, clean or packaging job in a script and
run it with `gnux shell -e`. The same file serves a developer laptop and every CI runner.

**Embedding.** Import the engine you need from `core/`, hand it your own streams, and check the
returned error against `core.ErrFailed`.

**Contributing.** Add an engine under `core/` and its adapter under `cmd/`, register it in
`cmd/registry.go`, then run `make ci` - vet, lint, tests, and regeneration of every generated
doc. Commit the regenerated files with your change.

**Continuous integration.** Every push and pull request runs these GitHub workflows:

| Workflow | Checks                                                                                                   |
|----------|----------------------------------------------------------------------------------------------------------|
| Test     | `go vet` and `go test -race` on Linux, macOS and Windows; a multi-call smoke test; the WebAssembly build |
| Build    | cross-compiles `gnux` for Linux, macOS and Windows and uploads the binaries                              |
| Lint     | `golangci-lint`                                                                                          |
| Docs     | regenerates `documentation/` and `web/content/docs/` and fails if the committed copies are stale         |
| Web      | builds the WebAssembly binary, then lints, type-checks, tests and builds the website                     |
| Release  | on a `v*` tag: re-runs the checks, builds an archive per platform, publishes them with checksums and the tag's [`CHANGELOG.md`](CHANGELOG.md) section as notes |

## Documentation

The website in [`web/`](web/) - Fumadocs documentation plus the playground - is the friendliest
way in: `make web/dev` serves it locally.

A full reference is also generated into [`documentation/`](documentation/markdown/index.md) in markdown, man, YAML and
reST, grouped by format so each directory is self-contained (`documentation/markdown/` holds all
markdown, etc.). The same content prints in the terminal with `gnux help TOOL`:

- `documentation/<format>/index.*` - gnux itself (entry page, links to every tool)
- `documentation/<format>/guide.*` - installation and usage guide
- `documentation/<format>/<tool>/index.*` - one page per tool: description, permissions, use cases,
  flags, and examples for bash and PowerShell
- `documentation/<format>/<tool>/flags/<flag>.*` - one page per flag, for tools that document them
  individually (`rmx`)
- `documentation/<format>/shell/index.*` - the gnux shell

Hand-written documentation lives in [`docs/`](docs/README.md). The guides are also published on the website; the
requirements, workflows and rules are for contributors and live here only:

| Directory                                         | What it holds                                                                    |
|---------------------------------------------------|----------------------------------------------------------------------------------|
| [`documentation/guides/`](documentation/guides/README.md) | Getting started, a cookbook of recipes, portable scripting, Windows and PowerShell, CI and containers, the Go library, adoption next to GNU coreutils, troubleshooting |
| [`docs/requirements/`](docs/requirements/)        | The SRS and the business, functional and non-functional requirement registers   |
| [`docs/workflows/`](docs/workflows/README.md)     | How each process runs inside the code: dispatch, engines, the shell, docs, CI and releases |
| [`docs/rules/`](docs/rules/README.md)             | The rules the code follows, each with its reasoning and how it is checked        |

[`docs/README.md`](docs/README.md) is the map of all of it.

Start at [`documentation/markdown/index.md`](documentation/markdown/index.md). The website's pages
(`web/content/docs/`) are generated from the same source. Regenerate everything with `make docs`
(or `go generate ./...`).

## Development

```bash
make build        # build ./bin/gnux (version injected via ldflags)
make links        # build, then link every tool name in ./bin
make test         # go test -race ./...
make cover        # tests + coverage.out summary
make lint         # golangci-lint
make format       # format everything: Go (gofmt -s + goimports) and the website (Prettier)
make hooks        # install the lefthook Git hooks (lefthook.yml)
make docs         # regenerate documentation/ and web/content/docs/
make ci           # vet + lint + test + docs freshness check
make web/dev      # build the wasm + docs content, then serve the website locally
make web/test     # test the playground's filesystem against the real wasm
make web/build    # production build of the website
make help         # list all targets
```

Each tool is two packages: an engine in [`core/<tool>/`](core/) holding all the behavior, and a
thin adapter in [`cmd/<tool>x/`](cmd/) that declares the flags and the documentation. One
registry ([`cmd/registry.go`](cmd/registry.go)) feeds the root command, the name-based dispatch
and the doc generator. See [`AGENTS.md`](AGENTS.md) for the architecture, conventions, and how
to add a tool.

## Roadmap

Not yet implemented, roughly in priority order:

- **File attributes:** `chmod`, `chown`, `chgrp`, `stat`, `du`, `df`, `mktemp`, `install`,
  `truncate`, `readlink`, `link`, `unlink`
- **Text:** `sort`, `uniq`, `cut`, `tr`, `tee`, `paste`, `join`, `nl`, `fold`, `fmt`, `expand`,
  `unexpand`, `tac`, `split`, `csplit`, `comm`, `shuf`, `seq`, `printf`, `od`
- **Checksums and encoding:** `md5sum`, `sha1sum`, `sha256sum`, `sha512sum`, `b2sum`, `cksum`,
  `base32`, `base64`
- **System:** `date`, `uname`, `whoami`, `id`, `hostname`, `nproc`, `printenv`, `tty`, `who`,
  `uptime`, `timeout`, `nice`, `nohup`, `kill`, `test`, `expr`, `dd`

## License

See [LICENSE](LICENSE).
