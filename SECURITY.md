# Security policy

gnux runs with your permissions on your files, and `gnux mcp` and `gnux shell --sandbox` run
commands for AI agents inside a kernel sandbox, so a security problem in it matters. This page
says how to report one and what gnux does to prevent them.

## Reporting a vulnerability

Please report it privately, not in a public issue:
[report a vulnerability](https://github.com/braswelljr/gnux/security/advisories/new) through
GitHub's private vulnerability reporting. Say which version (`gnux --version`) and platform, what
an attacker can do, and how to reproduce it. You will get an answer within a few days, and
credit in the advisory and the release notes unless you prefer otherwise.

The most serious reports are those that break a boundary gnux promises:

- a command run through `gnux mcp` or `gnux shell --sandbox` that reads or writes outside the
  directories it was given (Landlock on Linux, Seatbelt on macOS, AppContainer on Windows,
  Capsicum on FreeBSD);
- a tool that follows a symbolic link, crosses a file system or removes a root it was told not
  to (`rmx --preserve-root`, `--one-file-system`, and the like);
- the playground reaching anything outside its in-memory file system;
- a release archive, package or image that does not match its signed `checksums.txt`.

## Supported versions

Fixes go into the newest release, and a new release ships them. Use the latest version
(`brew upgrade`, `scoop update`, `go install github.com/braswelljr/gnux@latest`, or the
setup-gnux action's default `version: latest`).

## What gnux does to prevent them

- **Scanning.** Every change, and every week, is checked by govulncheck on each release
  platform, `pnpm audit`, zizmor (the workflows), gitleaks (the whole history) and CodeQL (the
  Go, TypeScript and workflow code): `.github/workflows/security.yml`.
- **A patched toolchain.** Releases are built with the Go release named in `go.mod`'s
  `toolchain` line, kept at the newest patch release.
- **Locked-down automation.** Workflows run with read-only permissions, every action is pinned
  to a commit, no checkout keeps its credentials but the one that must push a tag, and no release
  job restores a cache.
- **Verifiable releases.** Each release's `checksums.txt` is signed with cosign (keyless, tied
  to the Release workflow) and comes with a build provenance attestation and an SBOM per
  archive. `docs/workflows/publishing-a-release.md` shows how to verify them.
- **Small surface.** Three direct Go dependencies, no cgo, a container image that runs as an
  unprivileged user on a supported Alpine release.
