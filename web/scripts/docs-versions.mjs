// The docs of each older major version, for the version picker: every release
// carries its own docs as gnux-docs-vX.Y.Z.tar.gz (made by tools/docsnap when
// the release is built, and listed in its checksums.txt). The site keeps one
// set per major version, the newest release's: the latest docs stand for the
// newest major version, and each older one (v0, ...) gets the archive of its
// last release, checked and unpacked into content/docs/vX. An archive made
// before 1.4.0 holds a vX.Y folder with links into /docs/vX.Y; it is filed
// under vX with its links rewritten. Nothing of it is committed.
//
// One GitHub API call lists the releases; the archives come from the
// releases' download URLs, which the API's rate limit does not count. Set
// GITHUB_TOKEN (or GH_TOKEN) where many builds share an address, as on Vercel.
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import {
  existsSync,
  mkdirSync,
  readdirSync,
  readFileSync,
  renameSync,
  rmSync,
  writeFileSync
} from 'node:fs';
import { join } from 'node:path';

const stable = /^v(\d+)\.(\d+)\.(\d+)$/;

/** The asset name of a release's docs. */
export const archiveName = (tag) => `gnux-docs-${tag}.tar.gz`;

/**
 * The release to take for each older major version: the newest stable,
 * published release of every major version but the newest, whose docs are the
 * latest docs themselves; newest first. `releases` is the GitHub API's list.
 */
export function pickReleases(releases) {
  const best = new Map();
  for (const release of Array.isArray(releases) ? releases : []) {
    if (!release || typeof release !== 'object' || release.draft || release.prerelease) continue;
    const match = typeof release.tag_name === 'string' ? stable.exec(release.tag_name) : null;
    if (!match) continue;
    const [major, minor, patch] = match.slice(1).map(Number);
    const key = `v${major}`;
    const seen = best.get(key);
    if (!seen || minor > seen.minor || (minor === seen.minor && patch > seen.patch))
      best.set(key, { version: key, major, minor, patch, release });
  }
  const picked = [...best.values()].sort((a, b) => b.major - a.major);
  return picked.slice(1);
}

/**
 * The first entry of the archive at file that could write outside the folder
 * it is unpacked in, or null: a path that is absolute or climbs out with
 * `..`, and anything but a plain file or directory (a link, a device), since a
 * link unpacked first would let a later file land wherever it points. The
 * checksum proves the archive is the one the release published, not that the
 * release made it right, so this holds even for a checked archive.
 */
export function unsafeEntry(file, entries) {
  for (const e of entries) {
    if (e.startsWith('/') || /^[A-Za-z]:/.test(e) || e.split(/[\\/]/).includes('..')) return e;
  }
  const verbose = spawnSync('tar', ['-tvzf', file], { encoding: 'utf8' });
  if (verbose.status !== 0) return file;
  for (const line of verbose.stdout.split('\n').filter(Boolean)) {
    if (line[0] !== '-' && line[0] !== 'd') return line;
  }
  return null;
}

/** The folder names an archive of release major.minor may hold its docs in. */
const archiveFolders = (major, minor) => [`v${major}`, `v${major}.${minor}`];

/**
 * Rewrites the links and the root title of docs unpacked from a vX.Y folder so
 * that they live under vX: every file below dir that mentions /docs/vX.Y.
 */
export function refile(dir, from, to) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) {
      refile(path, from, to);
      continue;
    }
    if (!/\.(mdx|json)$/.test(entry.name)) continue;
    let text = readFileSync(path, 'utf8');
    const next = text
      .replaceAll(`/docs/${from}/`, `/docs/${to}/`)
      .replaceAll(`/docs/${from})`, `/docs/${to})`)
      .replaceAll(`/docs/${from}#`, `/docs/${to}#`)
      .replaceAll(`"title": "${from}"`, `"title": "${to}"`)
      .replaceAll(`"title":"${from}"`, `"title":"${to}"`);
    if (next !== text) {
      text = next;
      writeFileSync(path, text);
    }
  }
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
  for (const { version, major, minor, release } of picked) {
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
      const folder = archiveFolders(major, minor).find(
        (f) => entries.length > 0 && entries.every((e) => e === `${f}/` || e.startsWith(`${f}/`))
      );
      if (!folder) throw new Error(`${archiveName(tag)} does not hold ${version}/ alone`);
      const unsafe = unsafeEntry(file, entries);
      if (unsafe) throw new Error(`${archiveName(tag)} holds an unsafe entry: ${unsafe}`);

      // Unpacked aside first, so that a failure leaves the docs as they were.
      const staging = join(cacheDir, 'unpack');
      rmSync(staging, { recursive: true, force: true });
      mkdirSync(staging, { recursive: true });
      const untar = spawnSync('tar', ['-xzf', file, '-C', staging], { stdio: 'inherit' });
      if (untar.status !== 0) throw new Error(`unpacking ${archiveName(tag)} failed`);
      if (folder !== version) refile(join(staging, folder), folder, version);
      rmSync(join(docsDir, version), { recursive: true, force: true });
      renameSync(join(staging, folder), join(docsDir, version));
      rmSync(staging, { recursive: true, force: true });
      unpacked.push(version);
    } catch (err) {
      problems.push(`${tag}: ${err instanceof Error ? err.message : err}`);
    }
  }

  // A version whose release is gone or replaced, the folder of the newest
  // major version (the latest docs are that), and a vX.Y folder left by an
  // earlier build, when versions were kept per minor release, all go.
  if (existsSync(docsDir)) {
    for (const entry of readdirSync(docsDir, { withFileTypes: true })) {
      if (
        entry.isDirectory() &&
        /^v\d+(\.\d+)?$/.test(entry.name) &&
        !unpacked.includes(entry.name)
      ) {
        if (picked.some((p) => p.version === entry.name)) continue; // kept from an earlier build
        rmSync(join(docsDir, entry.name), { recursive: true, force: true });
      }
    }
  }
  return { unpacked, problems };
}
