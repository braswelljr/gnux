# No em dashes, and US spelling

Everything written in this repository, in prose, comments, user-facing copy, metadata, accessible names and diagnostics, uses US spelling and no em dashes. Where an em dash would go, use the punctuation that names the relationship: a colon before an explanation, a comma for an aside, parentheses for a parenthetical, or a spaced hyphen ` - ` in the few places a plain dash reads best.

## Why

One voice reads as one product. An em dash is the punctuation that avoids deciding what the relationship between two clauses is, and it renders inconsistently in terminals, man pages and some screen readers. US spelling is the convention the code already uses (`color`, `behavior`, `initialize`), and mixing spellings makes search unreliable: a reader grepping for `behavior` should find every mention.

This rule covers both halves of the repository. It is listed with the frontend rules because the website is where it is checked most strictly.

## What it looks like

From the website's copy, [`web/src/app/(home)/playground/page.tsx`](../../web/src/app/(home)/playground/page.tsx), a colon introducing what follows:

```tsx
a virtual filesystem held in memory: nothing touches your disk, nothing is uploaded, and
external programs are unavailable.
```

From a Go comment, [`core/touch/lchtimes_other.go`](../../core/touch/lchtimes_other.go), a spaced hyphen:

```go
// is refused - silently touching its target instead would change the wrong
// file.
```

Do: "gnux mirrors GNU: flags, semantics and diagnostics."

Do not: put an em dash (U+2014) between the two halves of that sentence.

## How it is checked

- "no em dashes in anything we write" in [`web/test/rules.test.ts`](../../web/test/rules.test.ts) scans every text file under `web/` (except generated content) for U+2014. It runs in `pnpm test` and the Web workflow.
- `misspell` with `locale: US` in [`.golangci.yml`](../../.golangci.yml), run by the Lint workflow, checks spelling in Go sources.
- Em dashes in Go sources and in the markdown under `docs/` are review.

## Related

- NFR-42 in [non-functional requirements](../requirements/non-functional-requirements.md).
- [The markup says what the thing is](the-markup-says-what-the-thing-is.md), for accessible names.
- [Diagnostics are written in GNU's words](diagnostics-are-written-in-gnus-words.md)
