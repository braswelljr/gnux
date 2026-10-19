// Checks what a crawler would read from the built site: robots.txt, the
// sitemap, the llms files, the Open Graph images, and the head of each kind of
// page. Run `pnpm build` first; this starts `next start` on a free port, makes
// its requests, and stops the server.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { createServer } from 'node:net';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const web = join(dirname(fileURLToPath(import.meta.url)), '..');

if (!existsSync(join(web, '.next', 'BUILD_ID'))) {
  console.error('check-seo: no production build found. Run `pnpm build` first.');
  process.exit(1);
}

/** Asks the system for a port nobody is using. */
function freePort() {
  return new Promise((resolve, reject) => {
    const server = createServer();
    server.on('error', reject);
    server.listen(0, '127.0.0.1', () => {
      const { port } = server.address();
      server.close(() => resolve(port));
    });
  });
}

const port = await freePort();
const origin = `http://127.0.0.1:${port}`;
const next = join(web, 'node_modules', 'next', 'dist', 'bin', 'next');
const server = spawn(process.execPath, [next, 'start', '-p', String(port), '-H', '127.0.0.1'], {
  cwd: web,
  stdio: ['ignore', 'pipe', 'pipe']
});
let log = '';
server.stdout.on('data', (chunk) => (log += chunk));
server.stderr.on('data', (chunk) => (log += chunk));

async function get(path, headers) {
  const response = await fetch(origin + path, { headers });
  return { response, type: response.headers.get('content-type') ?? '' };
}

async function waitUntilUp() {
  for (let attempt = 0; attempt < 100; attempt++) {
    if (server.exitCode !== null) throw new Error(`next start exited early:\n${log}`);
    try {
      await fetch(`${origin}/robots.txt`);
      return;
    } catch {
      await new Promise((resolve) => setTimeout(resolve, 100));
    }
  }
  throw new Error(`next start did not come up:\n${log}`);
}

const checks = [];
function check(name, fn) {
  checks.push({ name, fn });
}

/** The value of a meta tag or link in a page's head, whichever way round its attributes are. */
function head(html, attr, name, valueAttr = 'content') {
  const tags = html.match(/<(meta|link)\b[^>]*>/g) ?? '';
  for (const tag of Array.isArray(tags) ? tags : []) {
    if (!new RegExp(`\\b${attr}="${name}"`).test(tag)) continue;
    const value = tag.match(new RegExp(`\\b${valueAttr}="([^"]*)"`));
    if (value)
      return value[1]
        .replace(/&amp;/g, '&')
        .replace(/&#x27;/g, "'")
        .replace(/&quot;/g, '"');
  }
  return undefined;
}

function jsonLd(html) {
  return [...html.matchAll(/<script type="application\/ld\+json">(.*?)<\/script>/gs)].map((m) =>
    JSON.parse(m[1])
  );
}

function count(html, tag) {
  return (html.match(new RegExp(`<${tag}[\\s>]`, 'g')) ?? '').length;
}

/** Frontmatter of a generated page, read the plain way: it is two quoted fields. */
function frontmatter(file) {
  const source = readFileSync(join(web, 'content', 'docs', file), 'utf8');
  const field = (name) => {
    const match = source.match(new RegExp(`^${name}:\\s*(.*)$`, 'm'));
    return match
      ? JSON.parse(match[1].startsWith('"') ? match[1] : JSON.stringify(match[1]))
      : undefined;
  };
  return { title: field('title'), description: field('description') };
}

/** Every generated page, however deep, so a new page or folder is checked without a change here. */
function docsFiles(dir = '') {
  return readdirSync(join(web, 'content', 'docs', dir), { withFileTypes: true }).flatMap(
    (entry) => {
      const path = dir === '' ? entry.name : `${dir}/${entry.name}`;
      if (entry.isDirectory()) return docsFiles(path);
      return entry.name.endsWith('.mdx') ? [path] : [];
    }
  );
}

/** The route of a page file: `tools/rmx/force.mdx` is /docs/tools/rmx/force, an `index.mdx` is its folder. */
function docsRoute(file) {
  const slug = file.replace(/\.mdx$/, '').replace(/(^|\/)index$/, '');
  return slug === '' ? '/docs' : `/docs/${slug}`;
}

const pageFiles = docsFiles();
// A release's snapshot (content/docs/v1.1/...) is served but neither listed in
// the sitemap nor in llms.txt, and is marked noindex, so that searches lead to
// the latest docs.
const isVersioned = (file) => /^v\d+\.\d+\//.test(file);
const expectedDocs = pageFiles.filter((file) => !isVersioned(file)).map(docsRoute);
const versionedDocs = pageFiles.filter(isVersioned).map(docsRoute);

let site = '';

check('robots.txt allows crawling and names the sitemap', async () => {
  const { response, type } = await get('/robots.txt');
  assert.equal(response.status, 200);
  assert.match(type, /text\/plain/);
  const body = await response.text();
  assert.match(body, /User-Agent: \*/i);
  assert.match(body, /Allow: \//);
  assert.match(body, /Disallow: \/api\//);
  const sitemap = body.match(/Sitemap: (\S+)\/sitemap\.xml/);
  assert.ok(sitemap, 'no Sitemap line');
  site = sitemap[1];
  assert.match(site, /^https:\/\//, 'the sitemap URL is absolute');
});

check('sitemap.xml lists home, install, the playground and every docs page', async () => {
  const { response, type } = await get('/sitemap.xml');
  assert.equal(response.status, 200);
  assert.match(type, /xml/);
  const locs = [...(await response.text()).matchAll(/<loc>([^<]+)<\/loc>/g)].map((m) => m[1]);
  for (const path of ['', '/install', '/playground', ...expectedDocs]) {
    assert.ok(locs.includes(site + path), `sitemap is missing ${path || '/'}`);
  }
  assert.equal(locs.length, expectedDocs.length + 3, 'sitemap lists nothing else');
  assert.equal(new Set(locs).size, locs.length, 'no duplicates');
});

check('llms.txt indexes the docs and llms-full.txt carries them', async () => {
  const index = await get('/llms.txt');
  assert.equal(index.response.status, 200);
  assert.match(index.type, /text\/plain/);
  const indexBody = await index.response.text();
  for (const path of expectedDocs)
    assert.ok(indexBody.includes(path), `llms.txt is missing ${path}`);

  const full = await get('/llms-full.txt');
  assert.equal(full.response.status, 200);
  const fullBody = await full.response.text();
  for (const file of pageFiles.filter((f) => !isVersioned(f))) {
    assert.ok(
      fullBody.includes(`# ${frontmatter(file).title} (`),
      `llms-full.txt is missing ${file}`
    );
  }
  assert.ok(fullBody.length > indexBody.length * 3, 'llms-full.txt holds the page bodies');
});

check('a release snapshot is served, not indexed, and points at the latest page', async () => {
  const sample = versionedDocs.find((path) => path.endsWith('/tools/rmx'));
  if (!sample) return; // no snapshot yet
  const version = sample.split('/')[2];
  const { response } = await get(sample);
  assert.equal(response.status, 200);
  const html = await response.text();
  assert.match(head(html, 'name', 'robots'), /noindex/);
  assert.equal(head(html, 'rel', 'canonical', 'href'), `${site}/docs/tools/rmx`);
  // React separates the text and the version with an empty comment.
  assert.ok(
    html.replace(/<!-- -->/g, '').includes(`These are the docs of gnux ${version}.`),
    'no version notice'
  );
  assert.ok(html.includes('Documentation version'), 'no version select');

  const missing = await fetch(`${origin}/docs/${version}/no-such-page`, { redirect: 'manual' });
  assert.ok([307, 308].includes(missing.status), `a missing page answers ${missing.status}`);
  assert.equal(new URL(missing.headers.get('location'), 'http://x').pathname, `/docs/${version}`);
});

check('the manifest and the icon are served', async () => {
  const manifest = await get('/manifest.webmanifest');
  assert.equal(manifest.response.status, 200);
  const body = await manifest.response.json();
  assert.equal(body.short_name, 'gnux');
  assert.ok(Array.isArray(body.icons) && body.icons.length > 0);
  const icon = await get('/icon.svg');
  assert.equal(icon.response.status, 200);
  assert.match(icon.type, /image\/svg/);
});

check('the home page describes itself', async () => {
  const { response } = await get('/');
  assert.equal(response.status, 200);
  const html = await response.text();
  assert.match(html, /<html[^>]*\blang="en"/);
  assert.match(html, /<title>[^<]+ \| gnux<\/title>/);
  assert.ok(head(html, 'name', 'description').length > 50);
  assert.equal(head(html, 'rel', 'canonical', 'href'), site);
  assert.equal(head(html, 'property', 'og:type'), 'website');
  assert.equal(head(html, 'property', 'og:url'), site);
  assert.ok(head(html, 'property', 'og:title'));
  assert.equal(head(html, 'name', 'twitter:card'), 'summary_large_image');
  assert.match(head(html, 'name', 'robots'), /index, follow/);
  assert.match(head(html, 'rel', 'manifest', 'href'), /manifest\.webmanifest/);
  assert.equal(count(html, 'h1'), 1, 'one h1');
  assert.equal(count(html, 'main'), 1, 'one main landmark');

  const image = head(html, 'property', 'og:image');
  assert.ok(image, 'no og:image');
  const og = await get(new URL(image).pathname + new URL(image).search);
  assert.equal(og.response.status, 200);
  assert.match(og.type, /image\/png/);

  const app = jsonLd(html).find((item) => item['@type'] === 'SoftwareApplication');
  assert.ok(app, 'no SoftwareApplication JSON-LD');
  assert.equal(app.name, 'gnux');
  assert.equal(app.url, site);
  assert.match(app.codeRepository, /github\.com\/braswelljr\/gnux/);
});

check('the playground describes itself', async () => {
  const { response } = await get('/playground');
  assert.equal(response.status, 200);
  const html = await response.text();
  assert.match(html, /<title>Playground \| gnux<\/title>/);
  assert.match(head(html, 'name', 'description'), /browser/);
  assert.equal(head(html, 'rel', 'canonical', 'href'), `${site}/playground`);
  assert.equal(head(html, 'property', 'og:url'), `${site}/playground`);
  assert.equal(head(html, 'property', 'og:image'), `${site}/opengraph-image`);
  assert.equal(count(html, 'h1'), 1, 'one h1');
});

for (const [path, file] of [
  ['/docs', 'index.mdx'],
  ['/docs/guide', 'guide.mdx'],
  ['/docs/tools/rmx', 'tools/rmx/index.mdx'],
  ['/docs/tools/rmx/force', 'tools/rmx/force.mdx']
]) {
  check(`${path} takes its head from its frontmatter`, async () => {
    const { title, description } = frontmatter(file);
    const { response } = await get(path);
    assert.equal(response.status, 200);
    const html = await response.text();
    assert.ok(html.includes(`<title>${title} | gnux</title>`), `title is not "${title} | gnux"`);
    assert.equal(head(html, 'name', 'description'), description);
    assert.equal(head(html, 'rel', 'canonical', 'href'), site + path);
    assert.equal(head(html, 'property', 'og:type'), 'article');
    assert.equal(head(html, 'property', 'og:title'), title);
    assert.equal(head(html, 'property', 'og:url'), site + path);
    assert.equal(count(html, 'h1'), 1, 'one h1');
    assert.equal(count(html, 'main'), 1, 'one main landmark');

    const image = head(html, 'property', 'og:image');
    assert.ok(image.startsWith(`${site}/og/docs/`), `og:image is ${image}`);
    assert.equal(head(html, 'name', 'twitter:image'), image);
    const og = await get(new URL(image).pathname);
    assert.equal(og.response.status, 200);
    assert.match(og.type, /image\/png/);

    const data = jsonLd(html);
    const article = data.find((item) => item['@type'] === 'TechArticle');
    assert.ok(article, 'no TechArticle JSON-LD');
    assert.equal(article.headline, title);
    assert.equal(article.description, description);
    assert.equal(article.url, site + path);
    const crumbs = data.find((item) => item['@type'] === 'BreadcrumbList');
    assert.ok(crumbs, 'no BreadcrumbList JSON-LD');
    const items = crumbs.itemListElement;
    assert.equal(items[0].item, site);
    assert.equal(items[1].item, `${site}/docs`);
    assert.equal(items.at(-1).item, site + path);
    items.forEach((item, i) => assert.equal(item.position, i + 1));
  });
}

check('a missing page is a 404 that asks not to be indexed', async () => {
  const { response } = await get('/docs/no-such-page');
  assert.equal(response.status, 404);
  assert.match(head(await response.text(), 'name', 'robots'), /noindex/);
});

check('gnux.wasm is served as application/wasm', async () => {
  const { response, type } = await get('/gnux.wasm');
  assert.equal(response.status, 200);
  assert.match(type, /^application\/wasm/);
  await response.body?.cancel();
  const loader = await get('/wasm_exec.js');
  assert.equal(loader.response.status, 200);
  assert.match(loader.type, /javascript/);
});

check('search answers from the docs', async () => {
  const { response } = await get('/api/search?query=recursive');
  assert.equal(response.status, 200);
  const results = await response.json();
  assert.ok(Array.isArray(results) && results.length > 0, 'no search results');
});

let failed = 0;
try {
  await waitUntilUp();
  for (const { name, fn } of checks) {
    try {
      await fn();
      console.log(`ok    ${name}`);
    } catch (err) {
      failed++;
      console.log(`FAIL  ${name}\n      ${String(err.message).split('\n').join('\n      ')}`);
    }
  }
} finally {
  server.kill();
}

console.log(
  failed === 0
    ? `\ncheck-seo: ${checks.length} checks passed`
    : `\ncheck-seo: ${failed} of ${checks.length} checks failed`
);
process.exit(failed === 0 ? 0 : 1);
