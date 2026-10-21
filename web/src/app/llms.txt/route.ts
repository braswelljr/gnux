import { yamlReferenceSection } from 'lib/reference';
import { docsLlms } from 'lib/source';
import { siteConfig } from '@/config/site';

export const revalidate = false;

export async function GET() {
  const body = (await docsLlms.index()) + yamlReferenceSection(siteConfig.url);
  return new Response(body, {
    headers: { 'Content-Type': 'text/plain; charset=utf-8' }
  });
}
