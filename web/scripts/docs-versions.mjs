// The docs of each stable release, for the version picker: every release
// carries its own docs as gnux-docs-vX.Y.Z.tar.gz (made by tools/docsnap when
// the release is built, and listed in its checksums.txt). This downloads the
// archive of the latest patch of each minor version, checks it, and unpacks it
// into content/docs/vX.Y. Nothing of it is committed.
//
// One GitHub API call lists the releases; the archives come from the
// releases' download URLs, which the API's rate limit does not count. Set
// GITHUB_TOKEN (or GH_TOKEN) where many builds share an address, as on Vercel.
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, readdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

const stable = /^v(\d+)\.(\d+)\.(\d+)$/;

/** The asset name of a release's docs. */
export const archiveName = (tag) => `gnux-docs-${tag}.tar.gz`;

/**
 * The release to take for each minor version: the highest patch of the
 * stable, published releases, newest minor first. `releases` is the GitHub
 * API's list.
 */
export function pickReleases(releases) {
  const best = new Map();
  for (const release of Array.isArray(releases) ? releases : []) {
    if (!release || typeof release !== 'object' || release.draft || release.prerelease) continue;
    const match = typeof release.tag_name === 'string' ? stable.exec(release.tag_name) : null;
    if (!match) continue;
    const [major, minor, patch] = match.slice(1).map(Number);
    const key = `v${major}.${minor}`;
    const seen = best.get(key);
    if (!seen || patch > seen.patch) best.set(key, { version: key, major, minor, patch, release });
  }
  return [...best.values()].sort((a, b) => b.major - a.major || b.minor - a.minor);
}

/** The SHA-256 a checksums file (sha256sum's format) gives for name, or null. */
export function checksumFor(text, name) {
  for (const line of text.split('\n')) {
    const match = /^([0-9a-f]{64})\s+\*?(.+)$/.exec(line.trim());
    if (match && match[2] === name) return match[1];
  }
  return null;
}

async function fetchOk(url, headers) {
  const response = await fetch(url, { headers, redirect: 'follow' });
  if (!response.ok) throw new Error(`${url}: HTTP ${response.status}`);
  return response;
}

/**
 * Downloads, checks and unpacks the docs of every release into docsDir.
 * Returns the versions unpacked and the problems met; it throws only when the
 * releases cannot be listed at all.
 */
export async function fetchDocsVersions({ repo, docsDir, cacheDir, token, log = console.log }) {
  const headers = { Accept: 'application/vnd.github+json', 'User-Agent': 'gnux-website-build' };
  const auth = token ? { ...headers, Authorization: `Bearer ${token}` } : headers;
  const list = await fetchOk(`https://api.github.com/repos/${repo}/releases?per_page=100`, auth);
  const picked = pickReleases(await list.json());

  const unpacked = [];
  const problems = [];
  mkdirSync(cacheDir, { recursive: true });
  for (const { version, release } of picked) {
    const tag = release.tag_name;
    const assets = Array.isArray(release.assets) ? release.assets : [];
    const asset = (name) => assets.find((a) => a && a.name === name);
    const archive = asset(archiveName(tag));
    if (!archive) {
      // Releases before docs travelled with them have none until one is uploaded.
      problems.push(`${tag} has no ${archiveName(tag)}`);
      continue;
    }
    try {
      // The signed checksums.txt when it lists the archive, else the archive's
      // own .sha256 (uploaded with archives added to a release afterwards).
      let want = null;
      for (const name of ['checksums.txt', `${archiveName(tag)}.sha256`]) {
        const sums = asset(name);
        if (!sums) continue;
        const text = await (await fetchOk(sums.browser_download_url, headers)).text();
        want = checksumFor(text, archiveName(tag));
        if (want) break;
      }
      if (!want) throw new Error(`no checksum for ${archiveName(tag)}`);

      const file = join(cacheDir, archiveName(tag));
      const sha = (data) => createHash('sha256').update(data).digest('hex');
      let data = existsSync(file) ? readFileSync(file) : null;
      if (!data || sha(data) !== want) {
        log(`gen: downloading the ${version} docs (${tag})`);
        data = Buffer.from(
          await (await fetchOk(archive.browser_download_url, headers)).arrayBuffer()
        );
        if (sha(data) !== want) throw new Error(`${archiveName(tag)} does not match its checksum`);
        writeFileSync(file, data);
      }

      const listing = spawnSync('tar', ['-tzf', file], { encoding: 'utf8' });
      const entries = listing.status === 0 ? listing.stdout.split('\n').filter(Boolean) : [];
      if (
        entries.length === 0 ||
        entries.some((e) => e !== `${version}/` && !e.startsWith(`${version}/`))
      )
        throw new Error(`${archiveName(tag)} does not hold ${version}/ alone`);
      rmSync(join(docsDir, version), { recursive: true, force: true });
      const untar = spawnSync('tar', ['-xzf', file, '-C', docsDir], { stdio: 'inherit' });
      if (untar.status !== 0) throw new Error(`unpacking ${archiveName(tag)} failed`);
      unpacked.push(version);
    } catch (err) {
      problems.push(`${tag}: ${err instanceof Error ? err.message : err}`);
    }
  }

  // A version whose release is gone, or was replaced by a newer patch, goes too.
  if (existsSync(docsDir)) {
    for (const entry of readdirSync(docsDir, { withFileTypes: true })) {
      if (entry.isDirectory() && /^v\d+\.\d+$/.test(entry.name) && !unpacked.includes(entry.name)) {
        if (picked.some((p) => p.version === entry.name)) continue; // kept from an earlier build
        rmSync(join(docsDir, entry.name), { recursive: true, force: true });
      }
    }
  }
  return { unpacked, problems };
}
