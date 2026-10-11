# The playground never touches the visitor's files

The playground runs the real gnux, compiled to WebAssembly, against an in-memory filesystem that the page supplies. Nothing it does reads or writes the visitor's disk, nothing is uploaded, and no external program can be started. The only network request it makes is the download of `gnux.wasm` itself.

## Why

The playground invites strangers to run `rmx -rf` in their browser. That is only acceptable if it provably cannot reach anything real. It also has to be the real gnux, not a simulation, or it demonstrates nothing; the in-memory filesystem is how both hold at once. Go's `js/wasm` runtime reaches files through Node-style `fs` and `process` objects on the global object, so supplying those is enough to confine every tool, with no change to the tools.

## What it looks like

From [`tools/wasm/main.go`](../../tools/wasm/main.go):

```go
// The tools reach the filesystem through the Node-style `fs` and
// `process` objects Go's js/wasm runtime expects on the global object; the
// playground supplies in-memory ones, so nothing touches the visitor's disk.
```

The page installs them before the program starts, from [`web/lib/playground/runtime.ts`](../../web/lib/playground/runtime.ts):

```ts
g.fs = memfs.nodeFs();
g.path = memfs.nodePath();
g.process = memfs.nodeProcess();
```

The filesystem lives in [`web/lib/playground/memfs.ts`](../../web/lib/playground/memfs.ts) and is seeded from `seed.ts`; Reset discards it and starts a fresh one. The page says the same thing to the visitor, in [`web/src/app/(home)/playground/page.tsx`](../../web/src/app/(home)/playground/page.tsx): "nothing touches your disk, nothing is uploaded, and external programs are unavailable."

Do: add a file to the playground by adding it to the seed.

Do not: add a file picker, the File System Access API, persistence of the virtual filesystem to storage, or an upload of its contents.

## How it is checked

- [`web/test/memfs.test.ts`](../../web/test/memfs.test.ts) tests the filesystem's behavior on its own.
- [`web/test/wasm.test.ts`](../../web/test/wasm.test.ts) runs the real `gnux.wasm` in Node against it: every tool, pipelines, redirections, permission errors.
- That nothing else is reachable is review: the only `fetch` in `web/lib` is the wasm download in `browser.ts`.

## Related

- BRULE-5, BRULE-9 in [business requirements](../requirements/business-requirements.md); FR-47, FR-84, FR-85 in [functional requirements](../requirements/functional-requirements.md); NFR-17, NFR-19, NFR-33 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [A memfs error carries a code Go knows](a-memfs-error-carries-a-code-go-knows.md)
- [The shell runs no external program implicitly](the-shell-runs-no-external-program-implicitly.md)
- [Workflows](../workflows/README.md), in particular the WebAssembly playground.
