import { llms, loader, type InferPageType } from 'fumadocs-core/source';
import { docs as docsContent } from '@content-source/server';

export const docsRoute = '/docs';

export const docs = loader({
  baseUrl: docsRoute,
  source: docsContent.toFumadocsSource()
});

export type DocsPage = InferPageType<typeof docs>;

/**
 * A release's snapshot folder (`v1.1`): the docs of that version as it
 * shipped, written by tools/docsnap beside the latest ones.
 */
const versionFolder = /^v(\d+)\.(\d+)$/;

/** The release a page belongs to, or null for the latest docs. */
export function pageVersion(slugs: string[]): string | null {
  const first = slugs[0];
  return first !== undefined && versionFolder.test(first) ? first : null;
}

/** The released versions with docs of their own, newest first. */
export function docsVersions(): string[] {
  const versions = new Set<string>();
  for (const page of docs.getPages()) {
    const version = pageVersion(page.slugs);
    if (version) versions.add(version);
  }
  const key = (v: string) => {
    const match = versionFolder.exec(v);
    return match ? match.slice(1).map(Number) : [];
  };
  return [...versions].sort((a, b) => {
    const [aMajor = 0, aMinor = 0] = key(a);
    const [bMajor = 0, bMinor = 0] = key(b);
    return bMajor - aMajor || bMinor - aMinor;
  });
}

/**
 * The docs with the latest pages only, for search and llms.txt: an older
 * version's page would otherwise compete with the current one.
 */
export const latestDocs: typeof docs = {
  ...docs,
  getPages: (...args: Parameters<typeof docs.getPages>) =>
    docs.getPages(...args).filter((page) => pageVersion(page.slugs) === null)
};

/** Builds llms.txt (an index) and llms-full.txt (every page) from the docs. */
export const docsLlms = llms(latestDocs, {
  renderPage: async (page) => `# ${page.data.title} (${page.url})

${page.data.description ?? ''}

${await page.data.getText('processed')}`
});

/** The generated Open Graph image of a docs page. */
export function getPageImage(page: { slugs: string[] }) {
  const segments = [...page.slugs, 'image.png'];

  return {
    segments,
    url: `/og/docs/${segments.join('/')}`
  };
}

/** A tool as the landing page lists it. */
export type ToolSummary = { name: string; gnu: string; description: string; url: string };

/**
 * Every tool, read from the generated reference: one page per registered tool
 * under `tools/`, so this follows `cmd.Tools()` without a list of its own.
 */
export function getTools(): ToolSummary[] {
  return docs
    .getPages()
    .filter((page) => page.slugs.length === 2 && page.slugs[0] === 'tools')
    .map((page) => ({
      name: page.data.title,
      // Every tool is its GNU name with an x appended; the registry enforces it.
      gnu: page.data.title.replace(/x$/, ''),
      description: page.data.description ?? '',
      url: page.url
    }))
    .sort((a, b) => a.name.localeCompare(b.name));
}
