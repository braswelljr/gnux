# Changelog

Every notable change to gnux, release by release, newest first. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and gnux uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Changes land under Unreleased as they are made, and that section becomes the next version when it is tagged; the release workflow publishes the version's section as its GitHub release notes.

## [Unreleased]

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
