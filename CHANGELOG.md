# Changelog

Every notable change to gnux, release by release, newest first. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and gnux uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Changes land under Unreleased as they are made, and that section becomes the next version when it is tagged; the release workflow publishes the version's section as its GitHub release notes.

## [Unreleased]

### Added

- `seqx` (GNU `seq`) and `printfx` (GNU `printf`): number sequences with `-f`, `-s` and `-w`, exact decimal steps and negative operands without `--`; and C-style formatting with every GNU conversion, flag, `*` width and precision, escape (`\u`, `\x`, octal), `%b`, `%q`, format reuse and GNU's numeric diagnostics.
- `sortx`, GNU `sort`: sort, merge (`-m`) or check (`-c`, `-C`) lines, with keys (`-k`), field separators (`-t`), numeric, general-numeric, human-numeric, month and version ordering, `-u`, `-s`, `-o` (in place), `-z` and `--files0-from`.
- `trx` (GNU `tr`) and `teex` (GNU `tee`): translate, squeeze and delete bytes with GNU's full SET syntax, and copy standard input to files with `-a`, `-p` and `--output-error`.
- `cutx` (GNU `cut`) and `uniqx` (GNU `uniq`), with GNU's full flag surfaces: byte, character and field lists with `--complement` and `--output-delimiter`, and counting, duplicate selection, `--all-repeated` and `--group` with field, character and case-insensitive comparison.
- Releases are built by GoReleaser (`.goreleaser.yaml`) and now also cover `linux/riscv64`, `freebsd/amd64`, `freebsd/arm64` and `windows/arm64`, with a build provenance attestation for every archive. Archive names are unchanged.
- Homebrew cask (`brew install --cask braswelljr/tap/gnux`), Scoop and winget manifests, published with each release.
- setup-gnux, a GitHub Action (`uses: braswelljr/gnux@v0`) that installs a verified release on Linux, macOS and Windows runners and puts every tool on `PATH`.
- `gnux completion bash|zsh|fish|powershell`, now listed in `gnux --help`, with the scripts shipped in every release archive.
- Fuzz tests for the shell's parser and expansion, run in CI.
- `shell.Shell.Glob`, the injection point for globbing (defaults to `filepath.Glob`), for hosts with their own filesystem and for tests.
- `docs/requirements/checklist.md`, the list of planned tools and integrations.

### Changed

- Release tags must be full versions (`v1.2.3`, `v1.2.3-rc.1`); each stable release moves its major tag (`v0`) for the GitHub Action.
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

[Unreleased]: https://github.com/braswelljr/gnux/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/braswelljr/gnux/releases/tag/v0.0.1
