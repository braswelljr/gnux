# The markup says what the thing is

The website uses the HTML element that names what each thing is: a `button` for an action, a link for navigation, `section` and `aside` with a label for regions, a `label` for every input, one `h1` per page and no skipped heading levels. Every control has an accessible name, ARIA is added only where HTML has no element for the role, and no meaning is carried by color alone.

## Why

A screen reader, a keyboard, a search engine and a reader-mode view all see the markup, not the styling. A `div` with an `onClick` is not focusable and is not announced as a button; a heading chosen for its size breaks the outline that assistive technology navigates by; red text tells a reader who cannot see red nothing. Native elements get focus, keyboard handling and roles for free, so they come first and ARIA fills the gaps.

## What it looks like

Outline and size are separate decisions. From [`web/src/components/ui/heading.tsx`](../../web/src/components/ui/heading.tsx):

```tsx
type HeadingProps = ComponentProps<'h1'> & {
  /** The heading's place in the page outline. */
  level: 1 | 2 | 3 | 4;
  /** How large it reads, chosen separately so the outline never follows the type size. */
  size?: keyof typeof sizes;
};
```

so the side panel's small headings are still `h2` in the outline: `<Heading level={2} size="sm" ...>`.

Regions are labeled, and an input has a real label even when it is visually hidden. From [`web/src/components/playground/terminal.tsx`](../../web/src/components/playground/terminal.tsx):

```tsx
<section aria-label="Terminal" ...>
  ...
  <div role="log" aria-live="polite" aria-relevant="additions" aria-label="Terminal output">
  ...
  <label htmlFor={inputId} className="sr-only">Command</label>
  <input ref={inputRef} id={inputId} type="text" name="command" ... aria-describedby={hintId} />
```

Color is never the only signal. Error output is red, and also named:

```tsx
{/* Color alone must not carry the meaning: the stream is named for screen readers. */}
{entry.kind === 'stderr' && <span className="sr-only">Error output: </span>}
```

Status changes are announced: the copy button's result is in a `role="status"` element, a failed start of the playground in `role="alert"`, and the loading state in `role="status"`. The `Button` atom defaults `type="button"` and has a visible `focus-visible` ring.

Do: `<Button onClick={onReset}>`, `<Link href="/docs">`, `<aside aria-label="Examples and tools">`.

Do not: `<div onClick>`, a heading level chosen for its size, or a red border as the only sign of an error.

## How it is checked

- Review. The Web workflow lints, type-checks and builds, and `Heading` requires a `level` at the type level, but nothing runs an accessibility audit or a browser test.

## Related

- NFR-24 in [non-functional requirements](../requirements/non-functional-requirements.md); FR-45 in [functional requirements](../requirements/functional-requirements.md).
- [An icon-only button takes an icon size](an-icon-only-button-takes-an-icon-size.md)
- [The shortest path, and only one of them](the-shortest-path-and-only-one-of-them.md)
- [Build the UI from atoms upward](build-the-ui-from-atoms-upward.md)
