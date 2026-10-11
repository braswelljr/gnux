# Type sizes come from the scale

Text on the website is sized with Tailwind's scale, `text-xs` through `text-3xl`, and never with an arbitrary value such as `text-[13px]`. Headings take their size from the `Heading` atom's named sizes.

## Why

A scale gives the page a rhythm a reader can feel: each step is a clear difference. Arbitrary sizes creep in one pixel at a time until two labels side by side are 13px and 14px for no reason. Scale steps are also in `rem`, so they follow the reader's browser font setting, which a pixel value written by hand does not.

## What it looks like

From [`web/src/components/ui/heading.tsx`](../../web/src/components/ui/heading.tsx):

```ts
const sizes = {
  '3xl': 'text-3xl font-semibold tracking-tight',
  '2xl': 'text-2xl font-semibold tracking-tight',
  xl: 'text-xl font-semibold tracking-tight',
  lg: 'text-lg font-semibold',
  base: 'text-base font-semibold',
  sm: 'text-sm font-semibold'
};
```

One deliberate exception sits inside the scale rather than outside it: the playground's input is `text-base` on small screens and `sm:text-sm` above, because anything smaller than 16px makes iOS zoom the page on focus ([`web/src/components/playground/terminal.tsx`](../../web/src/components/playground/terminal.tsx)).

Do: `text-sm`, `text-xs`, or `<Heading level={2} size="sm">`.

Do not: `text-[13px]`, `text-[0.8rem]`.

## How it is checked

- "type sizes come from the scale, never text-[...]" in [`web/test/rules.test.ts`](../../web/test/rules.test.ts) fails on `text-[<number>px|rem|em]` anywhere in `src/` or `lib/`. It runs in `pnpm test` and the Web workflow.

## Related

- [Build the UI from atoms upward](build-the-ui-from-atoms-upward.md)
- [A dark step mirrors its light one](a-dark-step-mirrors-its-light-one.md)
