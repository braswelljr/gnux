# An array fallback is a type check, never a nullish default

When a value might not be an array, the fallback is `Array.isArray(v) ? v : []`, or `arrayOf(v, guard)` from `lib/guards.ts`. It is never `v ?? []` or `v || []`.

## Why

`v ?? []` only replaces `null` and `undefined`. A string, a number or an object passes straight through and is then used as an array: `'abc'.map` throws, and `'abc'.length` quietly gives 3. The nullish form looks like a guard and is not one. A real type check makes the fallback true for every non-array, and `arrayOf` also checks each element.

## What it looks like

From [`web/lib/guards.ts`](../../web/lib/guards.ts):

```ts
/** The elements of `value` that pass `guard`; an empty array if it is not an array at all. */
export function arrayOf<T>(value: unknown, guard: (item: unknown) => item is T): T[] {
  return Array.isArray(value) ? value.filter(guard) : [];
}
```

used where the WebAssembly program reports its tools ([`web/lib/playground/runtime.ts`](../../web/lib/playground/runtime.ts)):

```ts
tools: arrayOf(prop(value, 'tools'), isString),
```

Do: `Array.isArray(v) ? v : []`, or `arrayOf(v, isString)`.

Do not: `v ?? []`, `v || []`.

## How it is checked

- "an array fallback is a type check, never ?? [] or || []" in [`web/test/rules.test.ts`](../../web/test/rules.test.ts) fails on either pattern anywhere in `src/` or `lib/`. It runs in `pnpm test` and the Web workflow.
- The same file tests that `arrayOf('not an array', isString)` and `arrayOf(undefined, isString)` both return `[]`.

## Related

- [Data from outside is unknown until checked](data-from-outside-is-unknown-until-checked.md)
