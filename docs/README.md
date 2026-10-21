# Documentation

The hand-written documentation of gnux: how to use it, what it must do, the rules the code
follows and how its processes run. The reference for every tool is generated, on the website at
[gnux.vercel.app/docs](https://gnux.vercel.app/docs) and into `../documentation/` by `make docs`.
This page is the way in to both.

Two entry points live at the repository root: [`../README.md`](../README.md), the short tour of
installing and using gnux, and [`../AGENTS.md`](../AGENTS.md), the architecture, the tool
contract and the conventions the code follows. The long-form usage documentation, with recipes
and complete examples, is [`../documentation/guides/`](../documentation/guides/README.md).

## How the documentation is arranged

Two kinds of document, and the difference decides how each is changed: `docs/` (this
directory) is written by hand for contributors; `documentation/` is what is published, the
hand-written guides in `documentation/guides/` beside the reference `make docs` generates into one
folder per format, which is not committed.

| Directory                                                   | Kind         | What belongs in it                                                              |
|-------------------------------------------------------------|--------------|---------------------------------------------------------------------------------|
| [`../documentation/guides/`](../documentation/guides/README.md)        | Hand-written | How to use gnux: getting started, recipes, scripting, Windows, CI, the library  |
| [`requirements/`](requirements/)                            | Hand-written | What gnux must do and how well: the SRS and the requirement registers under it  |
| [`rules/`](rules/README.md)                                 | Hand-written | The cross-cutting rules, one file each; `AGENTS.md` indexes them inline         |
| [`workflows/`](workflows/README.md)                         | Hand-written | How a process actually runs, step by step, with a diagram where it helps        |
| `../documentation/markdown/`                                | Generated    | The reference for gnux, the guide, every tool and the shell, as markdown        |
| `../documentation/man/`                                     | Generated    | The same reference as section 1 man pages                                       |
| `../documentation/rest/`                                    | Generated    | The same reference as reStructuredText                                          |
| `../documentation/yaml/`                                    | Generated    | The same reference as YAML, for tooling                                         |

**Generated directories are never committed or edited by hand.** `make docs` (or `go generate
./...`) rebuilds each of the four format directories from scratch, from the tool registry in
`cmd/registry.go` and each tool's `Meta()`; `.gitignore` keeps them out of the repository, and
the website build and the release generate their own copies. To change a reference page, change
the tool's metadata or flags. The rule is
[the reference documents itself, and it is checked](rules/the-reference-documents-itself-and-it-is-checked.md).

**Hand-written directories are edited like code.** Only the guides are published on the website:
the generator reads it, takes each page's title from its first heading and its description from
its first paragraph, and writes it into `web/content/docs/guides/` beside the generated pages.
`requirements/`, `workflows/` and `rules/` are for contributors and live in the repository only;
a guide's link to one of them becomes a link to its page on GitHub. Every file here still starts
with one `# Title` and a one-paragraph summary and uses only standard markdown, so any folder can
be published by adding it to `handwrittenFolders` in `tools/gendocs/main.go`.

File names are lower-case and hyphenated throughout, `README.md` excepted. A document is named
for its subject, never for its status or its date.

## Start here

| If you are                                  | Read                                                                                              |
|---------------------------------------------|---------------------------------------------------------------------------------------------------|
| New to gnux                                 | [guides/getting-started](../documentation/guides/getting-started.md), then [guides/cookbook](../documentation/guides/cookbook.md)   |
| Using the tools                             | [`../README.md`](../README.md), then [the generated guide](https://gnux.vercel.app/docs/guide)                     |
| Adopting gnux next to GNU coreutils         | [guides/adoption](../documentation/guides/adoption.md)                                                           |
| On Windows or in PowerShell                 | [guides/windows-and-powershell](../documentation/guides/windows-and-powershell.md)                                 |
| Running gnux in CI or a container           | [guides/ci-and-containers](../documentation/guides/ci-and-containers.md)                                           |
| Something went wrong                        | [guides/troubleshooting](../documentation/guides/troubleshooting.md)                                               |
| Looking up one tool's options               | [the reference index](https://gnux.vercel.app/docs), which links to every tool page, or `gnux help TOOL`                          |
| Writing a portable script                   | [guides/portable-scripting](../documentation/guides/portable-scripting.md), then [the shell reference](https://gnux.vercel.app/docs/shell) |
| Embedding gnux in a Go program              | [guides/go-library](../documentation/guides/go-library.md), then [software-requirements-specification](requirements/software-requirements-specification.md) section 3 |
| Deciding whether a change belongs           | [business-requirements](requirements/business-requirements.md) and [functional-requirements](requirements/functional-requirements.md) |
| Adding a tool                               | [`../AGENTS.md`](../AGENTS.md), then [workflows/adding-a-tool](workflows/adding-a-tool.md)        |
| Changing engine or adapter code             | [`rules/`](rules/README.md), backend half                                                         |
| Working on the website or the playground    | [`rules/`](rules/README.md), frontend half, and [workflows/wasm-playground](workflows/wasm-playground.md) |
| Reviewing a pull request                    | [`rules/`](rules/README.md) and the verification column of the requirement registers              |
| Debugging CI                                | [workflows/ci-and-release](workflows/ci-and-release.md)                                           |
| Cutting a release                           | [workflows/publishing-a-release](workflows/publishing-a-release.md)                               |

## Guides

[`../documentation/guides/`](../documentation/guides/README.md) is the usage half of the documentation: task-oriented, example
first, every command run against a real build before it was written down. Start with
[getting-started](../documentation/guides/getting-started.md); the [cookbook](../documentation/guides/cookbook.md) is the place
to look up "how do I ..." questions.

## Requirements

| Document                                                                                   | Covers                                                                          |
|--------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------|
| [software-requirements-specification](requirements/software-requirements-specification.md) | The SRS: scope, interfaces, features, conventions and the traceability matrix   |
| [business-requirements](requirements/business-requirements.md)                             | Why gnux exists: vision, objectives, stakeholders, rules, risks                 |
| [functional-requirements](requirements/functional-requirements.md)                         | What gnux must do, including one section per tool with its required options     |
| [non-functional-requirements](requirements/non-functional-requirements.md)                 | How well it must do it: portability, safety, performance, quality               |
| [srs-workflows](requirements/srs-workflows.md)                                             | The user-facing workflows, step by step, with the requirements each exercises   |
| [checklist](requirements/checklist.md)                                                     | Every planned upgrade and integration, ticked off as it lands                   |

Requirement identifiers (`BR-n`, `FR-n`, `NFR-n`) are stable. Cite them in commits and pull
requests where it helps, and update the register in the same change that alters the behavior it
describes.

## Rules

[`rules/`](rules/README.md) has its own index: one row per rule, with the one-line statement and
the file that carries the reasoning, the examples and the check that enforces it. `AGENTS.md`
repeats each statement inline so that nothing is left to a link to discover.

## Workflows

[`workflows/`](workflows/README.md) has its own index. Three are worth naming here:

- [command-line-to-tool-run](workflows/command-line-to-tool-run.md): how `argv` becomes a tool
  run, which is the path every other workflow passes through.
- [documentation-generation](workflows/documentation-generation.md): how the registry becomes
  the four reference formats and the website content, and how CI keeps them fresh.
- [ci-and-release](workflows/ci-and-release.md): each GitHub workflow, the requirement it
  enforces, and what a release consists of today.

## The generated reference

The reference lives in `documentation/` at the repository root, one directory per format
(`markdown`, `man`, `rest`, `yaml`). Each is self-contained and has the same layout:

| Path                                  | Page                                                                 |
|---------------------------------------|----------------------------------------------------------------------|
| `<format>/index.*`                    | gnux itself: the entry page, linking to every tool                   |
| `<format>/guide.*`                    | Requirements, installation and usage                                 |
| `<format>/<tool>/index.*`             | One page per tool: description, permissions, use cases, flags, examples |
| `<format>/<tool>/flags/<flag>.*`      | One page per flag, for tools that document them individually (`rmx`) |
| `<format>/shell/index.*`              | The gnux shell                                                       |

The same pages are printed in the terminal by `gnux help TOOL`, and the website's copy lives in
`web/content/docs/`, generated too before every website build, with the docs of each release
downloaded from the release itself. None of it is committed. A guide links to a reference page by its website
URL (`https://gnux.vercel.app/docs/tools/rmx`), which works on GitHub and becomes a path on the
site itself.
