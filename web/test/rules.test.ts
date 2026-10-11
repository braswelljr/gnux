/**
 * The house rules that a pattern can check, checked. A rule nothing enforces
 * decays, so each of these walks the source and fails on the shape it bans.
 * The rules themselves live in ../docs/rules/.
 */
import assert from 'node:assert/strict';
import { readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';
import { describe, test } from 'node:test';
import { fileURLToPath } from 'node:url';

const web = fileURLToPath(new URL('..', import.meta.url));

/** Not ours: dependencies, build output, and what the Go toolchain generates. */
const SKIP = new Set(['node_modules', '.next', '.source', 'out', 'build', 'coverage']);
const GENERATED = ['content/docs/', 'public/gnux.wasm', 'public/wasm_exec.js', 'pnpm-lock.yaml'];

function walk(dir: string, found: string[] = []): string[] {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (SKIP.has(entry.name)) continue;
    const path = join(dir, entry.name);
    if (entry.isDirectory()) walk(path, found);
    else found.push(relative(web, path).split('\\').join('/'));
  }
  return found;
}

const files = walk(web).filter((file) => !GENERATED.some((g) => file.startsWith(g)));
const text = files.filter((file) =>
  /\.(ts|tsx|js|mjs|css|md|mdx|json|yaml|yml|svg)$|^\.[a-z]+$/.test(file)
);
const source = text.filter((file) => /^(src|lib)\/.*\.(ts|tsx)$/.test(file));
const read = (file: string) => readFileSync(join(web, file), 'utf8');

/** Every `file:line` where `pattern` matches. */
function matches(among: string[], pattern: RegExp): string[] {
  const hits: string[] = [];
  for (const file of among) {
    read(file)
      .split('\n')
      .forEach((line, i) => {
        if (pattern.test(line)) hits.push(`${file}:${i + 1}`);
      });
  }
  return hits;
}

describe('house rules', () => {
  test('the walk found the source', () => {
    assert.ok(source.length > 20, `only ${source.length} source files found`);
  });

  test('no em dashes in anything we write', () => {
    assert.deepEqual(matches(text, /\u2014/), []);
  });

  test('an array fallback is a type check, never ?? [] or || []', () => {
    assert.deepEqual(matches(source, /(\?\?|\|\|)\s*\[\s*\]/), []);
  });

  test('type sizes come from the scale, never text-[...]', () => {
    assert.deepEqual(matches(source, /\btext-\[[\d.]+(px|rem|em)\]/), []);
  });

  test('copying goes through useCopyToClipboard', () => {
    const outside = source.filter((file) => file !== 'src/hooks/use-copy-to-clipboard.ts');
    assert.deepEqual(matches(outside, /navigator\??\.clipboard/), []);
  });

  test('icons come from lucide-react only', () => {
    assert.deepEqual(
      matches(source, /from '(react-icons|iconsax-react|@radix-ui\/react-icons|@heroicons)/),
      []
    );
  });

  test('every page builds its metadata with createMetadata', () => {
    const pages = source.filter((file) => /^src\/app\/.*(page|layout|not-found)\.tsx$/.test(file));
    assert.ok(pages.length >= 5);
    for (const file of pages) {
      const body = read(file);
      const describes = /export (const metadata|(async )?function generateMetadata)/.test(body);
      // A layout that only arranges its children describes nothing itself.
      if (!describes && /layout\.tsx$/.test(file)) continue;
      assert.ok(describes, `${file} exports no metadata`);
      assert.ok(body.includes('createMetadata('), `${file} does not use createMetadata`);
      assert.ok(!/:\s*Metadata\s*=\s*\{/.test(body), `${file} writes a Metadata literal`);
    }
  });

  test('the boundary guards narrow what they claim to', async () => {
    const { arrayOf, isNumber, isRecord, isString, prop } = await import('../lib/guards.ts');
    assert.equal(isRecord([]), false);
    assert.equal(isRecord(null), false);
    assert.equal(isNumber(Number.NaN), false);
    assert.deepEqual(arrayOf(['a', 1, 'b', null], isString), ['a', 'b']);
    assert.deepEqual(arrayOf('not an array', isString), []);
    assert.deepEqual(arrayOf(undefined, isString), []);
    assert.equal(prop({ a: 1 }, 'a'), 1);
    assert.equal(prop('text', 'length'), undefined);
  });
});
