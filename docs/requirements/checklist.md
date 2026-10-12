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
- [x] `shufx` (FR-134)
- [x] `odx` (FR-125)

### Checksums and encoding

- [x] `md5sumx` (FR-119), `sha1sumx` (FR-129), `sha224sumx` (FR-130), `sha256sumx` (FR-131),
  `sha384sumx` (FR-132), `sha512sumx` (FR-133), `b2sumx` (FR-96)
- [x] `cksumx` (FR-102)
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
- [x] `unamex` (FR-142), `whoamix` (FR-146), `idx` (FR-114), `hostnamex` (FR-113), `nprocx` (FR-124),
  `printenvx` (FR-127), `ttyx` (FR-141), `whox` (FR-147), `uptimex` (FR-145)
- [x] `timeoutx` (FR-139), `nicex` (FR-121), `nohupx` (FR-123), `killx` (FR-117)
- [x] `testx` (FR-138), `exprx` (FR-110)
- [ ] `[` as another name for `testx`, requiring the closing `]`
- [x] `ddx` (FR-106)

## 2. Distribution

- [x] Releases built by GoReleaser from `.goreleaser.yaml`, keeping the existing archive names
  (FR-36, NFR-1)
- [x] More release platforms: `windows/arm64`, `linux/riscv64`, `freebsd/amd64`,
  `freebsd/arm64` (NFR-1)
- [x] Homebrew cask, with shell completions (FR-36)
  - Needs: the `braswelljr/homebrew-tap` repository and a `HOMEBREW_TAP_GITHUB_TOKEN` secret
    that can write to it.
- [x] Scoop manifest (FR-36)
  - Needs: the `braswelljr/scoop-bucket` repository and a `SCOOP_BUCKET_GITHUB_TOKEN` secret.
- [x] winget manifest, sent as a pull request to `microsoft/winget-pkgs` (FR-36)
  - Needs: a fork at `braswelljr/winget-pkgs` and a `WINGET_GITHUB_TOKEN` secret (a classic
    token with `public_repo`).
- [x] Build provenance attestation for every release archive (FR-36)
- [ ] An SBOM for every release archive (GoReleaser `sboms`, with syft)
- [ ] cosign signatures for `checksums.txt`
- [ ] The Docker image published to GHCR on every release
- [ ] A `FROM scratch` image with the per-tool links, as a busybox-style base
- [ ] AUR and Nix packages
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
- [x] Dependabot for the website's pnpm packages, next to Go modules and Actions (NFR-20)
- [ ] A GNU conformance harness: run each case through GNU coreutils and gnux on Linux and
  diff stdout, stderr and the exit status
- [ ] Benchmarks against GNU coreutils and uutils for `cpx -r`, `rmx -r`, `wcx` and `sortx`
- [ ] Fuzz tests for the option parsers of `printfx`, `seqx`, `trx` and `cutx`

## 5. Language and website

- [ ] Shell language growth (`if`, `for`, functions, `$(...)`): a deliberate decision first,
  since the shell is a small portable subset by design
- [x] `llms.txt` and `llms-full.txt` on the website (`web/src/app/llms*.txt`)
- [ ] A plain markdown copy of every documentation page (`/docs/<page>.md`) for agents
