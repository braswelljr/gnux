# An icon-only button takes an icon size

A button whose only content is an icon uses `size="icon"` or `size="icon-sm"`, has an `aria-label` that names the action, and marks its icon `aria-hidden`. It is never squared off with a className.

## Why

The icon sizes set the button's box and the glyph's size together, so every icon button is the same target size and its icon is centered the same way. A button sized by hand drifts by a pixel or two from the others. The glyph is not a name: without an `aria-label` a screen reader announces "button" and nothing else, and without `aria-hidden` on the SVG it may announce the icon's markup too.

## What it looks like

The sizes, from [`web/src/components/ui/button.tsx`](../../web/src/components/ui/button.tsx):

```ts
/**
 * The icon sizes set the box and the glyph together, so an icon-only button is
 * sized with `size="icon"` or `"icon-sm"`, never squared off with a className.
 */
const sizes = {
  ...
  'icon-sm': 'size-8 [&_svg]:size-4',
  icon: 'size-10 [&_svg]:size-5'
};
```

In use, from [`web/src/components/ui/copy-button.tsx`](../../web/src/components/ui/copy-button.tsx), where the label also changes to report the result:

```tsx
<Button
  variant="ghost"
  size="icon-sm"
  aria-label={isCopied ? 'Copied' : label}
  title={label}
  onClick={() => void copyToClipboard(value)}
  {...props}
>
  {isCopied ? <Check aria-hidden="true" /> : <Copy aria-hidden="true" />}
</Button>
```

A button with text beside its icon (Reset, Copy link) keeps a text size and still marks the icon `aria-hidden`.

Do: `<Button size="icon-sm" aria-label="Copy install command"><Copy aria-hidden="true" /></Button>`.

Do not: `<Button className="size-8 p-0"><Copy /></Button>`.

## How it is checked

- Review. The `size` prop is typed, so a misspelled size does not compile, but nothing checks that an icon-only button has an `aria-label`.

## Related

- NFR-24 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [The markup says what the thing is](the-markup-says-what-the-thing-is.md)
- [Icons come from the installed set](icons-come-from-the-four-installed-sets.md)
- [Build the UI from atoms upward](build-the-ui-from-atoms-upward.md)
