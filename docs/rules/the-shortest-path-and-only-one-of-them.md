# The shortest path, and only one of them

Each thing on the website is reached by one obvious route in as few steps as possible, and that route is not duplicated by a second control that does the same thing. Each view has one primary action, every label says what will happen, anything disabled has its reason shown beside it, and no state is carried by color alone.

## Why

Two buttons that do the same thing make a reader wonder how they differ; a third navigation route to the same page is one more thing to scan past. A visitor comes to the site to do one of three things (learn what gnux is, look something up, try it), so each of those should be one click from wherever they are. A disabled control with no reason looks broken, and a label like "Submit" makes the reader guess what will happen.

## What it looks like

The site's navigation has two links, shared by every layout. From [`web/src/config/layout.tsx`](../../web/src/config/layout.tsx):

```tsx
links: [
  { text: 'Docs', url: '/docs', active: 'nested-url' },
  { text: 'Playground', url: '/playground', active: 'nested-url' }
],
```

The landing page has one primary action and one secondary one. From [`web/src/components/home/hero.tsx`](../../web/src/components/home/hero.tsx):

```tsx
<Link href="/playground" className={buttonVariants({ variant: 'primary', size: 'lg' })}>
  Try it in your browser
  <ArrowRight aria-hidden="true" />
</Link>
<Link href="/docs" className={buttonVariants({ variant: 'secondary', size: 'lg' })}>
  <BookOpen aria-hidden="true" />
  Read the docs
</Link>
```

In the playground, an example runs in one click; each tool in the side panel links straight to its reference page; a shared link (`?cmd=`) arrives with the command already at the prompt. Disabled examples carry their reason. From [`web/src/components/playground/side-panel.tsx`](../../web/src/components/playground/side-panel.tsx):

```tsx
{busy && (
  <p className="mb-2 text-xs text-fd-muted-foreground">
    A command is running. Examples are available again when it finishes.
  </p>
)}
```

Labels say what will happen: "Copy link" with the title "Copy a link that opens the playground with this command"; "Reset" with the title "Start a fresh gnux with the original files". On small screens the side panel folds under one "Examples and tools" button rather than adding a second copy of its contents elsewhere.

Do: one primary button per view, a label naming the outcome, and a sentence beside a disabled control.

Do not: a second "Run" button beside the prompt that does what Enter does, a third navigation entry to the docs, or a grayed-out control with no explanation.

## How it is checked

- Review. Nothing automated checks navigation or labels. `TestMarkdownLinksResolve` and `pnpm test:seo` catch broken routes, not redundant ones.

## Related

- NFR-24 in [non-functional requirements](../requirements/non-functional-requirements.md); FR-45, FR-47, FR-48 in [functional requirements](../requirements/functional-requirements.md).
- [The markup says what the thing is](the-markup-says-what-the-thing-is.md)
- [State a reader can arrive at lives in the URL](state-a-reader-can-arrive-at-lives-in-the-url.md)
- [Build the UI from atoms upward](build-the-ui-from-atoms-upward.md)
