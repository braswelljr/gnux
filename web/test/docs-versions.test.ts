import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, describe, test } from 'node:test';
import {
  archiveName,
  checksumFor,
  fetchDocsVersions,
  pickReleases
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
  test('takes the highest patch of each minor, newest first, stable only', () => {
    const picked = pickReleases([
      release('v1.1.0'),
      release('v1.2.0'),
      release('v1.1.3'),
      release('v1.3.0-rc.1', { prerelease: true }),
      release('v1.4.0', { draft: true }),
      release('latest'),
      null
    ]);
    assert.deepEqual(
      picked.map((p: { release: { tag_name: string } }) => p.release.tag_name),
      ['v1.2.0', 'v1.1.3']
    );
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

/** A docs archive as tools/docsnap packs it: vX.Y/ at the top. */
function makeArchive(dir: string, version: string): Buffer {
  mkdirSync(join(dir, version, 'tools'), { recursive: true });
  writeFileSync(join(dir, version, 'index.mdx'), `# ${version}\n`);
  writeFileSync(join(dir, version, 'tools', 'rmx.mdx'), 'rm\n');
  const file = join(dir, `${version}.tar.gz`);
  assert.equal(spawnSync('tar', ['-czf', file, '-C', dir, version]).status, 0);
  return readFileSync(file);
}

describe('fetchDocsVersions', () => {
  test('downloads, checks and unpacks each release, and reports the rest', async () => {
    const work = mkdtempSync(join(tmpdir(), 'docs-versions-'));
    const docsDir = join(work, 'docs');
    mkdirSync(join(docsDir, 'v0.9'), { recursive: true }); // a release that is gone
    const v12 = new Uint8Array(makeArchive(work, 'v1.2'));
    const v11 = new Uint8Array(makeArchive(work, 'v1.1'));
    const sha = (b: Uint8Array) => createHash('sha256').update(b).digest('hex');
    const files: Record<string, Uint8Array<ArrayBuffer> | string> = {
      'https://dl/v1.2/archive': v12,
      'https://dl/v1.2/checksums.txt': `${sha(v12)}  ${archiveName('v1.2.0')}\n`,
      'https://dl/v1.1/archive': v11,
      'https://dl/v1.1/sha256': `${sha(v11)}  ${archiveName('v1.1.0')}\n`,
      'https://dl/v1.0/archive': 'not the archive',
      'https://dl/v1.0/sha256': `${'0'.repeat(64)}  ${archiveName('v1.0.0')}\n`
    };
    const releases = [
      release('v1.2.0', {
        assets: [
          { name: archiveName('v1.2.0'), browser_download_url: 'https://dl/v1.2/archive' },
          { name: 'checksums.txt', browser_download_url: 'https://dl/v1.2/checksums.txt' }
        ]
      }),
      release('v1.1.0', {
        assets: [
          { name: archiveName('v1.1.0'), browser_download_url: 'https://dl/v1.1/archive' },
          {
            name: `${archiveName('v1.1.0')}.sha256`,
            browser_download_url: 'https://dl/v1.1/sha256'
          }
        ]
      }),
      release('v1.0.0', {
        assets: [
          { name: archiveName('v1.0.0'), browser_download_url: 'https://dl/v1.0/archive' },
          {
            name: `${archiveName('v1.0.0')}.sha256`,
            browser_download_url: 'https://dl/v1.0/sha256'
          }
        ]
      }),
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
    assert.deepEqual(unpacked, ['v1.2', 'v1.1']);
    assert.equal(authorization, 'Bearer secret');
    assert.equal(readFileSync(join(docsDir, 'v1.2', 'index.mdx'), 'utf8'), '# v1.2\n');
    assert.ok(existsSync(join(docsDir, 'v1.1', 'tools', 'rmx.mdx')));
    assert.ok(!existsSync(join(docsDir, 'v0.9')), 'the gone release was not removed');
    assert.ok(!existsSync(join(docsDir, 'v1.0')), 'a corrupt archive was unpacked');
    assert.deepEqual(problems, [
      `v1.0.0: ${archiveName('v1.0.0')} does not match its checksum`,
      `v0.0.2 has no ${archiveName('v0.0.2')}`
    ]);
  });
});
