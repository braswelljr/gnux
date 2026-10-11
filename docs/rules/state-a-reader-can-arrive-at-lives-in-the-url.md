# State a reader can arrive at lives in the URL

If a reader could be sent to a particular state of a page, that state lives in the URL, read and written through `nuqs`. The playground's `?cmd=` is the example: a link that opens the playground with a command ready at the prompt.

## Why

A link is the shortest path there is. State held only in React memory cannot be shared, bookmarked or restored with the back button, so "try this command" would need instructions instead of a link. Keeping such state in the URL makes the link the whole of the state. Only state a reader would want to arrive at belongs there: whether a panel is folded open does not.

## What it looks like

From [`web/src/components/playground/playground.tsx`](../../web/src/components/playground/playground.tsx):

```tsx
// `?cmd=` is the one piece of state a reader can arrive with: a link to the
// playground with a command ready at the prompt.
const [cmd, setCmd] = useQueryState('cmd', parseAsString.withDefault(''));
```

The last command run is written back, so the address bar always describes what the page shows, and the share button copies a link built the same way ([`web/src/components/playground/share-button.tsx`](../../web/src/components/playground/share-button.tsx)):

```tsx
const url = new URL(window.location.href);
url.search = '';
url.searchParams.set('cmd', command);
void copyToClipboard(url.toString());
```

A URL parameter is input from outside, so it is treated that way: the command is cut to one printable line of at most 500 characters and placed in the prompt for the reader to run or discard. It is never run on arrival.

Do: `useQueryState('name', parseAs...)` for state a link should carry.

Do not: `useState` for something a reader would want to share, or run a command because the URL contains it.

## How it is checked

- Review. The page reading `?cmd=` is wrapped in `Suspense` so it can prerender, which the build checks; nothing tests the round trip.

## Related

- FR-48 in [functional requirements](../requirements/functional-requirements.md).
- [Data from outside is unknown until checked](data-from-outside-is-unknown-until-checked.md)
- [The shortest path, and only one of them](the-shortest-path-and-only-one-of-them.md)
- [Copying goes through useCopyToClipboard](copying-goes-through-usecopytoclipboard.md)
