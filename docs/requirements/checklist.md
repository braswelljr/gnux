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

- [X] `sortx` (FR-88)
- [X] `uniqx` (FR-89)
- [X] `cutx` (FR-90)
- [X] `trx` (FR-91)
- [X] `teex` (FR-92)
- [X] `seqx` (FR-93)
- [X] `printfx` (FR-94)
- [X] `pastex` (FR-126)
- [X] `joinx` (FR-116)
- [X] `nlx` (FR-122)
- [X] `foldx` (FR-112)
- [X] `fmtx` (FR-111)
- [X] `expandx` (FR-109), `unexpandx` (FR-143)
- [X] `tacx` (FR-137)
- [X] `splitx` (FR-135), `csplitx` (FR-104)
- [X] `commx` (FR-103)
- [X] `tsortx` (FR-159)
- [X] `factorx` (FR-157)
- [X] `shufx` (FR-134)
- [X] `odx` (FR-125)
- [X] `numfmtx` (FR-175)
- [X] `prx` (FR-178), `ptxx` (FR-179)

### Checksums and encoding

- [X] `md5sumx` (FR-119), `sha1sumx` (FR-129), `sha224sumx` (FR-130), `sha256sumx` (FR-131),
  `sha384sumx` (FR-132), `sha512sumx` (FR-133), `b2sumx` (FR-96)
- [X] `cksumx` (FR-102), `sumx` (FR-158)
- [X] `base32x` (FR-97), `base64x` (FR-98), `basencx` (FR-170)

### File attributes

- [X] `chmodx` (FR-100), `chownx` (FR-101), `chgrpx` (FR-99)
- [X] `statx` (FR-136)
- [X] `dirx` (FR-160), `vdirx` (FR-164)
- [X] `dux` (FR-108), `dfx` (FR-107)
- [X] `mktempx` (FR-120)
- [X] `installx` (FR-115)
- [X] `truncatex` (FR-140)
- [X] `readlinkx` (FR-128)
- [X] `linkx` (FR-118), `unlinkx` (FR-144)
- [X] `mkfifox` (FR-173), `mknodx` (FR-174)
- [X] `pathchkx` (FR-176)
- [X] `shredx` (FR-180), `syncx` (FR-183)

### System

- [X] `datex` (FR-105)
- [X] `unamex` (FR-142), `archx` (FR-156), `whoamix` (FR-146), `idx` (FR-114), `hostnamex` (FR-113),
  `nprocx` (FR-124), `printenvx` (FR-127), `groupsx` (FR-161), `usersx` (FR-163),
  `lognamex` (FR-162), `ttyx` (FR-141), `whox` (FR-147), `uptimex` (FR-145)
- [X] `timeoutx` (FR-139), `nicex` (FR-121), `nohupx` (FR-123), `killx` (FR-117)
- [X] `testx` (FR-138), `exprx` (FR-110)
- [X] `[` as another name for `testx`, requiring the closing `]` (FR-138)
- [X] `ddx` (FR-106)
- [X] `hostidx` (FR-172), `pinkyx` (FR-177)
- [X] `dircolorsx` (FR-171), `sttyx` (FR-182), `stdbufx` (FR-181)

### Beyond coreutils

The tools a build or CI script reaches for next to coreutils, from GNU grep, GNU sed and GNU
findutils.

- [X] `grepx` (FR-149)
- [X] `sedx` (FR-150)
- [X] `findx` (FR-148)
- [X] `xargsx` (FR-151)

## 2. Distribution

- [X] Releases built by GoReleaser from `.goreleaser.yaml`, keeping the existing archive names
  (FR-36, NFR-1)
- [X] More release platforms: `windows/arm64`, `linux/riscv64`, `freebsd/amd64`,
  `freebsd/arm64` (NFR-1)
- [X] Homebrew cask, with shell completions (FR-36)
  - Served from this repository (`Casks/gnux.rb`), written by the release job. Needs nothing.
- [X] Scoop manifest (FR-36)
  - Served from this repository (`bucket/gnux.json`). Needs nothing.
- [X] winget manifest, sent as a pull request to `microsoft/winget-pkgs` (FR-36)
  - Needs: a fork at `braswelljr/winget-pkgs` and a `WINGET_GITHUB_TOKEN` secret (a classic
    token with `public_repo`).
- [X] `.deb`, `.rpm` and `.apk` packages on every release (FR-36)
- [X] A snap, strictly confined, published to the Snap Store (FR-36)
  - Needs: the name `gnux` registered and a `SNAPCRAFT_STORE_CREDENTIALS` secret.
- [X] `gnux-bin` on the AUR (FR-36)
  - Needs: an AUR account with an SSH key and an `AUR_KEY` secret.
- [ ] A formula in `homebrew/core`, so that `brew install gnux` needs no tap
- [X] Build provenance attestation for every release archive (FR-36)
- [X] An SBOM for every release archive (GoReleaser `sboms`, with syft)
- [X] cosign signatures for `checksums.txt`
- [X] The Docker image published to GHCR on every release
- [X] A `FROM scratch` image with the per-tool links, as a busybox-style base
- [ ] A Nix package
- [X] An npm package (`gnux`, with a package per platform) that installs the right binary, for
  Node projects (`tools/npm`, the `npm` job of the Release workflow)
- [x] A WASI build (`GOOS=wasip1`) for wasmtime, wasmer and edge runtimes (`make wasi`, compiled
  in the Test workflow)

## 3. Integrations

- [X] The setup-gnux GitHub Action, `uses: braswelljr/gnux@v0` (FR-36)
  - Needs: a release after this change, so the release workflow creates the `v0` tag; until
    then use `uses: braswelljr/gnux@main`. Optionally, list it on the GitHub Marketplace.
- [X] Shell completion: `gnux completion bash|zsh|fish|powershell`, listed in `gnux --help` and
  shipped in every archive (FR-95)
- [X] A shebang story for scripts: `#!/usr/bin/env -S gnux shell` (`cmd/shell_test.go`, the
  portable scripting guide)
- [X] A documented replacement for rimraf, cpy-cli and mkdirp in npm `scripts`
  (`documentation/guides/node-and-npm.md`)
- [x] An MCP server mode (`gnux mcp`) exposing the tools as sandboxed file operations (FR-169, on
  the sandbox of FR-168)
- [ ] An embeddable playground (`<iframe>` or web component) for other documentation sites
- [ ] Shareable playground sessions that carry the in-memory filesystem, not only `?cmd=`

## 4. Quality

- [X] Fuzz tests for the shell's parser and expansion, run in CI (NFR-48)
- [X] Fuzz tests for `sedx` scripts, `grepx` patterns, `findx` expressions and `xargsx` input,
  run in CI (NFR-48)
- [X] A benchmark suite over the real tools, `make bench`, with `benchstat` comparison and a
  pass/fail check against the base commit on every pull request (NFR-52)
- [X] Tests for bounded memory, leftover goroutines and open files, and line-by-line response
  (NFR-7, NFR-50, NFR-51)
- [X] Dependabot for the website's pnpm packages, next to Go modules and Actions (NFR-20)
- [X] A GNU conformance harness: run each case through GNU coreutils and gnux on Linux and
  diff stdout, stderr and the exit status (FR-166)
- [ ] Benchmarks against GNU coreutils and uutils for `cpx -r`, `rmx -r`, `wcx` and `sortx`
- [X] Fuzz tests for the option parsers of `printfx`, `seqx`, `trx` and `cutx` (NFR-48)

## 5. Language and website

- [X] Shell control flow and command substitution: `if`, `for`, `while`, `case`, `$(...)`,
  script arguments and `-u` (FR-152 to FR-155)
- [ ] Shell functions and arithmetic: a deliberate decision first, since the shell is a small
  portable subset by design
- [X] `llms.txt` and `llms-full.txt` on the website (`web/src/app/llms*.txt`)
- [ ] A plain markdown copy of every documentation page (`/docs/<page>.md`) for agents
