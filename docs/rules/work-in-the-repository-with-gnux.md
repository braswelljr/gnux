# Work in the repository with gnux

Anyone working in this repository, an agent above all, does the everyday command-line work with gnux itself: `rmx` rather than `rm`, `cpx`, `mvx`, `mkdirx`, `lsx`, `catx`, `headx`, `grepx`, `sedx`, `findx`, `xargsx`, `wcx`, `sortx` and the rest, run from `./bin` after `make build` (or through `make links`, which puts every name beside the binary). A tool that does something GNU would not is a bug to fix in the same change, with a test, not a reason to reach for the system's own tool.

## Why

gnux promises to behave as GNU coreutils, grep, sed and findutils do. The tests and the conformance harness check the behavior someone thought to write down; daily use finds the rest. Searching the code with `grepx`, editing with `sedx -i`, cleaning up with `rmx -rf` and scripting with `gnux shell` puts the tools through the cases real work produces, every day, and the people who find the differences are the ones who can fix them.

This rule found its first bugs while it was being written: `gnux shell` turned `2>&1` into a file named `&1`, and `${#x}` expanded to nothing (both fixed in 1.4.0).

## What it looks like

```bash
make build
./bin/gnux grepx -rn 'ErrFailed' core | ./bin/gnux headx -20
./bin/gnux findx core -name '*_test.go' -newer go.mod
./bin/gnux sedx -i 's/old/new/' docs/README.md
./bin/gnux rmx -rf dist
./bin/gnux shell -c 'for f in core/*/; do echox ${f%/}; done'
```

Do: rebuild before relying on a change; when a tool misbehaves, compare with GNU (`make conformance/docker` has the real tools), fix it, add a test and a `CHANGELOG.md` line.

Do not: switch to the system tool and move on. A step gnux cannot do yet (`git`, `go`, `pnpm`, network tools) is not a coreutils job and runs as usual.

## How it is checked

Review, and the agents' memory. Nothing can stop a shell from running `/bin/rm`; this file and its line in `AGENTS.md` are what carry it.

## Related

- [GNU is the specification, and every difference is documented](gnu-is-the-specification-and-differences-are-documented.md): what a bug found this way is measured against.
- [GNU conformance](../workflows/gnu-conformance.md): how to check a difference against the real tools.
- [Agents hand over the git commands](agents-hand-over-git-commands.md): the one kind of command this rule does not touch.
