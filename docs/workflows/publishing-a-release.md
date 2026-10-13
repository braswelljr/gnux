# Publishing a Release

How a gnux version gets from a commit on `main` to every place a user installs it from: the GitHub release, the `.deb`, `.rpm` and `.apk` packages, Homebrew, Scoop, winget, the Snap Store, the AUR, `go install` and the setup-gnux action. Pushing a `vX.Y.Z` tag does all of it, and Homebrew and Scoop are served from this repository with nothing to set up, but winget, the Snap Store and the AUR each first need something set up once, outside it: a fork, an account, a registered name, a secret. This page is that setup, channel by channel, then the release itself step by step, how to check each channel afterwards, how a user installs one particular version from each, and what to do when a release goes wrong. What the Release workflow runs, job by job, is in [ci-and-release](ci-and-release.md); what a release contains is written once, in `.goreleaser.yaml`.

## Where a release goes

| Channel            | A user types                                      | Published by                         | Needs, once                                                         |
|--------------------|---------------------------------------------------|--------------------------------------|---------------------------------------------------------------------|
| GitHub release     | downloads `gnux-<tag>-<os>-<arch>.tar.gz` / `.zip` | GoReleaser                           | Nothing                                                             |
| Linux packages     | `sudo dpkg -i`, `sudo rpm -i`, `sudo apk add` on a file from the release | GoReleaser (`nfpms`)  | Nothing                                                             |
| `go install`       | `go install github.com/braswelljr/gnux@v0.0.2`    | The tag itself                       | Nothing                                                             |
| setup-gnux action  | `uses: braswelljr/gnux@v0`                        | The workflow moves the `v0` tag      | Nothing                                                             |
| Homebrew (own tap) | `brew tap braswelljr/gnux https://github.com/braswelljr/gnux`, then `brew install --cask braswelljr/gnux/gnux` | GoReleaser (`homebrew_casks`), into `Casks/` here | Nothing |
| Scoop              | `scoop bucket add gnux https://github.com/braswelljr/gnux`, then `scoop install gnux/gnux` | GoReleaser (`scoops`), into `bucket/` here | Nothing |
| winget             | `winget install braswelljr.gnux`                  | GoReleaser (`winget`), then Microsoft's review | A fork of `microsoft/winget-pkgs`, `WINGET_GITHUB_TOKEN`    |
| Snap Store         | `sudo snap install gnux`                          | GoReleaser (`snapcrafts`)            | An Ubuntu One account, the name `gnux` registered, `SNAPCRAFT_STORE_CREDENTIALS` |
| AUR                | `yay -S gnux-bin`                                 | GoReleaser (`aurs`)                  | An AUR account with an SSH key, `AUR_KEY`                           |
| Homebrew core      | `brew install gnux`                               | A pull request to `Homebrew/homebrew-core`, by hand | Homebrew's acceptance; see [Homebrew core](#homebrew-core-a-formula) |

**A channel whose secret is not set is skipped, not failed.** The `Choose what to publish` step of the Release workflow adds it to GoReleaser's `--skip`, so the first six rows work today and each of the others starts working at the first release after its secret exists. Nothing has to be set up all at once.

**A prerelease stays out of the package managers.** A tag with a hyphen (`v0.0.2-rc.1`) gets a GitHub prerelease with every archive and package, and a snap on the `edge` channel; Homebrew, Scoop, winget and the AUR are left alone (`skip_upload: auto`), and `v0` does not move.

## One-time setup

Every command below is run once, by the repository's owner. `gh` is the [GitHub CLI](https://cli.github.com/), signed in as `braswelljr`.

### Homebrew tap and Scoop bucket

Nothing to set up. The gnux repository is its own tap and its own bucket: on each stable release, GoReleaser commits `Casks/gnux.rb` and `bucket/gnux.json` to `main` with the release job's token, which already has `contents: write` here. Those two files are generated; do not edit them by hand.

What follows from that:

- **Two commits appear on `main` after every release**, `release: Homebrew cask for v0.0.2` and `release: Scoop manifest for v0.0.2`, authored by the workflow. They come after the tag, so `git pull` before the next piece of work. A push made with the workflow's token does not start other workflows, so they run no CI.
- **`main` must accept that push.** If a branch protection rule or ruleset ever requires pull requests on `main`, the two commits are refused with a 403; add the GitHub Actions app to the rule's bypass list, or move to separate repositories as described below.
- **Users name the repository when they add it**, because it is not called `homebrew-tap`: `brew tap braswelljr/gnux https://github.com/braswelljr/gnux`, and `scoop bucket add gnux https://github.com/braswelljr/gnux`. Both clone this whole repository once.

The cask installs the binary and the bash, zsh and fish completions, and clears the quarantine flag macOS puts on a download, because the binary is not notarized. It works on Linux with Homebrew too.

**Moving to repositories of their own later** gives the shorter `brew install --cask braswelljr/tap/gnux`: create `braswelljr/homebrew-tap` and `braswelljr/scoop-bucket`, each with one commit; create a fine-grained token with Contents read and write on those two only and store it as a secret; then in `.goreleaser.yaml` point each `repository` at its new name and that token, drop `directory: Casks` for the tap, and pass the secret to the `Build and publish` step.

### winget

winget has no personal repositories: every package is a set of manifests in `microsoft/winget-pkgs`, added by pull request. GoReleaser writes the manifests to a branch of your fork and opens the pull request.

1. Fork the repository:

   ```bash
   gh repo fork microsoft/winget-pkgs --clone=false
   ```

2. Create a **classic** token (Settings, Developer settings, Tokens (classic)) with the `public_repo` scope. A fine-grained token cannot open a pull request against a repository it does not own, which is why this one is classic.

3. Store it:

   ```bash
   gh secret set WINGET_GITHUB_TOKEN --repo braswelljr/gnux
   ```

What to expect from the first release:

- A pull request titled `New version: braswelljr.gnux 0.0.2` appears on `microsoft/winget-pkgs`. A bot asks you to sign the Microsoft CLA once; comment as it instructs.
- Automated validation installs the package in a sandbox and scans it. A new package is then reviewed by a moderator, which takes from hours to a few days. Later versions of an accepted package usually merge without a person.
- `winget install braswelljr.gnux` works once the pull request is merged and the index refreshes, not when the release finishes.
- Keep the fork's `master` reasonably fresh (`gh repo sync braswelljr/winget-pkgs`); a fork far behind makes the branch slow to create.

The package is a portable one: winget unpacks `gnux.exe` and puts it on `PATH` through its links directory. The identifier `braswelljr.gnux` is permanent once accepted.

### Snap Store

1. Create an [Ubuntu One](https://login.ubuntu.com/) account, then, on a Linux machine or in a container with `snapcraft` installed (`sudo snap install snapcraft --classic`):

   ```bash
   snapcraft login
   snapcraft register gnux
   ```

   Registration is first come, first served, and is the step most likely to need a second try: if `gnux` is taken, register another name and change `name:` under `snapcrafts` in `.goreleaser.yaml`.

2. Export a credential that can do nothing but publish this snap:

   ```bash
   snapcraft export-login --snaps=gnux \
     --acls=package_access,package_push,package_update,package_release \
     --channels=stable,edge \
     credentials.txt
   gh secret set SNAPCRAFT_STORE_CREDENTIALS --repo braswelljr/gnux < credentials.txt
   rm credentials.txt
   ```

   The credential expires after a year by default (`--expires` sets another date). Export a new one before then.

3. On [snapcraft.io/gnux/listing](https://snapcraft.io/gnux/listing), after the first upload, add an icon, the website and the source link. The summary and description come from `.goreleaser.yaml` on each release.

When the secret is set, the Release workflow installs `snapcraft`, and GoReleaser packs one snap per Linux architecture (amd64, arm64, riscv64) and releases a stable version to the `stable` channel.

**Confinement.** The snap is `strict`, which the store accepts with no review. It means the tools see the user's home directory (not its hidden files) and, after `sudo snap connect gnux:removable-media`, `/media` and `/mnt`; they do not see `/etc`, `/usr` or `/tmp` of the host. That is enough for working on a project and too little for system administration. A `classic` snap sees everything, but the store holds every upload of one until a reviewer has granted the snap classic confinement; ask on the [snapcraft forum](https://forum.snapcraft.io/c/store-requests), in the store-requests category, explaining that gnux is a set of file tools that must reach any path. If it is granted, change `confinement: strict` to `classic` and remove `plugs`.

**Tool names.** A snap exposes only `gnux`, so the tools are run as `gnux rmx ...`. Exposing `rmx`, `cpx` and the rest as commands of their own would need an alias for each, and aliases are granted by the same forum process.

### AUR

`gnux-bin` repackages the release archive for Arch Linux (x86_64 and aarch64). An AUR package is a git repository on `aur.archlinux.org` that you push to over SSH; it is created by the first push.

1. Create an account at [aur.archlinux.org](https://aur.archlinux.org/register), and check that `gnux-bin` is free.

2. Make a key pair used for nothing else, with no passphrase, since a workflow has to use it:

   ```bash
   ssh-keygen -t ed25519 -C "gnux release" -f aur_key -N ""
   ```

3. Paste `aur_key.pub` into **SSH Public Key** under My Account on the AUR.

4. Store the private half, then delete both files:

   ```bash
   gh secret set AUR_KEY --repo braswelljr/gnux < aur_key
   rm aur_key aur_key.pub
   ```

The package installs `/usr/bin/gnux`, the license and the three completions. It `provides` and `conflicts` with `gnux`, so a from-source `gnux` package can replace it later.

### Linux packages

Nothing to set up. Every release carries `gnux-<tag>-linux-<arch>.deb`, `.rpm` and `.apk` for amd64, arm64 and riscv64, listed in `checksums.txt` and covered by the provenance attestation. Each installs `/usr/bin/gnux`, the license and the bash, zsh and fish completions.

They are files on the release, not a repository: `apt upgrade` does not know about a newer one. Hosting an apt or yum repository is a separate decision.

### The GitHub Marketplace listing

Optional. The setup-gnux action works as `uses: braswelljr/gnux@v0` without it. To list it, open the release on GitHub after it is published, choose **Edit**, tick **Publish this Action to the GitHub Marketplace**, and accept the agreement the first time. `action.yml` already has the name, description and branding the listing uses.

### Homebrew core: a formula

The tap above is a *cask* because GoReleaser publishes prebuilt binaries, and a cask is what Homebrew uses for those. Homebrew's own guidance is that open source command-line software Homebrew can build from source belongs in a *formula*, and gnux is exactly that. A formula in `homebrew/core` is what makes `brew install gnux` work with no tap, with bottles built by Homebrew for each platform.

It is a manual submission, done once, and then Homebrew's bots bump the version on each new tag. Before starting:

- Read [Acceptable Formulae](https://docs.brew.sh/Acceptable-Formulae). A new formula has to be notable enough (the page gives the thresholds for stars, forks and watchers, and they are higher for a project submitted by its own author), have a stable tagged release, and build from source. A project that is a few weeks old is usually turned away; that is a reason to wait, not a problem with the formula.
- Search the open and closed pull requests of `Homebrew/homebrew-core` for `gnux`. An earlier rejection says what has to change first.

Then, following [How to Open a Homebrew Pull Request](https://docs.brew.sh/How-To-Open-a-Homebrew-Pull-Request):

1. Generate a starting point from the tag's source archive:

   ```bash
   brew tap --force homebrew/core
   brew create --go --tap homebrew/core \
     https://github.com/braswelljr/gnux/archive/refs/tags/v0.0.2.tar.gz
   ```

2. Replace what it generated with the formula. This is the shape to end up with; `brew create` fills in the `sha256`:

   ```ruby
   class Gnux < Formula
     desc "GNU core utilities in one portable, multi-call binary"
     homepage "https://github.com/braswelljr/gnux"
     url "https://github.com/braswelljr/gnux/archive/refs/tags/v0.0.2.tar.gz"
     sha256 "<filled in by brew create>"
     license "MIT"
     head "https://github.com/braswelljr/gnux.git", branch: "main"

     depends_on "go" => :build

     def install
       ldflags = "-s -w -X github.com/braswelljr/gnux/internal/common.Version=v#{version}"
       system "go", "build", *std_go_args(ldflags:)
       generate_completions_from_executable(bin/"gnux", "completion")
     end

     test do
       assert_match "rmx", shell_output("#{bin}/gnux --list")
       (testpath/"a.txt").write "one\ntwo\n"
       assert_equal "2", shell_output("#{bin}/gnux wcx -l < a.txt").strip
     end
   end
   ```

   The test has to exercise the program, not only print its version; Homebrew's reviewers ask for that.

3. Test it the way the reviewers' CI will:

   ```bash
   HOMEBREW_NO_INSTALL_FROM_API=1 brew install --build-from-source gnux
   brew test gnux
   brew audit --strict --new --online gnux
   brew style --fix --formula gnux
   ```

   Look at what was installed (`brew list gnux`) and at the output, not only at the exit statuses.

4. Commit as `gnux 0.0.2 (new formula)`, push to your fork and open the pull request, filling in its template.

Once it is merged, the README and the install page can say `brew install gnux`. The tap's cask can stay for people who already use it; a cask and a formula of the same name do not conflict, because one is installed with `--cask`.

## The secrets, in one table

| Secret                        | Channel   | What it is                                                         | Expires                    |
|-------------------------------|-----------|--------------------------------------------------------------------|----------------------------|
| `WINGET_GITHUB_TOKEN`         | winget    | Classic token with `public_repo`                                   | When you set it to         |
| `SNAPCRAFT_STORE_CREDENTIALS` | Snap Store | Output of `snapcraft export-login`, limited to the `gnux` snap    | After a year, by default   |
| `AUR_KEY`                     | AUR       | An ed25519 private key whose public half is on the AUR account     | Never                      |

`gh secret list --repo braswelljr/gnux` shows which are set. No secret is ever read by a pull request's workflows: only the `release` job of the Release workflow receives them, and it runs only on a pushed tag.

## Cutting the release

The example is `v0.0.2`. The same steps apply to every version.

### 1. Get the commit onto `main`, green

A release is cut from `main`. Merge the work, then check that Test, Build, Lint, Docs and Web passed on the merge commit:

```bash
git switch main && git pull
gh run list --branch main --limit 8
```

The Release workflow repeats vet, the tests and the docs check on Linux, but not Lint, the Windows and macOS test legs or Web; those have to be green here.

### 2. Finish the changelog

`CHANGELOG.md` gets a section for the version, with today's date, and an empty `## [Unreleased]` above it:

```markdown
## [Unreleased]

## [0.0.2] - 2026-10-13
```

Check the link definitions at the bottom (`[Unreleased]` compares `v0.0.2...HEAD`, `[0.0.2]` compares `v0.0.1...v0.0.2`), then read the notes exactly as the release will show them:

```bash
go run ./tools/changelog v0.0.2
```

A release tag with no section, or an empty one, stops the workflow before anything is built.

### 3. Rehearse

```bash
make docs                # /docs/changelog on the website is generated from CHANGELOG.md
make ci                  # vet, lint, test and the docs freshness check
make release/check       # .goreleaser.yaml is valid
make release/snapshot    # everything a release builds, into dist/, nothing published
```

`make release/snapshot` needs [GoReleaser](https://goreleaser.com/install/) v2, and builds the snaps only when `snapcraft` is installed. Look at what it made:

```bash
ls dist/*.tar.gz dist/*.zip dist/*.deb dist/*.rpm dist/*.apk
cat dist/homebrew/Casks/gnux.rb dist/scoop/gnux.json dist/aur/gnux-bin.pkgbuild
ls dist/winget/manifests/b/braswelljr/gnux/*/
```

Those manifests are what would be pushed. To rehearse on GitHub as well, push a prerelease tag first (`v0.0.2-rc.1`): it runs the whole workflow, publishes a prerelease and touches no package manager except the snap's `edge` channel. Delete the prerelease afterwards if you do not want to keep it.

### 4. Commit, tag, push

```bash
git commit -am "release: v0.0.2"
git push origin main
git tag -a v0.0.2 -m "gnux v0.0.2"
git push origin v0.0.2
```

The tag must be annotated (`-a`) and a full version: `v0.0.2`, not `v0.0` or `0.0.2`. Pushing the tag is the release; there is no button to press.

### 5. Watch it

```bash
gh run watch "$(gh run list --workflow release.yml --limit 1 --json databaseId -q '.[0].databaseId')"
```

`verify` takes a few minutes, `release` a few more. The log of `Choose what to publish` shows the `--skip` list, which is the quickest way to see which channels this release went to.

## After the release

Check each channel that was not skipped. Every line should end with the new version.

| Channel          | Check                                                                                         | Ready                              |
|------------------|-----------------------------------------------------------------------------------------------|------------------------------------|
| GitHub release   | `gh release view v0.0.2` lists 9 archives, 9 packages, the snaps if built, and `checksums.txt` | At once                           |
| Attestation      | `gh attestation verify gnux-v0.0.2-linux-amd64.tar.gz --repo braswelljr/gnux`                 | At once                            |
| `go install`     | `go install github.com/braswelljr/gnux@v0.0.2 && gnux --version`                              | Within minutes (the module proxy)  |
| setup-gnux       | `git ls-remote origin refs/tags/v0 refs/tags/v0.0.2^{}` shows the same commit                 | At once                            |
| Homebrew         | `brew update && brew info --cask braswelljr/gnux/gnux`                                         | At once                            |
| Scoop            | `scoop update; scoop info gnux/gnux`                                                    | At once                            |
| winget           | The pull request on `microsoft/winget-pkgs`, then `winget show braswelljr.gnux`               | After review: hours to days        |
| Snap Store       | `snapcraft status gnux`, or `snap info gnux`                                                  | Minutes, after the automatic review |
| AUR              | `https://aur.archlinux.org/packages/gnux-bin`                                                 | At once                            |
| Linux package    | `sudo dpkg -i gnux-v0.0.2-linux-amd64.deb && gnux --version`                                  | At once                            |
| Website          | `/docs/changelog` shows the section                                                           | After the site deploys             |

Then update the two places that name a version in an example, so that copied commands fetch the release just made: the `v=` line under Prebuilt binaries in `README.md` and the one in `web/src/config/install.ts`.

## Installing one particular version

Users pin a version for reproducible builds; this is what each channel offers, and what the install documentation says. `gnux --version` (or `rmx --version`) reports the version installed, whichever way it arrived.

| Channel          | A particular version                                                    | Notes                                                        |
|------------------|-------------------------------------------------------------------------|--------------------------------------------------------------|
| Release archive  | Download `gnux-v0.0.2-<os>-<arch>.tar.gz` and `checksums.txt` from the `v0.0.2` release | Every version stays available                |
| `go install`     | `go install github.com/braswelljr/gnux@v0.0.2`                          | Reports `v0.0.2` through the module's build info             |
| setup-gnux       | `with: { version: v0.0.2 }`                                             | `latest` is the default                                      |
| winget           | `winget install braswelljr.gnux --version 0.0.2`                        | No `v`; every accepted version stays in the index            |
| Scoop            | `scoop install gnux/gnux@0.0.2`                                   | Scoop rebuilds the manifest from the bucket's history; `scoop hold gnux` stops upgrades |
| `.deb` / `.rpm` / `.apk` | Download the file from the version's release                    | `sudo apt-mark hold gnux` if a repository ever carries it    |
| Snap             | `sudo snap install gnux --channel=edge` for a prerelease; `sudo snap refresh --hold gnux` stays on the installed revision | A channel holds one revision; older ones are not installable by version |
| Homebrew         | The newest only. `brew pin` does not apply to casks                     | For an older version, use the archive or `go install`        |
| AUR              | The newest only                                                         | An older `PKGBUILD` is in the package's git history          |
| Docker           | `docker build --build-arg VERSION=v0.0.2 -t gnux:v0.0.2 .` on a checkout of the tag |                                                  |

## When a release goes wrong

**Before anything was published** (`verify` failed, or GoReleaser failed to build): nothing left the repository. Fix it on `main`, then move the tag:

```bash
git push --delete origin v0.0.2 && git tag -d v0.0.2
git tag -a v0.0.2 -m "gnux v0.0.2" && git push origin v0.0.2
```

**After the GitHub release exists but a package manager failed** (an expired credential, a store that was down): fix the secret and re-run the failed `release` job from the Actions page. The release is completed in place (`mode: replace`, `replace_existing_artifacts`), and the archives are rebuilt from the same commit.

**After users could have installed it**: do not move the tag. The Go module proxy has recorded `v0.0.2` permanently, a snap revision cannot be replaced, winget's manifest carries the archive's checksum, and anyone who verified `checksums.txt` would see it change. Fix the problem and release `v0.0.3`. If the version is harmful, mark the GitHub release as a prerelease or delete it, and for Go add a `retract` directive to `go.mod` in the next version.

| Symptom                                                   | Cause                                                             | Fix                                                             |
|-----------------------------------------------------------|-------------------------------------------------------------------|-----------------------------------------------------------------|
| A channel is missing and the job is green                 | Its secret is not set, so it was in `--skip`                      | Set the secret; the next release publishes it                   |
| Homebrew or Scoop: `403` on push                          | `main` is protected against direct pushes                         | Let the release job bypass the rule, or move to repositories of their own (below) |
| GoReleaser: `git is in a dirty state`                     | A step wrote a file into the checkout before GoReleaser ran       | Write it under `$RUNNER_TEMP` or a gitignored path              |
| winget: `422` creating the pull request                   | A fine-grained token, or no fork                                  | Use a classic token with `public_repo`; fork `winget-pkgs`      |
| winget: validation fails on the pull request              | The installer URL was not reachable yet, or a manifest field is wrong | Comment `@wingetbot run` to retry; otherwise fix the `winget` block and release again |
| snapcraft: `not logged in` or `invalid credentials`       | The exported login expired, or does not cover the channel         | Export a new one with `--channels=stable,edge`                  |
| snapcraft: the upload is held for manual review           | The snap declares something that needs approval (classic, an alias) | Ask on the forum, or return to `strict` with `home` and `removable-media` |
| AUR: `Permission denied (publickey)`                      | The public key is not on the AUR account, or `AUR_KEY` is not its private half | Add the key again; set the secret from the private file |
| `uses: braswelljr/gnux@v0` still installs the old version | The tag was a prerelease, so `v0` did not move                    | Release a stable version                                        |
| `go install ...@latest` still gets the old version        | The module proxy has not seen the tag yet                         | `GOPROXY=direct go install ...@v0.0.2` once, or wait            |

## Adding another package manager

GoReleaser has a block for most of them (`nix`, `chocolateys`, `dockers`). To add one:

1. Add its block to `.goreleaser.yaml`, with `skip_upload: auto` so that a prerelease stays out, and read its credential as `{{ .Env.NAME }}`: GoReleaser accepts nothing else in a `token` (no template function, `envOrDefault` included), and reads it only when that publisher runs, so the configuration stays valid without the secret.
2. In `.github/workflows/release.yml`, pass the secret to `Choose what to publish` and to `Build and publish`, and add the publisher's name to `skip` when the secret is empty. `goreleaser release --help` lists the names `--skip` accepts.
3. Run `make release/check` and `make release/snapshot`, and read the manifest it wrote into `dist/`.
4. Add the channel to the tables on this page, to the README's Installation section, to [getting started](../../documentation/guides/getting-started.md), to the Installation section of `cmd/docs/guide.go` and to `web/src/config/install.ts`, and to `CHANGELOG.md`.

## Related

- [ci-and-release](ci-and-release.md): the Release workflow job by job, and where the version string comes from.
- [documentation-generation](documentation-generation.md): how `CHANGELOG.md` becomes `/docs/changelog`.
- [Getting started](../../documentation/guides/getting-started.md): the same channels from the user's side.
- [GoReleaser's documentation](https://goreleaser.com/customization/) for every field of `.goreleaser.yaml`.
