# Data from outside is unknown until checked

A value that crossed a boundary into the website (the WebAssembly program's API, a URL parameter, anything read off `globalThis`) is typed `unknown` and narrowed with the guards in `lib/guards.ts`. It is never cast to the type it is hoped to have.

## Why

A cast tells the compiler to stop asking; it does not make the value what it claims to be. The WebAssembly API is another program, built separately, and a URL parameter is written by whoever wrote the link. If either is not what the code expects, a cast turns that into a crash far from its cause, or worse, into behavior the page never meant to have. A guard turns it into a clear error at the boundary, or into a safe default.

## What it looks like

The guards, from [`web/lib/guards.ts`](../../web/lib/guards.ts):

```ts
/** Reads a property without vouching for its type, so the next check stays honest. */
export function prop(value: unknown, key: string): unknown {
  return isRecord(value) ? value[key] : undefined;
}

/** The elements of `value` that pass `guard`; an empty array if it is not an array at all. */
export function arrayOf<T>(value: unknown, guard: (item: unknown) => item is T): T[] {
  return Array.isArray(value) ? value.filter(guard) : [];
}
```

The playground reads the API published by `tools/wasm/main.go` through them, from [`web/lib/playground/runtime.ts`](../../web/lib/playground/runtime.ts):

```ts
function readApi(value: unknown) {
  if (!isRecord(value)) throw new Error('gnux.wasm started but did not publish its API');

  const version = prop(value, 'version');
  const run = prop(value, 'run');
  const cwd = prop(value, 'cwd');
  if (!isFunction(run) || !isFunction(cwd)) {
    throw new Error('gnux.wasm published an API without run() and cwd()');
  }
  ...
    tools: arrayOf(prop(value, 'tools'), isString),
```

Do: `isString(prop(value, 'version')) ? ... : 'unknown'`.

Do not: `(globalThis.gnux as GnuxApi).tools`.

## How it is checked

- "the boundary guards narrow what they claim to" in [`web/test/rules.test.ts`](../../web/test/rules.test.ts) tests the guards themselves (`isRecord([])` is false, `isNumber(NaN)` is false, `arrayOf` drops non-matching elements).
- "publishes its API" in `web/test/wasm.test.ts` runs `readApi` against the real build.
- That no boundary value is cast is review; `pnpm lint` runs typescript-eslint but no rule bans `as`.

## Related

- FR-84 in [functional requirements](../requirements/functional-requirements.md).
- [An array fallback is a type check](an-array-fallback-is-a-type-check-never.md)
- [State a reader can arrive at lives in the URL](state-a-reader-can-arrive-at-lives-in-the-url.md)
- [A memfs error carries a code Go knows](a-memfs-error-carries-a-code-go-knows.md)
