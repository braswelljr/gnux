# The reference documents itself, and it is checked

The reference for every tool, in every format, is generated from the program's own definitions: the registry and each tool's `Meta()` and flags. Nothing generated is edited by hand or committed: it is built whenever it is needed, the output is deterministic, and CI proves it still generates.

## Why

A hand-written reference drifts from the code the first time someone adds a flag and forgets the page. Generating it from the same `Spec` that builds the command means the help text, the man page, the markdown and the website cannot disagree. Generating it at the point of use, rather than committing it, means there is no copy to fall behind: the website builds its pages from the code it ships with, and a release its man pages from the tagged code. Committing four formats plus the website's MDX of the same content only repeated it, and a change to one tool touched dozens of files. Determinism (no timestamps, a pinned man date) keeps two builds of the same code identical.

## What it looks like

A reference page is changed by changing the tool. From [`cmd/rmx/interactive/interactive.go`](../../cmd/rmx/interactive/interactive.go), the metadata that becomes the `rmx` interactive flag page in markdown, man, YAML, reST and MDX:

```go
return docmeta.Meta{
	Name:  "interactive",
	Use:   "interactive [FILE]...",
	Short: "Prompt before removals: -i (every file), -I (once), --interactive[=WHEN].",
	...
```

`make docs` (or `go generate ./...`) runs [`tools/gendocs`](../../tools/gendocs) twice: once for `documentation/` in all four formats, once for `web/content/docs` as MDX. Each format directory is rebuilt from scratch. The website build (`web/scripts/gen.mjs`) does the MDX run itself, and GoReleaser generates the man pages it ships.

Generated, never edited or committed (`.gitignore`): `documentation/markdown/`, `documentation/man/`, `documentation/rest/`, `documentation/yaml/`, `web/content/docs/` except the release snapshots `web/content/docs/vX.Y`, which [`tools/docsnap`](../../tools/docsnap) writes and the Release workflow commits. Hand-written and edited like code: `documentation/guides/`, which sits beside the generated formats and which the generator publishes on the website, taking each page's title from its first heading and its description from its first paragraph; and `docs/requirements/`, `docs/rules/`, `docs/workflows/`, which stay in the repository.

Do: edit `Meta()` or the flag's usage string, then run `make docs` to read the result. In a hand-written document, link to a reference page by its website URL (`https://gnux.vercel.app/docs/tools/rmx`): it works on GitHub, and the MDX generator turns it into a path on the site.

Do not: edit `documentation/markdown/rmx/index.md` or anything under `web/content/docs/`; the next run discards it, and it is not in the repository to begin with. Do not link a hand-written document into `documentation/markdown/`: that folder exists only where someone ran `make docs`.

The generator is a tool dependency only: `cmd/docs` pulls in `cobra/doc` and `go-md2man`, and only `tools/gendocs` imports it, so neither links into the `gnux` binary.

## How it is checked

- The [Docs workflow](../../.github/workflows/docs.yml) tests the generator, runs `go generate ./...` and keeps the markdown and man reference as a run artifact. `make docs` (part of `make ci`) generates it locally, the Release workflow at the tag, and the Web workflow before it builds the site.
- `TestGenerateAll`, `TestMarkdownLinksResolve`, `TestGenerateAllDeterministic`, `TestGenerateMDX` and `TestHandwrittenPages` in [`cmd/docs/docs_test.go`](../../cmd/docs/docs_test.go).
- `TestRegistryIsConsistent` requires `Short`, `Long` and an example for every tool.

## Related

- BRULE-8 in [business requirements](../requirements/business-requirements.md); FR-39 to FR-46, FR-82, FR-83 in [functional requirements](../requirements/functional-requirements.md); NFR-29, NFR-40 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [There is one list of tools](there-is-one-list-of-tools.md)
- [Workflows](../workflows/README.md), in particular documentation generation.
