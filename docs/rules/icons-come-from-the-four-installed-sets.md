# Icons come from the installed set

Every icon on the website comes from `lucide-react`, the one icon library installed. No other icon package is imported, and no SVG icon is pasted inline.

## Why

Icons from different sets differ in stroke width, corner radius and optical size, and side by side they look like a mistake. One set also means one dependency to update and one way to size an icon (through the `[&_svg]:size-*` classes on `Button`). The file name keeps its original wording from a template that allowed several sets; in this repository the installed set is `lucide-react` alone.

## What it looks like

From [`web/src/components/playground/playground.tsx`](../../web/src/components/playground/playground.tsx):

```tsx
import { LoaderCircle, TriangleAlert } from 'lucide-react';
...
<LoaderCircle
  aria-hidden="true"
  className="size-6 animate-spin text-fd-muted-foreground motion-reduce:animate-none"
/>
```

Do: `import { Copy } from 'lucide-react'`.

Do not: `import { FiCopy } from 'react-icons/fi'`, or an `<svg>` copied from another icon site.

## How it is checked

- "icons come from lucide-react only" in [`web/test/rules.test.ts`](../../web/test/rules.test.ts) fails on imports from `react-icons`, `iconsax-react`, `@radix-ui/react-icons` and `@heroicons` in `src/` and `lib/`. It runs in `pnpm test` and the Web workflow.
- An inline SVG icon is review. (The site's own `icon.svg` and generated Open Graph images are not icons in this sense.)

## Related

- [An icon-only button takes an icon size](an-icon-only-button-takes-an-icon-size.md)
- [Build the UI from atoms upward](build-the-ui-from-atoms-upward.md)
