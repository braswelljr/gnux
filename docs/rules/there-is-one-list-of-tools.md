# There is one list of tools

`Tools()` in `cmd/registry.go` is the only list of tools. The root command, the `argv[0]` dispatcher, `gnux --list`, the shell, the reference generator, the WebAssembly build and the website's playground all derive the set of tools from it, and none of them keeps a second copy.

## Why

Every second list goes stale. A tool added to the registry but not to a hand-kept list in the docs, the installer or the playground is a tool that exists in one place and is missing in another. With one list, registering a tool is the whole of making it available everywhere, and removing one cannot leave a dangling reference.

## What it looks like

From [`cmd/registry.go`](../../cmd/registry.go):

```go
// Tools is the registry of every tool gnux ships, in the order they are listed
// in help and documentation. Add a new tool's Spec here to make it available as
// a subcommand, as an argv[0] name, and in the generated reference.
func Tools() []tool.Spec {
	return []tool.Spec{
		basenamex.Spec(),
		catx.Spec(),
		...
```

The WebAssembly build publishes the same list to the page, from [`tools/wasm/main.go`](../../tools/wasm/main.go):

```go
var tools []any
for _, t := range cmd.Tools() {
	tools = append(tools, t.Name())
}
api.Set("tools", js.ValueOf(tools))
```

And the playground reads it from the running build instead of listing tools itself, filtering its examples against it. From [`web/src/components/playground/playground.tsx`](../../web/src/components/playground/playground.tsx):

```ts
const tools = useMemo(() => [...session.tools].sort(), [session]);
const available = useMemo(
  () => examples.filter((example) => example.needs.every((tool) => tools.includes(tool))),
  [tools]
);
```

`make links`, `install.sh` and the Dockerfile create the per-tool links from `gnux --list`.

Do: add `<gnu>x.Spec()` to `Tools()`, alphabetically.

Do not: write a tool name list in a test, the website, an install script or a doc page.

## How it is checked

- `TestRegistryIsConsistent`, `TestEveryToolAnswersHelpAndVersion` and `TestRootListAndVersion` in [`cmd/main_test.go`](../../cmd/main_test.go) iterate over `Tools()`; none names a tool, so a new tool is covered as soon as it is registered.
- `TestGenerateAll` and `TestGenerateMDX` in [`cmd/docs/docs_test.go`](../../cmd/docs/docs_test.go) require a page per registered tool in every format.
- "publishes its API" in `web/test/wasm.test.ts` checks the list the page receives.
- That the website hardcodes no tool list is review.

## Related

- BRULE-7 in [business requirements](../requirements/business-requirements.md); FR-17, FR-21 in [functional requirements](../requirements/functional-requirements.md); NFR-26, NFR-43 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [The reference documents itself, and it is checked](the-reference-documents-itself-and-it-is-checked.md)
- [Workflows](../workflows/README.md)
