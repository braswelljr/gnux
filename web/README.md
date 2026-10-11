# gnux website

The website for [gnux](../README.md): a landing page, the documentation, and a playground
that runs the real gnux in the browser.

- **Docs** (`/docs`): [Fumadocs](https://fumadocs.dev) over MDX that the Go project generates.
- **Playground** (`/playground`): gnux compiled to WebAssembly, on an in-memory filesystem.
  Nothing touches the visitor's disk.

## Requirements

- Node.js 22.18 or newer (the tests run TypeScript directly, which needs Node's type
  stripping) and pnpm 11.
- Go 1.25 or newer, to regenerate the documentation and the WebAssembly build. Without Go the
  site still builds from the copies already under `content/docs/` and `public/`.

## Commands

Run these in `web/`.

| Task                        | Command             |
| --------------------------- | ------------------- |
| Install dependencies        | `pnpm install`      |
| Dev server                  | `pnpm dev`          |
| Production build            | `pnpm build`        |
| Serve the build             | `pnpm start`        |
| Lint                        | `pnpm lint`         |
| Lint and fix                | `pnpm lint:fix`     |
| Type check                  | `pnpm typecheck`    |
| Unit and WebAssembly tests  | `pnpm test`         |
| Check the built site's SEO  | `pnpm test:seo`     |
| Format                      | `pnpm format`       |
| Check formatting            | `pnpm format:check` |
| Regenerate the docs content | `pnpm docs:gen`     |
| Rebuild `public/gnux.wasm`  | `pnpm wasm`         |
| Regenerate both             | `pnpm gen`          |

The repository's Makefile has the same entry points: `make web/dev`, `make web/build`,
`make web/docs` and `make web/wasm`.

## Generated inputs

Two things the site serves are produced by the Go project at the repository root, not written
here:

| Path                                         | Produced by                                      | Command         |
| -------------------------------------------- | ------------------------------------------------ | --------------- |
| `content/docs/**`                            | `go run ./tools/gendocs web/content/docs mdx`    | `pnpm docs:gen` |
| `public/gnux.wasm` and `public/wasm_exec.js` | `make web/wasm` (`GOOS=js GOARCH=wasm go build`) | `pnpm wasm`     |

**Do not edit them by hand**: the next generation overwrites them. To change a docs page,
change the tool's metadata in the Go source and regenerate.

`pnpm dev` and `pnpm build` regenerate both first, through `scripts/gen.mjs`:

- With `go` on the `PATH`, it runs the two commands above (the wasm build directly with `go`,
  the same flags as `make web/wasm`, so `make` is not needed).
- Without `go`, when something is missing (a fresh hosted build: `gnux.wasm` is never
  committed), it downloads the Go release `../go.mod` names from go.dev, checks it against the
  SHA-256 go.dev publishes, and keeps it, with Go's build and module caches, in
  `node_modules/.cache/gnux`. A later build on the same host reuses it. Linux and macOS only.
- Without `go` and with nothing missing, it carries on with the files that are already there.
- It fails only if `content/docs/index.mdx` or `public/gnux.wasm` still does not exist, and says
  how to produce them.
- `GNUX_SKIP_GEN=1` skips generation; `GNUX_FETCH_GO=0` never downloads Go.

## Deploying on Vercel

Import the repository and set **Root Directory** to `web`. Keep **Include files outside the
Root Directory in the Build Step** on (the default): the build reads `../go.mod`, `../tools`
and the rest of the Go source. Everything else is detected: Next.js, pnpm from the lockfile,
and Node.js from `engines`. No Go setup is needed: `pnpm build` runs `scripts/gen.mjs`, which
downloads Go on the first build and builds `gnux.wasm` from the commit being deployed. Vercel
restores `node_modules` between builds, so later builds skip the download and reuse Go's
caches.

### What the docs generator must emit

- Every page has `title` and `description` frontmatter. Both are required: they become the
  page heading, the `<title>`, the meta description, the Open Graph card and the structured
  data.
- Each directory has a `meta.json` with a `title` and a `pages` array naming its entries in
  order (file names without the extension; a subdirectory by its name).
- The body is MDX, so it is stricter than markdown: a literal `<` or `{` in prose must be
  inside a code span or a code fence, or escaped. Standard markdown (headings, lists, tables,
  links, fenced code) needs nothing else.
- Fenced code blocks name their language (`bash`, `powershell`, `go`, `text`).

A new page needs no change here. It appears in the sidebar, the search index, `sitemap.xml`,
`llms.txt` and `llms-full.txt` as soon as it is generated and listed in `meta.json`.

## The playground

`tools/wasm/main.go` builds gnux for `js/wasm`. Once started it publishes an API on
`globalThis.gnux` (`run(line)`, `cwd()`, `tools`, `version`, `onOutput`).

Go's js/wasm runtime performs every file and process operation through Node-style `fs` and
`process` globals. The browser has neither, so the site supplies them:

| File                         | Role                                                                   |
| ---------------------------- | ---------------------------------------------------------------------- |
| `lib/playground/memfs.ts`    | The in-memory filesystem, and the `fs`/`process`/`path` objects        |
| `lib/playground/seed.ts`     | The files a fresh session starts with                                  |
| `lib/playground/runtime.ts`  | Boots the program against a filesystem and checks the API it publishes |
| `lib/playground/browser.ts`  | Fetches `wasm_exec.js` and `gnux.wasm` (streaming, with a fallback)    |
| `lib/playground/complete.ts` | Tab completion                                                         |
| `lib/playground/ansi.ts`     | ANSI color escapes                                                     |

These files have no dependency on React or the DOM (`browser.ts` aside), so `pnpm test` boots
the real `public/gnux.wasm` in Node through the same `runtime.ts` and drives it: it runs
commands and asserts on their output, their exit status and the files they leave behind. That
suite is the proof that the filesystem honors the contract in `$GOROOT/src/syscall/fs_js.go`.
Run it after rebuilding the wasm.

A link can carry a command: `/playground?cmd=lsx%20-la` opens the playground with `lsx -la`
typed at the prompt. It is never run on arrival.

## SEO

`src/config/site.ts` holds the site's name, description, keywords and URL. The URL comes from
`NEXT_PUBLIC_SITE_URL`; the fallback in that file is a placeholder to confirm before launch.

Every page builds its metadata with `createMetadata` (`src/utils/metadata.ts`), which sets
the canonical URL, the Open Graph tags, the Twitter card and the robots directives together.
The site also serves `robots.txt`, `sitemap.xml`, `manifest.webmanifest`, an icon, generated
Open Graph images (`/opengraph-image` and `/og/docs/<page>/image.png`), `llms.txt`,
`llms-full.txt`, and JSON-LD on the home page and on every docs page.

`pnpm test:seo` starts the production build, requests all of the above and checks it. It
needs `pnpm build` to have run.

## Layout

```text
content/docs/        Generated MDX (see above)
public/              gnux.wasm and wasm_exec.js (generated)
lib/                 Framework-agnostic helpers: cn, guards, the docs source, the playground engine
scripts/             gen.mjs (pre-build generation), check-seo.mjs
source.config.ts     Fumadocs MDX collections
src/
  app/               Routes: (home) landing and playground, docs, search, SEO routes
  components/
    ui/              Atoms and molecules: Button, Heading, Card, CodeBlock, CopyButton, Reveal
    home/            Landing page sections
    playground/      Terminal, side panel
    shared/          Used across pages (JSON-LD)
  config/            site.ts, navigation, playground examples
  hooks/             use-gnux, use-terminal, use-copy-to-clipboard
  providers/         The client provider tree
  styles/            main.css (Tailwind v4 entry and theme), fumadocs.css
  utils/             metadata.ts (createMetadata)
test/                node --test suites: the wasm, the filesystem, completion, the house rules
```

See [`AGENTS.md`](AGENTS.md) for the conventions.
