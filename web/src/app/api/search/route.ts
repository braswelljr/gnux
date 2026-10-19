import { createFromSource } from 'fumadocs-core/search/server';
import { latestDocs } from 'lib/source';

// The latest docs only: an older version's page would compete with the current one.
export const { GET } = createFromSource(latestDocs);
