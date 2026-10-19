import { docs, docsVersions, pageVersion } from 'lib/source';

// Built once: the pages change only with a deploy.
export const dynamic = 'force-static';

/**
 * Each documentation version's pages, as paths within it ('' for its first
 * page, '/tools/rmx' for a tool), for the version select to know whether the
 * page being read exists in the version picked.
 */
export function GET() {
  const paths: Record<string, string[]> = { latest: [] };
  for (const v of docsVersions()) paths[v] = [];
  for (const page of docs.getPages()) {
    const version = pageVersion(page.slugs);
    const slugs = version ? page.slugs.slice(1) : page.slugs;
    paths[version ?? 'latest']?.push(slugs.length > 0 ? '/' + slugs.join('/') : '');
  }
  return Response.json(paths);
}
