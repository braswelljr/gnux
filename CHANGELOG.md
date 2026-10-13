# Changelog

Every notable change to gnux, release by release, newest first. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and gnux uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Changes land under Unreleased as they are made, and that section becomes the next version when it is tagged; the release workflow publishes the version's section as its GitHub release notes.

## [Unreleased]

### Added

- Four tools from beyond coreutils, the ones a build or CI script reaches for next, bringing
  gnux to 85: `grepx` (GNU grep), `sedx` (GNU sed), and `findx` and `xargsx` (GNU findutils).
  Each mirrors its GNU counterpart's options, diagnostics and exit statuses, and lists what it
  does not implement on its reference page.
  - `grepx`: basic, extended and fixed-string patterns, context lines, recursive search with
    `--include` / `--exclude` / `--exclude-dir`, counts, file lists, `-o`, colors.
  - `sedx`: the whole script language (addresses, `s` with its flags and case conversions,
    hold space, branches, `a` / `i` / `c`, `r` / `w`) and in-place editing with `-i[SUFFIX]`.
  - `findx`: the expression language with its operators, the name, type, size, time,
    permission and owner tests, `-printf`, `-delete`, `-prune`, and `-exec` / `-execdir` /
    `-ok`.
  - `xargsx`: quoting, `-0`, `-d`, `-n`, `-L`, `-I`, `-P`, `-p`, `-t`, `-r`, and GNU's exit
    statuses 123 to 127.
- `xargsx` and `findx -exec` run a command named after a gnux tool in-process, so
  `findx . -name '*.map' | xargsx rm` works on a machine with no GNU tools installed, Windows
  included, and inside `gnux shell`. Any other command is a program found on `PATH`.

- A benchmark suite over the real tools: `make bench` reports throughput and allocations for
  each tool on generated text, a directory tree and shell pipelines; `make bench/save` and
  `make bench/compare` compare two runs, and `make bench/check` fails when a benchmark got
  slower or allocates more, which the new Bench workflow runs on every pull request against
  its base commit. `make fuzz` fuzzes every parser.

### Changed

- A filter answers each line as it gets it: `catx -n`, `nlx`, `cutx`, `trx`, `foldx`,
  `expandx`, `unexpandx`, `pastex`, `headx`, `uniqx`, `grepx` and `sedx` pass their output on
  whenever their input has run dry, instead of holding it until a 64 KiB buffer fills. Behind
  `tail -f`, a build log or a terminal they no longer look hung; on bulk input nothing changes.
- `gnux shell` starts about 25 times sooner: it no longer builds every tool's command before
  reading a script.
- A tool that panics is reported as `TOOL: internal error` with status 2 instead of ending the
  process, so one tool's bug does not take a shell pipeline, the playground or an embedding
  program down with it. That holds for a panic on a goroutine the tool started too (`rmx`
  workers, `dux`, `xargsx -P`, a pipeline stage): `core.Trap` carries it back.
- Running a tool with tens of thousands of arguments, as `xargsx` and `findx -exec` do, no
  longer scans them several times before the tool starts: about 20 times less memory for such
  a call, and a third less work for every call.
- Patterns in `grepx`, `sedx` and `findx -regex` are matched by Go's RE2 engine, which runs in
  linear time but has no back references: a pattern that uses one (`\(a\)\1`) is refused with
  a message instead of being matched differently. `sedx` refuses its `e` command, and `grepx`
  refuses `-P`.

- `gnux TOOL` starts as fast as the tool does under its own name: a first argument that names a
  tool runs it directly, without building the root command and every other tool's flags first
  (about 100 times less work before the tool starts). The gnux shell no longer rebuilds the
  tool list for each command it runs.
- Every tool reads and writes through 64 KiB buffers (`core.NewReader`, `core.NewWriter`,
  `core.BufferSize`) where it used 4 KiB ones, and `odx` no longer reads its input one output
  line at a time.
- Faster on large inputs, with the same output: `odx` (about 6 times), `cksumx` with the default
  CRC (5 times), `seqx` with a fractional operand (14 times), `catx -n`, `nlx`, `uniqx -c`,
  `shufx` and `sortx -r` (2 times).

### Fixed

- `seqx` keeps the decimals of a whole-valued operand, as GNU seq does: `seqx 1.0 3` prints
  `1.0`, `2.0`, `3.0`, and `seqx -w 1 1.00 3` prints `1.00`, `2.00`, `3.00`. It used to print
  them as integers.

## [0.0.2] - 2026-10-12

### Added

- 59 more tools, bringing gnux from 22 to 81 and covering nearly all of coreutils:
  - text: `sortx`, `uniqx`, `cutx`, `trx`, `teex`, `seqx`, `printfx`, `pastex`, `joinx`,
    `commx`, `nlx`, `foldx`, `fmtx`, `expandx`, `unexpandx`, `tacx`, `splitx`, `csplitx`,
    `shufx`, `odx`;
  - checksums and encoding: `md5sumx`, `sha1sumx`, `sha224sumx`, `sha256sumx`, `sha384sumx`,
    `sha512sumx`, `b2sumx`, `cksumx`, `base32x`, `base64x`;
  - file attributes: `chmodx`, `chownx`, `chgrpx`, `statx`, `dux`, `dfx`, `mktempx`, `installx`,
    `truncatex`, `readlinkx`, `linkx`, `unlinkx`;
  - system: `datex`, `unamex`, `whoamix`, `idx`, `hostnamex`, `nprocx`, `printenvx`, `ttyx`,
    `whox`, `uptimex`, `timeoutx`, `nicex`, `nohupx`, `killx`, `testx`, `exprx`, `ddx`.
- Where GNU has a known rough edge, the new tools fix it and document it: `dux` reads
  directories in parallel and lists them in a stable order; recursive `chmodx`, `chownx` and
  `chgrpx` refuse `/` by default; `nicex` maps niceness onto Windows priority classes and
  `nohupx` detaches from the console on Windows, so both work there; `killx 1234 -9` never
  signals process group 9; `dfx`, `whox`, `uptimex` and `statx` report the same fields on Linux,
  macOS, FreeBSD and Windows without cgo.
- Releases built by GoReleaser (`.goreleaser.yaml`), for four more platforms (`linux/riscv64`,
  `freebsd/amd64`, `freebsd/arm64`, `windows/arm64`), with shell completions in every archive
  and a build provenance attestation. Archive names are unchanged.
- Homebrew cask (`brew install --cask braswelljr/tap/gnux`), Scoop and winget manifests,
  published with each release.
- setup-gnux, a GitHub Action (`uses: braswelljr/gnux@v0`) that installs a checksum-verified
  release on Linux, macOS and Windows runners and puts every tool on `PATH`.
- `gnux completion bash|zsh|fish|powershell`, listed in `gnux --help`.
- Documentation examples can show their output (`docmeta.Example.Output`), rendered under the
  commands in every reference page.
- Website: an install page with copyable steps per platform (macOS, Windows, Linux and FreeBSD,
  Go, GitHub Actions, Docker); a pausable marquee of every tool; a "why gnux" strip; and, on
  every tool page, a GNU reference and examples with bash and PowerShell tabs that remember your
  choice.
- Library: `core.FlushOutput`, which reports a failed final write (`write error: No space left
  on device`) instead of exiting 0, and `shell.Shell.Glob`, the injection point for globbing.
- Fuzz tests for the shell's parser and expansion, run in CI.
- `docs/requirements/checklist.md`, the list of planned tools and integrations.

### Changed

- Release tags must be full versions (`v1.2.3`, `v1.2.3-rc.1`); each stable release moves its
  major tag (`v0`), which the GitHub Action is used by.
- Dependabot also updates the website's pnpm packages.

## [0.0.1] - 2026-10-11

### Added

- 22 GNU-compatible tools in one multi-call binary, `gnux`: `basenamex`, `catx`, `cpx`, `dirnamex`, `echox`, `envx`, `falsex`, `headx`, `lnx`, `lsx`, `mkdirx`, `mvx`, `pwdx`, `realpathx`, `rmx`, `rmdirx`, `sleepx`, `tailx`, `touchx`, `truex`, `wcx` and `yesx`. Each runs as its own name, as its GNU name through a link, or as `gnux <tool>`, with GNU's flags, diagnostics and exit statuses.
- `gnux shell`, a small portable POSIX sh subset whose commands are the tools, run in-process, so one script behaves the same on Linux, macOS and Windows.
- Every tool's engine as an importable Go package under `core/`, plus `cmd.RunTool` and `cmd.NewShell` for running tools and scripts from Go.
- `gnux help TOOL [OPTION]` for a tool's full manual, or one option in depth.
- The reference in markdown, man, YAML and reStructuredText under `documentation/`, generated from the code and checked in CI.
- A website with the documentation, a page for every option of every tool, and a playground that runs the real gnux, compiled to WebAssembly, on an in-memory filesystem.
- Release archives for Linux and macOS (amd64, arm64) and Windows (amd64), with a checksums file.

### Changed

- The project is now gnux, a portable implementation of the GNU core utilities. `rmx`, which it started as, was rewritten as one of its tools, with GNU's full `rm` flag surface.

[Unreleased]: https://github.com/braswelljr/gnux/compare/v0.0.2...HEAD
[0.0.2]: https://github.com/braswelljr/gnux/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/braswelljr/gnux/releases/tag/v0.0.1
