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

| Tool         | GNU         | What it does |
|--------------|-------------|--------------|
| `b2sumx`     | `b2sum`     | Print or check BLAKE2b (512-bit) checksums |
| `base32x`    | `base32`    | Encode or decode data in base32 |
| `base64x`    | `base64`    | Encode or decode data in base64 |
| `basenamex`  | `basename`  | Strip the directory and suffix from file names |
| `catx`       | `cat`       | Concatenate files and print them on standard output |
| `chgrpx`     | `chgrp`     | Change group ownership |
| `chmodx`     | `chmod`     | Change file mode bits |
| `chownx`     | `chown`     | Change file owner and group |
| `cksumx`     | `cksum`     | Print or check checksums and byte counts |
| `commx`      | `comm`      | Compare two sorted files line by line |
| `cpx`        | `cp`        | Copy files and directories |
| `csplitx`    | `csplit`    | Split a file into sections determined by context lines |
| `cutx`       | `cut`       | Remove sections from each line of files |
| `datex`      | `date`      | Print the date and time |
| `ddx`        | `dd`        | Convert and copy a file, block by block |
| `dfx`        | `df`        | Report file system space usage |
| `dirnamex`   | `dirname`   | Strip the last component from file names |
| `dux`        | `du`        | Estimate file space usage |
| `echox`      | `echo`      | Display a line of text |
| `envx`       | `env`       | Run a program in a modified environment, or print the environment |
| `expandx`    | `expand`    | Convert tabs to spaces |
| `exprx`      | `expr`      | Evaluate an expression and print its value |
| `falsex`     | `false`     | Do nothing, unsuccessfully |
| `findx`      | `find`      | Search for files in a directory hierarchy |
| `fmtx`       | `fmt`       | Reformat each paragraph of text |
| `foldx`      | `fold`      | Wrap each input line to fit in a specified width |
| `grepx`      | `grep`      | Print the lines that match patterns |
| `headx`      | `head`      | Print the first part of each file |
| `hostnamex`  | `hostname`  | Print the name of the current host system |
| `idx`        | `id`        | Print real and effective user and group IDs |
| `installx`   | `install`   | Copy files into place and set their mode, owner and group |
| `joinx`      | `join`      | Join lines of two files on a common field |
| `killx`      | `kill`      | Send a signal to processes, or list signals |
| `linkx`      | `link`      | Call the link function to create a hard link to a file |
| `lnx`        | `ln`        | Make hard or symbolic links between files |
| `lsx`        | `ls`        | List information about files and directory contents |
| `md5sumx`    | `md5sum`    | Print or check MD5 (128-bit) checksums |
| `mkdirx`     | `mkdir`     | Create the DIRECTORY(ies), if they do not already exist |
| `mktempx`    | `mktemp`    | Create a temporary file or directory, safely, and print its name |
| `mvx`        | `mv`        | Move (rename) files and directories |
| `nicex`      | `nice`      | Run a program with modified scheduling priority |
| `nlx`        | `nl`        | Number lines of files |
| `nohupx`     | `nohup`     | Run a command immune to hangups, with output to a non-terminal |
| `nprocx`     | `nproc`     | Print the number of processing units available |
| `odx`        | `od`        | Dump files in octal and other formats |
| `pastex`     | `paste`     | Merge lines of files |
| `printenvx`  | `printenv`  | Print all or part of the environment |
| `printfx`    | `printf`    | Format and print data |
| `pwdx`       | `pwd`       | Print the full name of the current working directory |
| `readlinkx`  | `readlink`  | Print resolved symbolic links or canonical file names |
| `realpathx`  | `realpath`  | Print the resolved absolute file name |
| `rmdirx`     | `rmdir`     | Remove the DIRECTORY(ies), if they are empty |
| `rmx`        | `rm`        | Remove (unlink) files and directories |
| `sedx`       | `sed`       | Filter and transform text with a stream editor |
| `seqx`       | `seq`       | Print a sequence of numbers |
| `sha1sumx`   | `sha1sum`   | Print or check SHA-1 (160-bit) checksums |
| `sha224sumx` | `sha224sum` | Print or check SHA-224 (224-bit) checksums |
| `sha256sumx` | `sha256sum` | Print or check SHA-256 (256-bit) checksums |
| `sha384sumx` | `sha384sum` | Print or check SHA-384 (384-bit) checksums |
| `sha512sumx` | `sha512sum` | Print or check SHA-512 (512-bit) checksums |
| `shufx`      | `shuf`      | Generate random permutations |
| `sleepx`     | `sleep`     | Delay for a specified amount of time |
| `sortx`      | `sort`      | Sort lines of text files |
| `splitx`     | `split`     | Split a file into pieces |
| `statx`      | `stat`      | Display file or file system status |
| `tacx`       | `tac`       | Concatenate and print files in reverse |
| `tailx`      | `tail`      | Print the last part of each file, optionally following it as it grows |
| `teex`       | `tee`       | Copy standard input to each FILE, and also to standard output |
| `testx`      | `test`      | Check file types and compare values |
| `timeoutx`   | `timeout`   | Run a command with a time limit |
| `touchx`     | `touch`     | Update the access and modification times of each FILE to the current time |
| `truex`      | `true`      | Do nothing, successfully |
| `truncatex`  | `truncate`  | Shrink or extend the size of each FILE to the specified size |
| `trx`        | `tr`        | Translate, squeeze, and/or delete characters from standard input |
| `ttyx`       | `tty`       | Print the file name of the terminal connected to standard input |
| `unamex`     | `uname`     | Print system information |
| `unexpandx`  | `unexpand`  | Convert spaces to tabs |
| `uniqx`      | `uniq`      | Report or omit repeated lines |
| `unlinkx`    | `unlink`    | Call the unlink function to remove the specified file |
| `uptimex`    | `uptime`    | Tell how long the system has been running |
| `wcx`        | `wc`        | Print newline, word, and byte counts for each file |
| `whoamix`    | `whoami`    | Print the user name associated with the current effective user ID |
| `whox`       | `who`       | Show who is logged on |
| `xargsx`     | `xargs`     | Build and run command lines from standard input |
| `yesx`       | `yes`       | Output a string repeatedly until killed |

`gnux --list` prints the tool names; `gnux --help` lists them with a summary. The few
remaining coreutils tools are on the [roadmap](#roadmap).

Four of the tools are not part of GNU coreutils: `grepx` (GNU grep), `sedx` (GNU sed), and
`findx` and `xargsx` (GNU findutils). They are here because a build or CI script rarely stays
inside coreutils: the first `find . -name '*.map' | xargs rm` or `sed -i` would otherwise send
it back to the system's own tools, which differ between Linux, macOS and Windows.

## Requirements

**To run gnux**

- Linux on amd64, arm64 or riscv64, macOS or FreeBSD on amd64 or arm64, or Windows on amd64 or
  arm64 (the platforms CI builds and releases). The binary
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

- A Go module on Go 1.25 or newer. The engines depend only on the standard library,
  `github.com/mattn/go-isatty`, `golang.org/x/sys` (system calls the standard library does not
  wrap, on Windows and the BSDs) and `golang.org/x/crypto` (BLAKE2b), all pure Go; cobra is
  compiled only into the command-line packages under `cmd/`.

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
v=v0.0.1
curl -LO "https://github.com/braswelljr/gnux/releases/download/$v/gnux-$v-linux-amd64.tar.gz"
curl -LO "https://github.com/braswelljr/gnux/releases/download/$v/checksums.txt"
sha256sum --check --ignore-missing checksums.txt
tar -xzf "gnux-$v-linux-amd64.tar.gz" && sudo mv "gnux-$v-linux-amd64/gnux" /usr/local/bin/
```

### Package managers

From v0.0.2 on, each release is also published to Homebrew (macOS and Linux), Scoop and winget
(Windows), the Snap Store and the AUR (Linux):

```bash
brew tap braswelljr/gnux https://github.com/braswelljr/gnux
brew install --cask braswelljr/gnux/gnux   # also installs bash, zsh and fish completion
scoop bucket add gnux https://github.com/braswelljr/gnux
scoop install gnux/gnux
winget install braswelljr.gnux
sudo snap install gnux                     # strictly confined: your home directory only
yay -S gnux-bin                            # Arch Linux, from the AUR
```

and carries a `.deb`, an `.rpm` and an `.apk` for amd64, arm64 and riscv64:

```bash
v=v0.0.2
curl -LO "https://github.com/braswelljr/gnux/releases/download/$v/gnux-$v-linux-amd64.deb"
sudo dpkg -i "gnux-$v-linux-amd64.deb"                        # Debian, Ubuntu
sudo rpm -i "gnux-$v-linux-amd64.rpm"                         # Fedora, RHEL, openSUSE
sudo apk add --allow-untrusted "gnux-$v-linux-amd64.apk"      # Alpine
```

These install the `gnux` binary; link the tool names to it as shown under `go install` below,
or run each tool as `gnux rmx ...`. A package manager shows up a little after the release it
belongs to, winget last, since each new version is reviewed there.

### A particular version

`gnux --version` reports what is installed. To install one version rather than the newest:

| With             | Command                                                                  |
|------------------|--------------------------------------------------------------------------|
| A release archive or package | set `v=` above to the tag: every release stays available     |
| `go install`     | `go install github.com/braswelljr/gnux@v0.0.2`                           |
| winget           | `winget install braswelljr.gnux --version 0.0.2`                         |
| Scoop            | `scoop install gnux/gnux@0.0.2`                                          |
| Snap             | `sudo snap install gnux --channel=edge` for the newest prerelease        |
| setup-gnux       | `version: v0.0.2`                                                        |

Homebrew and the AUR carry the newest version only.

### In GitHub Actions

The repository is also a GitHub Action, setup-gnux. It installs a release on a Linux, macOS or
Windows runner, checks it against the release's checksums and puts gnux and every tool name on
`PATH`:

```yaml
- uses: braswelljr/gnux@v0
  with:
    version: latest       # or a tag, such as v0.0.1
    gnu-names: false      # true also links rm, cp, ls, ... ahead of the runner's own
- run: rmx -rf build && mkdirx -p build && cpx -r assets build/
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

### Shell completion

`gnux completion SHELL` prints a completion script for bash, zsh, fish or powershell; the
release archives carry them pre-generated in `completions/`.

```bash
source <(gnux completion bash)                    # bash, this session
gnux completion zsh > "${fpath[1]}/_gnux"         # zsh, permanently
gnux completion fish > ~/.config/fish/completions/gnux.fish
```

```powershell
gnux completion powershell | Out-String | Invoke-Expression
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
| Test     | `go vet` and `go test -race` on Linux, macOS and Windows; a multi-call smoke test; the WebAssembly build; a short fuzzing run of the shell |
| Build    | cross-compiles `gnux` for Linux, macOS, FreeBSD and Windows and uploads the binaries                     |
| Lint     | `golangci-lint`                                                                                          |
| Docs     | regenerates `documentation/` and `web/content/docs/` and fails if the committed copies are stale         |
| Web      | builds the WebAssembly binary, then lints, type-checks, tests and builds the website                     |
| Action   | installs a release with the setup-gnux action (`action.yml`) on Linux, macOS and Windows runners and uses the tools |
| Release  | on a `v*` tag: re-runs the checks, then GoReleaser (`.goreleaser.yaml`) builds an archive per platform, publishes them with `.deb`, `.rpm` and `.apk` packages, checksums, a provenance attestation and the tag's [`CHANGELOG.md`](CHANGELOG.md) section as notes, and publishes to Homebrew, Scoop, winget, the Snap Store and the AUR ([how](docs/workflows/publishing-a-release.md)) |

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

Not yet implemented, roughly in priority order. The full list of planned tools, distribution
channels and integrations, ticked off as they land, is
[`docs/requirements/checklist.md`](docs/requirements/checklist.md).

- **Tools:** `[` (as another name for `testx`), `basenc`, `numfmt`, `pr`, `tsort`, `factor`,
  `sum`, `mkfifo`, `mknod`, `pathchk`, `logname`, `groups`, `users`, `pinky`, `stdbuf`, `sync`,
  `shred`, `dircolors`, `dir`, `vdir`, `arch`, `hostid`, `ptx`, `stty`
- **Out of scope by design:** `chroot`, `chcon` and `runcon`, which change system-wide state
  that gnux leaves to the system's own tools

## License

See [LICENSE](LICENSE).
