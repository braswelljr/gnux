# A public page describes itself

Every page and layout that has something to say about itself builds its metadata with `createMetadata(siteConfig, {...})`. No page writes a Next.js `Metadata` literal, and no component writes the site's origin: the URL comes from `siteConfig.url`.

## Why

A link preview, a search result and a crawler read the title, the description, the canonical URL, the Open Graph tags and the Twitter card, and those are exactly the fields a hand-written literal forgets. Next.js also replaces `openGraph` and `twitter` wholesale when a page sets them, so a page that sets only a title silently loses the site's share image. One builder emits every field in full, every time.

## What it looks like

From [`web/src/app/(home)/playground/page.tsx`](../../web/src/app/(home)/playground/page.tsx):

```tsx
export const metadata = createMetadata(siteConfig, {
  title: 'Playground',
  description:
    'Run the real gnux in your browser: every tool and the gnux shell, compiled to WebAssembly, on an in-memory filesystem. Nothing touches your disk.',
  keywords: ['playground', 'online terminal', 'try coreutils online'],
  canonical: '/playground'
});
```

The builder, [`web/src/utils/metadata.ts`](../../web/src/utils/metadata.ts), explains itself:

```ts
/**
 * Builds the metadata of a page. Every page and layout goes through this, and
 * none writes a `Metadata` literal, because the canonical URL, the Open Graph
 * tags and the Twitter card are the only things a link preview reads and they
 * are exactly what a hand-written literal forgets.
```

Do: `export const metadata = createMetadata(siteConfig, { title, description, canonical })`.

Do not: `export const metadata: Metadata = { title: 'Playground' }`, or `https://...` written into a component.

## How it is checked

- "every page builds its metadata with createMetadata" in [`web/test/rules.test.ts`](../../web/test/rules.test.ts): every `page.tsx`, `layout.tsx` and `not-found.tsx` under `src/app` that exports metadata must call `createMetadata(` and must not write `: Metadata = {`.
- `pnpm test:seo` ([`web/scripts/check-seo.mjs`](../../web/scripts/check-seo.mjs)) starts the built site and checks each page's title, description, canonical URL and structured data, plus `robots.txt`, the sitemap and `llms.txt`. The Web workflow runs it after the build.

## Related

- FR-49, FR-50, FR-86 in [functional requirements](../requirements/functional-requirements.md).
- [The markup says what the thing is](the-markup-says-what-the-thing-is.md)
- [State a reader can arrive at lives in the URL](state-a-reader-can-arrive-at-lives-in-the-url.md)
