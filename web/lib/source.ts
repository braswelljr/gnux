import { llms, loader, type InferPageType } from 'fumadocs-core/source';
import { docs as docsContent } from '@content-source/server';

export const docsRoute = '/docs';

export const docs = loader({
  baseUrl: docsRoute,
  source: docsContent.toFumadocsSource()
});

export type DocsPage = InferPageType<typeof docs>;

/** Builds llms.txt (an index) and llms-full.txt (every page) from the docs. */
export const docsLlms = llms(docs, {
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
