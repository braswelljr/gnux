# Rules

The cross-cutting rules the gnux code follows, one file each. A rule is a convention that holds across many files, so no single file states it: how engines are shaped, how failures are reported, how the reference stays true, how the website is built. Each file gives the rule in one sentence, why it exists, what it looks like in the code, and what checks it. Where nothing automated checks a rule, the file says "review" instead of pretending otherwise. The backend half covers the Go code: engines, adapters, the shell, the documentation generator and CI. The frontend half covers `web/`: the website and the playground. The last section covers how work reaches the repository.

## Backend

| Rule | Statement | Checked by |
|------|-----------|------------|
| [gnu-is-the-specification-and-differences-are-documented](gnu-is-the-specification-and-differences-are-documented.md) | GNU's behavior is the specification; any difference is deliberate and written into the tool's reference text | Engine suites (exact output); `TestRegistryIsConsistent`; review |
| [the-zero-value-is-the-safe-default](the-zero-value-is-the-safe-default.md) | `Options{}` is the tool with no flags, every safeguard on; a default-on field is named for its negative | Engine suites; `TestBuildOptions` "defaults" cases; review |
| [engines-know-nothing-of-the-command-line](engines-know-nothing-of-the-command-line.md) | Behavior lives in `core/<gnu>/`, flags in `cmd/<gnu>x/`; `core/` never imports cobra or pflag | `TestBuildOptions`; `TestRegistryIsConsistent`; review |
| [an-engine-reads-and-writes-only-its-streams](an-engine-reads-and-writes-only-its-streams.md) | An engine uses only the `core.Streams` it was given, never `os.Stdin` / `os.Stdout` / `os.Stderr` | Engine suites on in-memory streams; review |
| [there-is-one-list-of-tools](there-is-one-list-of-tools.md) | `Tools()` is the only list of tools; the binary, docs, wasm and website all derive from it | `cmd/main_test.go`; `cmd/docs/docs_test.go`; `web/test/wasm.test.ts` |
| [diagnostics-are-written-in-gnus-words](diagnostics-are-written-in-gnus-words.md) | Diagnostics go through `core.Diag` in GNU's exact wording, with `core.ErrText` for the same text on every platform | `TestErrText`; engine suites on three OSes; `TestDiagnostics` |
| [a-failing-operand-does-not-stop-the-rest](a-failing-operand-does-not-stop-the-rest.md) | One failing operand is reported and the others are still processed; the status is 1 at the end | Per-tool engine tests (`cat`, `ln`, `touch`, `realpath`, `ls`) |
| [conflicts-resolve-toward-the-most-cautious-option](conflicts-resolve-toward-the-most-cautious-option.md) | When GNU's "last one wins" cannot be reproduced, the most cautious option wins in any order | `TestBuildOptions` in each adapter, both orders |
| [a-failure-is-never-reported-as-success](a-failure-is-never-reported-as-success.md) | Every failure gets a diagnostic and a nonzero status, exactly once; nothing quietly does something else | `TestDiagnostics`; `TestWriteErrors`; `errcheck` |
| [platform-code-lives-behind-build-tags](platform-code-lives-behind-build-tags.md) | Platform code is in build-tagged files with a fallback that compiles everywhere, including js/wasm; no cgo | Build, Test (three OSes + wasm) and Web workflows |
| [the-shell-runs-no-external-program-implicitly](the-shell-runs-no-external-program-implicitly.md) | `gnux shell` runs builtins and registered tools only; anything else is `command not found` (127) | `TestDiagnostics` in `core/shell`; `cmd/shell_test.go` |
| [the-reference-documents-itself-and-it-is-checked](the-reference-documents-itself-and-it-is-checked.md) | The reference is generated from the registry and `Meta()`, never edited by hand, and CI rejects stale copies | Docs workflow; `make docs/check`; `cmd/docs/docs_test.go` |
| [tests-drive-engines-not-terminals](tests-drive-engines-not-terminals.md) | Tests build engines on temp dirs, buffers and injected funcs; the race detector is always on | `go test -race` in the Test workflow and `make test` |

## Frontend

| Rule | Statement | Checked by |
|------|-----------|------------|
| [no-em-dashes](no-em-dashes.md) | US spelling and no em dashes, anywhere in the repository | `web/test/rules.test.ts`; `misspell` (Go); review (markdown) |
| [a-public-page-describes-itself](a-public-page-describes-itself.md) | Every page builds its metadata with `createMetadata(siteConfig, {...})`, never a `Metadata` literal | `web/test/rules.test.ts`; `pnpm test:seo` |
| [the-markup-says-what-the-thing-is](the-markup-says-what-the-thing-is.md) | Semantic elements and landmarks before ARIA, an accessible name on every control, one `h1`, no meaning by color alone | Review |
| [the-shortest-path-and-only-one-of-them](the-shortest-path-and-only-one-of-them.md) | One obvious route to each thing in the fewest steps, one primary action per view, no duplicate controls, a reason beside anything disabled | Review |
| [build-the-ui-from-atoms-upward](build-the-ui-from-atoms-upward.md) | Compose from `src/components/ui` upward; pages compose organisms and do not style from scratch | Review |
| [a-dark-step-mirrors-its-light-one](a-dark-step-mirrors-its-light-one.md) | `fd-*` tokens first; otherwise every light color step has its mirrored `dark:` step | Review |
| [type-sizes-come-from-the-scale](type-sizes-come-from-the-scale.md) | Text sizes are `text-xs` through `text-3xl`, never `text-[13px]` | `web/test/rules.test.ts` |
| [an-icon-only-button-takes-an-icon-size](an-icon-only-button-takes-an-icon-size.md) | An icon-only button uses `size="icon"` or `"icon-sm"`, an `aria-label`, and an `aria-hidden` icon | Review |
| [icons-come-from-the-four-installed-sets](icons-come-from-the-four-installed-sets.md) | Icons come from `lucide-react` alone | `web/test/rules.test.ts` |
| [state-a-reader-can-arrive-at-lives-in-the-url](state-a-reader-can-arrive-at-lives-in-the-url.md) | Shareable state lives in the URL through `nuqs`, like the playground's `?cmd=` | Review |
| [data-from-outside-is-unknown-until-checked](data-from-outside-is-unknown-until-checked.md) | The wasm API and URL parameters are `unknown`, narrowed with `lib/guards.ts`, never cast | `web/test/rules.test.ts` (guards); `web/test/wasm.test.ts` |
| [an-array-fallback-is-a-type-check-never](an-array-fallback-is-a-type-check-never.md) | `Array.isArray(v) ? v : []`, never `v ?? []` or `v \|\| []` | `web/test/rules.test.ts` |
| [copying-goes-through-usecopytoclipboard](copying-goes-through-usecopytoclipboard.md) | Only `useCopyToClipboard` touches `navigator.clipboard`, and the control says whether the copy worked | `web/test/rules.test.ts` |
| [the-playground-never-touches-the-visitors-files](the-playground-never-touches-the-visitors-files.md) | The playground runs the real gnux on an in-memory filesystem; nothing reaches the visitor's disk | `web/test/memfs.test.ts`; `web/test/wasm.test.ts`; review |
| [a-memfs-error-carries-a-code-go-knows](a-memfs-error-carries-a-code-go-knows.md) | Every memfs error is an `FsError` with a code in Go's `errnoByCode` table, judged by the real wasm | `pnpm typecheck`; `web/test/wasm.test.ts` |

## See also

- The business rules (BRULE-n) in [business requirements](../requirements/business-requirements.md), which several of these rules carry out.
- [Workflows](../workflows/README.md): how the processes these rules constrain actually run.
- [Guides](../../documentation/guides/README.md)
- [`AGENTS.md`](../../AGENTS.md) and [`web/AGENTS.md`](../../web/AGENTS.md), which repeat each statement inline.

## Working on the repository

| Rule | Statement | Checked by |
|------|-----------|------------|
| [agents-hand-over-git-commands](agents-hand-over-git-commands.md) | An agent never commits, tags, pushes or releases; it prepares the change and hands the maintainer the commands and the message | Review |
