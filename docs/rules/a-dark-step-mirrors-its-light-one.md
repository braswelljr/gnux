# A dark step mirrors its light one

Every color the website sets has a dark-theme counterpart on the mirrored step of the same scale: `text-red-700 dark:text-red-300`, `text-green-700 dark:text-green-300`. The default choice is Fumadocs' `fd-*` tokens, which already switch with the theme and keep the site matching the docs.

## Why

A color picked for the light theme alone is either unreadable or glaring in the dark one. Mirroring the step (700 on light, 300 on dark; 600 and 400 for icons) keeps the contrast against the background roughly the same in both themes without a separate decision each time. Using the `fd-*` tokens where they fit means the landing page, the playground and the documentation share one palette and one theme switch.

## What it looks like

Semantic tokens first. From [`web/src/components/ui/button.tsx`](../../web/src/components/ui/button.tsx):

```ts
const variants = {
  primary: 'bg-fd-primary text-fd-primary-foreground hover:bg-fd-primary/90',
  secondary:
    'border bg-fd-card text-fd-card-foreground hover:bg-fd-accent hover:text-fd-accent-foreground',
  ghost: 'text-fd-muted-foreground hover:bg-fd-accent hover:text-fd-accent-foreground'
};
```

Where a hue carries information (a terminal's prompt, its error stream, ANSI colors), each light step has its mirror. From [`web/src/components/playground/terminal.tsx`](../../web/src/components/playground/terminal.tsx):

```tsx
<span className="text-green-700 dark:text-green-300">guest@gnux</span>
...
<span className="text-blue-700 dark:text-blue-300">{shortPath(cwd, home)}</span>
...
entry.kind === 'stderr' && 'text-red-700 dark:text-red-300',
```

and the error icon in [`web/src/components/playground/playground.tsx`](../../web/src/components/playground/playground.tsx): `text-red-600 dark:text-red-400`.

Do: `text-fd-muted-foreground`, or `text-red-700 dark:text-red-300`.

Do not: `text-red-500` with no `dark:` step, or `dark:text-red-200` paired with `text-red-700`.

## How it is checked

- Review. Nothing scans for an unpaired color class, and the Web workflow does not render either theme.

## Related

- NFR-24 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [The markup says what the thing is](the-markup-says-what-the-thing-is.md): a color is never the only signal.
- [Build the UI from atoms upward](build-the-ui-from-atoms-upward.md)
- [Type sizes come from the scale](type-sizes-come-from-the-scale.md)
