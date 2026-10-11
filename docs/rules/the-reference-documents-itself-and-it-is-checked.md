# The reference documents itself, and it is checked

The reference for every tool, in every format, is generated from the program's own definitions: the registry and each tool's `Meta()` and flags. Nothing generated is edited by hand, the output is deterministic, and CI fails a change whose committed copies differ from what the generator produces.

## Why

A hand-written reference drifts from the code the first time someone adds a flag and forgets the page. Generating it from the same `Spec` that builds the command means the help text, the man page, the markdown and the website cannot disagree. Determinism (no timestamps, a pinned man date) is what makes the freshness check possible: if the output changed on every run, the check could not tell drift from noise.

## What it looks like

A reference page is changed by changing the tool. From [`cmd/rmx/interactive/interactive.go`](../../cmd/rmx/interactive/interactive.go), the metadata that becomes the `rmx` interactive flag page in markdown, man, YAML, reST and MDX:

```go
return docmeta.Meta{
	Name:  "interactive",
	Use:   "interactive [FILE]...",
	Short: "Prompt before removals: -i (every file), -I (once), --interactive[=WHEN].",
	...
```

`make docs` (or `go generate ./...`) runs [`tools/gendocs`](../../tools/gendocs) twice: once for `docs/` in all four formats, once for `web/content/docs` as MDX. Each format directory is rebuilt from scratch.

Generated, never edited: `documentation/markdown/`, `documentation/man/`, `documentation/rest/`, `documentation/yaml/`, `web/content/docs/`. Hand-written and edited like code: `documentation/guides/`, which sits beside the generated formats and which the generator publishes on the website, taking each page's title from its first heading and its description from its first paragraph; and `docs/requirements/`, `docs/rules/`, `docs/workflows/`, which stay in the repository.

Do: edit `Meta()` or the flag's usage string, then run `make docs` and commit the result.

Do not: edit `documentation/markdown/rmx/index.md` or anything under `web/content/docs/`; the next run discards it, and the Docs workflow rejects it first.

The generator is a tool dependency only: `cmd/docs` pulls in `cobra/doc` and `go-md2man`, and only `tools/gendocs` imports it, so neither links into the `gnux` binary.

## How it is checked

- The [Docs workflow](../../.github/workflows/docs.yml) runs `go generate ./...` and fails if `git status --porcelain docs web/content/docs` is not empty. `make docs/check` (part of `make ci`) does the same locally, and the Release workflow repeats it at the tag.
- `TestGenerateAll`, `TestMarkdownLinksResolve`, `TestGenerateAllDeterministic`, `TestGenerateMDX` and `TestHandwrittenPages` in [`cmd/docs/docs_test.go`](../../cmd/docs/docs_test.go).
- `TestRegistryIsConsistent` requires `Short`, `Long` and an example for every tool.

## Related

- BRULE-8 in [business requirements](../requirements/business-requirements.md); FR-39 to FR-46, FR-82, FR-83 in [functional requirements](../requirements/functional-requirements.md); NFR-29, NFR-40 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [There is one list of tools](there-is-one-list-of-tools.md)
- [Workflows](../workflows/README.md), in particular documentation generation.
