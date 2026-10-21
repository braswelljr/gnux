# Agents hand over the git commands, and the maintainer runs them

An agent working in this repository never commits, tags, pushes, force-pushes, deletes a tag, or creates or deletes a release. It prepares the change in the working tree, checks it, and then hands the maintainer the exact commands to run and the commit or tag message to use. The maintainer runs them.

## Why

A commit is the maintainer's record of the project, and a push or a tag is published the moment it lands: a `v*` tag starts the Release workflow, which publishes a GitHub release, commits a Homebrew cask and a Scoop manifest to `main`, moves the major tag and reaches package managers within minutes. None of that can be taken back cleanly. A deleted release has already been downloaded, the Go module proxy may already hold the tag for good, and a force-push rewrites history other clones depend on. Releasing v1.0.0 from an agent session showed all of it: the release had to be deleted, both tags removed and `main` rewound by hand. Handing over the commands keeps every one of those steps a decision the maintainer makes with their own hands.

## What it looks like

At the end of a change the agent reports what it changed and verified, then ends with a block the maintainer can paste, for example:

```bash
git add .goreleaser.yaml CHANGELOG.md web/content/docs/changelog.mdx
git commit -m "release: v1.0.0" -m "The first stable release: ..."
git push origin main
git tag -a v1.0.0 -m "gnux v1.0.0"
git push origin v1.0.0
```

with a note on anything that is hard to reverse (a tag starts a release; a force-push rewrites `main`) and on what each command needs first.

Do: run the checks (`make ci`, `make docs`, `make release/check`), read `git status` and `git diff`, and write the message in the project's style (`release: vX.Y.Z`, `fix: ...`, `feat: ...`).

Do not: run `git commit`, `git push`, `git tag`, `git reset` on a pushed branch, `gh release create` or `gh release delete`, even when asked to "release it". Asked to release, the agent prepares the release and hands over the commands.

## How it is checked

Review. Nothing in the repository can stop an agent from running git; this file, its line in `AGENTS.md`, and the agent's own memory are what carry it.

## Related

- [Publishing a release](../workflows/publishing-a-release.md), whose steps the handed-over commands follow.
- [The reference documents itself, and it is checked](the-reference-documents-itself-and-it-is-checked.md): why the generated docs are not part of the commit.
