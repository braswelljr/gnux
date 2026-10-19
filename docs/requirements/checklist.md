# Checklist

Every planned upgrade and integration for gnux, in one list, ticked off as it lands. It is the
working companion to the requirement registers: an item here says what is planned or done, and
the register entry it cites says what the behavior must be and how it is verified. Add an item
when a feature is proposed, tick it in the change that delivers it, and note anything that still
has to happen outside the repository (a secret, a repository, a tag) under it.

## How to read this list

- `[x]` is done and verified in this repository; `[ ]` is not started or not finished.
- The IDs after an item are the requirements it is tracked by (see
  [functional-requirements](functional-requirements.md) and
  [non-functional-requirements](non-functional-requirements.md)).
- "Needs" lists what remains to do outside the code before the item works for users.

## 1. Tools

The roadmap in the README, as a list. Each tool is an engine in `core/<gnu>/`, an adapter in
`cmd/<gnu>x/`, a registry entry, a functional requirement, a README row and an adoption guide
entry ([adding-a-tool](../workflows/adding-a-tool.md)).

### Text

- [x] `sortx` (FR-88)
- [x] `uniqx` (FR-89)
- [x] `cutx` (FR-90)
- [x] `trx` (FR-91)
- [x] `teex` (FR-92)
- [x] `seqx` (FR-93)
- [x] `printfx` (FR-94)
- [x] `pastex` (FR-126)
- [x] `joinx` (FR-116)
- [x] `nlx` (FR-122)
- [x] `foldx` (FR-112)
- [x] `fmtx` (FR-111)
- [x] `expandx` (FR-109), `unexpandx` (FR-143)
- [x] `tacx` (FR-137)
- [x] `splitx` (FR-135), `csplitx` (FR-104)
- [x] `commx` (FR-103)
- [x] `tsortx` (FR-159)
- [x] `factorx` (FR-157)
- [x] `shufx` (FR-134)
- [x] `odx` (FR-125)

### Checksums and encoding

- [x] `md5sumx` (FR-119), `sha1sumx` (FR-129), `sha224sumx` (FR-130), `sha256sumx` (FR-131),
  `sha384sumx` (FR-132), `sha512sumx` (FR-133), `b2sumx` (FR-96)
- [x] `cksumx` (FR-102), `sumx` (FR-158)
- [x] `base32x` (FR-97), `base64x` (FR-98)

### File attributes

- [x] `chmodx` (FR-100), `chownx` (FR-101), `chgrpx` (FR-99)
- [x] `statx` (FR-136)
- [x] `dux` (FR-108), `dfx` (FR-107)
- [x] `mktempx` (FR-120)
- [x] `installx` (FR-115)
- [x] `truncatex` (FR-140)
- [x] `readlinkx` (FR-128)
- [x] `linkx` (FR-118), `unlinkx` (FR-144)

### System

- [x] `datex` (FR-105)
- [x] `unamex` (FR-142), `archx` (FR-156), `whoamix` (FR-146), `idx` (FR-114), `hostnamex` (FR-113),
  `nprocx` (FR-124), `printenvx` (FR-127), `ttyx` (FR-141), `whox` (FR-147), `uptimex` (FR-145)
- [x] `timeoutx` (FR-139), `nicex` (FR-121), `nohupx` (FR-123), `killx` (FR-117)
- [x] `testx` (FR-138), `exprx` (FR-110)
- [ ] `[` as another name for `testx`, requiring the closing `]`
- [x] `ddx` (FR-106)

### Beyond coreutils

The tools a build or CI script reaches for next to coreutils, from GNU grep, GNU sed and GNU
findutils.

- [x] `grepx` (FR-149)
- [x] `sedx` (FR-150)
- [x] `findx` (FR-148)
- [x] `xargsx` (FR-151)

## 2. Distribution

- [x] Releases built by GoReleaser from `.goreleaser.yaml`, keeping the existing archive names
  (FR-36, NFR-1)
- [x] More release platforms: `windows/arm64`, `linux/riscv64`, `freebsd/amd64`,
  `freebsd/arm64` (NFR-1)
- [x] Homebrew cask, with shell completions (FR-36)
  - Served from this repository (`Casks/gnux.rb`), written by the release job. Needs nothing.
- [x] Scoop manifest (FR-36)
  - Served from this repository (`bucket/gnux.json`). Needs nothing.
- [x] winget manifest, sent as a pull request to `microsoft/winget-pkgs` (FR-36)
  - Needs: a fork at `braswelljr/winget-pkgs` and a `WINGET_GITHUB_TOKEN` secret (a classic
    token with `public_repo`).
- [x] `.deb`, `.rpm` and `.apk` packages on every release (FR-36)
- [x] A snap, strictly confined, published to the Snap Store (FR-36)
  - Needs: the name `gnux` registered and a `SNAPCRAFT_STORE_CREDENTIALS` secret.
- [x] `gnux-bin` on the AUR (FR-36)
  - Needs: an AUR account with an SSH key and an `AUR_KEY` secret.
- [ ] A formula in `homebrew/core`, so that `brew install gnux` needs no tap
- [x] Build provenance attestation for every release archive (FR-36)
- [ ] An SBOM for every release archive (GoReleaser `sboms`, with syft)
- [ ] cosign signatures for `checksums.txt`
- [ ] The Docker image published to GHCR on every release
- [ ] A `FROM scratch` image with the per-tool links, as a busybox-style base
- [ ] A Nix package
- [ ] An npm package (`@gnux/cli`) that installs the right binary, for Node projects
- [ ] A WASI build (`GOOS=wasip1`) for wasmtime, wasmer and edge runtimes

## 3. Integrations

- [x] The setup-gnux GitHub Action, `uses: braswelljr/gnux@v0` (FR-36)
  - Needs: a release after this change, so the release workflow creates the `v0` tag; until
    then use `uses: braswelljr/gnux@main`. Optionally, list it on the GitHub Marketplace.
- [x] Shell completion: `gnux completion bash|zsh|fish|powershell`, listed in `gnux --help` and
  shipped in every archive (FR-95)
- [ ] A shebang story for scripts: `#!/usr/bin/env -S gnux shell`
- [ ] A documented replacement for rimraf, cpy-cli and mkdirp in npm `scripts`
- [ ] An MCP server mode (`gnux mcp`) exposing the tools as sandboxed file operations
- [ ] An embeddable playground (`<iframe>` or web component) for other documentation sites
- [ ] Shareable playground sessions that carry the in-memory filesystem, not only `?cmd=`

## 4. Quality

- [x] Fuzz tests for the shell's parser and expansion, run in CI (NFR-48)
- [x] Fuzz tests for `sedx` scripts, `grepx` patterns, `findx` expressions and `xargsx` input,
  run in CI (NFR-48)
- [x] A benchmark suite over the real tools, `make bench`, with `benchstat` comparison and a
  pass/fail check against the base commit on every pull request (NFR-52)
- [x] Tests for bounded memory, leftover goroutines and open files, and line-by-line response
  (NFR-7, NFR-50, NFR-51)
- [x] Dependabot for the website's pnpm packages, next to Go modules and Actions (NFR-20)
- [ ] A GNU conformance harness: run each case through GNU coreutils and gnux on Linux and
  diff stdout, stderr and the exit status
- [ ] Benchmarks against GNU coreutils and uutils for `cpx -r`, `rmx -r`, `wcx` and `sortx`
- [ ] Fuzz tests for the option parsers of `printfx`, `seqx`, `trx` and `cutx`

## 5. Language and website

- [x] Shell control flow and command substitution: `if`, `for`, `while`, `case`, `$(...)`,
  script arguments and `-u` (FR-152 to FR-155)
- [ ] Shell functions and arithmetic: a deliberate decision first, since the shell is a small
  portable subset by design
- [x] `llms.txt` and `llms-full.txt` on the website (`web/src/app/llms*.txt`)
- [ ] A plain markdown copy of every documentation page (`/docs/<page>.md`) for agents
