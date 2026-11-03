import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import {
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  symlinkSync,
  writeFileSync
} from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, describe, test } from 'node:test';
import {
  archiveName,
  checksumFor,
  fetchDocsVersions,
  pickReleases,
  unsafeEntry
} from '../scripts/docs-versions.mjs';

const realFetch = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = realFetch;
});

const release = (tag: string, extra: Record<string, unknown> = {}) => ({
  tag_name: tag,
  draft: false,
  prerelease: false,
  assets: [],
  ...extra
});

describe('pickReleases', () => {
  test('takes the newest release of each older major version, stable only', () => {
    const picked = pickReleases([
      release('v1.1.0'),
      release('v2.0.0'),
      release('v1.2.0'),
      release('v1.1.3'),
      release('v0.0.1'),
      release('v0.0.2'),
      release('v2.1.0-rc.1', { prerelease: true }),
      release('v3.0.0', { draft: true }),
      release('latest'),
      null
    ]);
    // v2 is the newest major version: the latest docs are its docs.
    assert.deepEqual(
      picked.map((p: { version: string; release: { tag_name: string } }) => [
        p.version,
        p.release.tag_name
      ]),
      [
        ['v1', 'v1.2.0'],
        ['v0', 'v0.0.2']
      ]
    );
  });

  test('a single major version picks nothing', () => {
    assert.deepEqual(pickReleases([release('v1.0.0'), release('v1.2.0')]), []);
  });

  test('anything but a list picks nothing', () => {
    assert.deepEqual(pickReleases({ message: 'rate limited' }), []);
  });
});

test('checksumFor reads sha256sum lines', () => {
  const sum = 'a'.repeat(64);
  const text = `${'b'.repeat(64)}  other.tar.gz\n${sum}  gnux-docs-v1.2.0.tar.gz\n`;
  assert.equal(checksumFor(text, 'gnux-docs-v1.2.0.tar.gz'), sum);
  assert.equal(checksumFor(text, 'missing'), null);
});

/** A docs archive as tools/docsnap packs it: one folder at the top. */
function makeArchive(dir: string, folder: string): Buffer {
  mkdirSync(join(dir, folder, 'tools'), { recursive: true });
  writeFileSync(
    join(dir, folder, 'index.mdx'),
    `# ${folder}\n\nSee [rmx](/docs/${folder}/tools/rmx).\n`
  );
  writeFileSync(join(dir, folder, 'meta.json'), `{"title": "${folder}", "root": true}\n`);
  writeFileSync(join(dir, folder, 'tools', 'rmx.mdx'), 'rm\n');
  const file = join(dir, `${folder}.tar.gz`);
  assert.equal(spawnSync('tar', ['-czf', file, '-C', dir, folder]).status, 0);
  return readFileSync(file);
}

describe('fetchDocsVersions', () => {
  test('downloads, checks and unpacks each older major version, and reports the rest', async () => {
    const work = mkdtempSync(join(tmpdir(), 'docs-versions-'));
    const docsDir = join(work, 'docs');
    mkdirSync(join(docsDir, 'v1.1'), { recursive: true }); // left by a per-minor build
    mkdirSync(join(docsDir, 'v4'), { recursive: true }); // the newest major: the latest docs
    const v2 = new Uint8Array(makeArchive(work, 'v2'));
    const v12 = new Uint8Array(makeArchive(work, 'v1.2')); // made before 1.4.0
    const sha = (b: Uint8Array) => createHash('sha256').update(b).digest('hex');
    const files: Record<string, Uint8Array<ArrayBuffer> | string> = {
      'https://dl/v2/archive': v2,
      'https://dl/v2/checksums.txt': `${sha(v2)}  ${archiveName('v2.1.0')}\n`,
      'https://dl/v1/archive': v12,
      'https://dl/v1/sha256': `${sha(v12)}  ${archiveName('v1.2.0')}\n`,
      'https://dl/v3/archive': 'not the archive',
      'https://dl/v3/sha256': `${'0'.repeat(64)}  ${archiveName('v3.0.0')}\n`
    };
    const withArchive = (tag: string, key: string, sums: string) =>
      release(tag, {
        assets: [
          { name: archiveName(tag), browser_download_url: `https://dl/${key}/archive` },
          {
            name: sums === 'checksums.txt' ? sums : `${archiveName(tag)}.sha256`,
            browser_download_url: `https://dl/${key}/${sums === 'checksums.txt' ? sums : 'sha256'}`
          }
        ]
      });
    const releases = [
      release('v4.0.0'),
      withArchive('v3.0.0', 'v3', 'sha256'),
      withArchive('v2.1.0', 'v2', 'checksums.txt'),
      withArchive('v1.2.0', 'v1', 'sha256'),
      release('v1.1.0'),
      release('v0.0.2')
    ];
    let authorization: string | null = null;
    globalThis.fetch = (async (input: string | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes('api.github.com')) {
        authorization = new Headers(init?.headers).get('authorization');
        return new Response(JSON.stringify(releases));
      }
      const body = files[url];
      return body === undefined ? new Response('', { status: 404 }) : new Response(body);
    }) as typeof fetch;

    const { unpacked, problems } = await fetchDocsVersions({
      repo: 'o/r',
      docsDir,
      cacheDir: join(work, 'cache'),
      token: 'secret',
      log: () => {}
    });
    assert.deepEqual(unpacked, ['v2', 'v1']);
    assert.equal(authorization, 'Bearer secret');
    assert.equal(
      readFileSync(join(docsDir, 'v2', 'index.mdx'), 'utf8'),
      '# v2\n\nSee [rmx](/docs/v2/tools/rmx).\n'
    );
    // The v1.2 archive is filed as v1, its links and title with it.
    assert.equal(
      readFileSync(join(docsDir, 'v1', 'index.mdx'), 'utf8'),
      '# v1.2\n\nSee [rmx](/docs/v1/tools/rmx).\n'
    );
    assert.equal(
      readFileSync(join(docsDir, 'v1', 'meta.json'), 'utf8'),
      '{"title": "v1", "root": true}\n'
    );
    assert.ok(existsSync(join(docsDir, 'v1', 'tools', 'rmx.mdx')));
    assert.ok(!existsSync(join(docsDir, 'v1.2')), 'the archive folder was left behind');
    assert.ok(!existsSync(join(docsDir, 'v1.1')), 'a per-minor folder was not removed');
    assert.ok(!existsSync(join(docsDir, 'v4')), 'the newest major version was kept');
    assert.ok(!existsSync(join(docsDir, 'v3')), 'a corrupt archive was unpacked');
    assert.deepEqual(problems, [
      `v3.0.0: ${archiveName('v3.0.0')} does not match its checksum`,
      `v0.0.2 has no ${archiveName('v0.0.2')}`
    ]);
  });
});

describe('unsafeEntry', () => {
  const list = (file: string) =>
    spawnSync('tar', ['-tzf', file], { encoding: 'utf8' }).stdout.split('\n').filter(Boolean);

  test('a plain archive is safe', () => {
    const work = mkdtempSync(join(tmpdir(), 'docs-unsafe-'));
    makeArchive(work, 'v1');
    const file = join(work, 'v1.tar.gz');
    assert.equal(unsafeEntry(file, list(file)), null);
  });

  test('a path that climbs out is refused', () => {
    const work = mkdtempSync(join(tmpdir(), 'docs-unsafe-'));
    assert.equal(
      unsafeEntry(join(work, 'none'), ['v1/', 'v1/../../escape.mdx']),
      'v1/../../escape.mdx'
    );
    assert.equal(unsafeEntry(join(work, 'none'), ['/etc/passwd']), '/etc/passwd');
    assert.equal(unsafeEntry(join(work, 'none'), ['C:/x']), 'C:/x');
  });

  test('a symbolic link is refused', () => {
    const work = mkdtempSync(join(tmpdir(), 'docs-unsafe-'));
    mkdirSync(join(work, 'v1'));
    symlinkSync('/etc', join(work, 'v1', 'link'));
    const file = join(work, 'link.tar.gz');
    assert.equal(spawnSync('tar', ['-czf', file, '-C', work, 'v1']).status, 0);
    const found = unsafeEntry(file, list(file));
    assert.ok(found && found.startsWith('l'), `a link was not reported: ${found}`);
  });
});
