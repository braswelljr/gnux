# A memfs error carries a code Go knows

The playground's in-memory filesystem implements a contract owned by Go: `$GOROOT/src/syscall/fs_js.go`. Every error it raises is an `FsError` whose `code` is one of the names in Go's `errnoByCode` table, and the real WebAssembly build is the judge of any change to it.

## Why

Go's `js/wasm` runtime turns a JavaScript error into a Go `syscall.Errno` by looking up its `code`. A code Go does not know is not a wrong error message: it is a panic that kills the whole program, and the playground with it. So `ENOENT` must be `ENOENT` exactly, and a new failure mode must reuse a code from the table. Because the contract belongs to another program, only that program can say whether the filesystem keeps it; a test against a mock of Go would only test the mock.

## What it looks like

The allowed codes are a closed type, from [`web/lib/playground/memfs.ts`](../../web/lib/playground/memfs.ts):

```ts
/**
 * The error codes this filesystem raises. Every one must be a key of Go's
 * `errnoByCode` table: Go panics on a code it does not know.
 */
export type ErrorCode =
  | 'EACCES'
  | 'EBADF'
  ...
  | 'ENOTEMPTY'
  | 'EPERM';

/** The error every operation throws, shaped like Node's system errors. */
export class FsError extends Error {
  code: ErrorCode;
  syscall: string;
  path?: string;
```

and operations throw it with a code and the syscall name: `throw new FsError('ENOENT', 'rmdir', path)`. Errors from `process.cwd` and `process.chdir` carry a `code` too.

`lib/playground/` is plain erasable TypeScript (no enums, no parameter properties, no path aliases), because Node runs these files directly in the tests by stripping the types.

Do: `throw new FsError('ENOTDIR', 'open', path)`.

Do not: `throw new Error('not a directory')`, or add a code to `ErrorCode` without checking it is in Go's table.

## How it is checked

- The `ErrorCode` union makes an unknown code a type error (`pnpm typecheck`).
- "the callback surface passes results and coded errors" in [`web/test/memfs.test.ts`](../../web/test/memfs.test.ts) checks that callbacks receive coded errors.
- [`web/test/wasm.test.ts`](../../web/test/wasm.test.ts) runs the real `gnux.wasm` against the filesystem, including "permission errors surface as diagnostics" and "a symbolic link loop is an error, not a hang". It runs in `make web/test` and the Web workflow after `make web/wasm`.

## Related

- FR-47, FR-84 in [functional requirements](../requirements/functional-requirements.md); NFR-19, NFR-33 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [The playground never touches the visitor's files](the-playground-never-touches-the-visitors-files.md)
- [Diagnostics are written in GNU's words](diagnostics-are-written-in-gnus-words.md): Go maps the code to the errno that `core.ErrText` renders.
- [Data from outside is unknown until checked](data-from-outside-is-unknown-until-checked.md)
