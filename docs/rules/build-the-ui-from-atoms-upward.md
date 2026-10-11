# Build the UI from atoms upward

The website's interface is composed bottom-up: atoms and molecules in `src/components/ui`, exported from one `index.ts`; organisms in `home/`, `playground/` and `shared/` built from them; pages that compose organisms. Before styling something from scratch, check `src/components/ui` for the piece that already exists.

## Why

A button styled by hand in one organism drifts from the button in the next: a different focus ring, a missing `type="button"`, no reduced-motion handling. Building from the same atoms keeps every control consistent with the docs (which use Fumadocs' own components) and puts each decision, such as how an icon-only button is sized or how a heading separates level from size, in one file.

## What it looks like

The atoms, from [`web/src/components/ui/index.ts`](../../web/src/components/ui/index.ts):

```ts
export { Button, buttonVariants, type ButtonProps } from './button';
export { Card, CardContent, CardDescription, CardHeader } from './card';
export { CodeBlock } from './code-block';
export { CopyButton } from './copy-button';
export { Heading } from './heading';
export { Reveal } from './reveal';
export { Container, Section } from './section';
```

A link that should look like an action reuses the button's classes rather than restyling. From [`web/src/components/home/hero.tsx`](../../web/src/components/home/hero.tsx):

```tsx
<Link href="/docs" className={buttonVariants({ variant: 'secondary', size: 'lg' })}>
```

Components below the page are presentational: they take data through props and report through callbacks. The `SidePanel` organism receives `tools`, `examples`, `busy`, `onRun` and `onReset`; the data comes from the `Playground` organism, which owns the session.

Do: `import { Button, Heading } from '@/components/ui'`, and add a variant to the atom when a new look is genuinely needed.

Do not: a `<button className="rounded-lg px-4 ...">` inside an organism, or a page that styles its own layout instead of using `Container` and `Section`.

## How it is checked

- Review. The type system enforces the atoms' props (a `Heading` without `level` does not compile), but nothing detects a hand-rolled control.

## Related

- [The markup says what the thing is](the-markup-says-what-the-thing-is.md)
- [An icon-only button takes an icon size](an-icon-only-button-takes-an-icon-size.md)
- [Type sizes come from the scale](type-sizes-come-from-the-scale.md)
- [A dark step mirrors its light one](a-dark-step-mirrors-its-light-one.md)
