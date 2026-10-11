# Copying goes through useCopyToClipboard

Every copy to the clipboard goes through the `useCopyToClipboard` hook, and the control that triggered it says whether it worked. No component calls `navigator.clipboard` itself.

## Why

A copy can fail. The Clipboard API is absent outside a secure context (plain http on a LAN address, for example) and a browser may refuse a write. A component that calls `navigator.clipboard.writeText` and then shows "Copied" unconditionally is telling the reader something that may be false, which is the website's version of reporting success for a failure. One hook means one place that handles the missing API, the refused write and the timer that resets the acknowledgement.

## What it looks like

From [`web/src/hooks/use-copy-to-clipboard.ts`](../../web/src/hooks/use-copy-to-clipboard.ts):

```ts
/**
 * Copies text to the clipboard, and tells the truth about whether it worked.
 *
 * This is the only place the site touches `navigator.clipboard`. ...
 */
...
// Over plain http the property is simply not there, so reading
// `.writeText` off it would throw.
if (typeof window === 'undefined' || !navigator?.clipboard?.writeText) {
  return failed(new Error('The clipboard is unavailable in this context'));
}
```

Copied and failed are one piece of state, so they cannot both be true. Each caller shows the outcome in place: `CopyButton` announces "Copied" or "Could not copy. Select the text instead." in a `role="status"` element, and `ShareButton` says "Link copied" or "Could not copy. Copy the address bar instead."

Do: `const { copyToClipboard, isCopied, error } = useCopyToClipboard();` and render `error` beside the control.

Do not: `navigator.clipboard.writeText(text); setCopied(true);` in a component.

## How it is checked

- "copying goes through useCopyToClipboard" in [`web/test/rules.test.ts`](../../web/test/rules.test.ts) fails on `navigator.clipboard` (or `navigator?.clipboard`) in any file under `src/` or `lib/` other than the hook. It runs in `pnpm test` and the Web workflow.

## Related

- [A failure is never reported as success](a-failure-is-never-reported-as-success.md), the same rule for the tools.
- [The markup says what the thing is](the-markup-says-what-the-thing-is.md)
- [State a reader can arrive at lives in the URL](state-a-reader-can-arrive-at-lives-in-the-url.md)
