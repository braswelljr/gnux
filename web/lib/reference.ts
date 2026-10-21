import { existsSync, readdirSync } from 'node:fs';
import { join, relative } from 'node:path';

/**
 * The YAML reference the build writes to public/reference/yaml (scripts/gen.mjs):
 * one file per page, the same pages as the docs, for scripts and AI tools that
 * read the reference as data.
 */
const yamlRoot = join(process.cwd(), 'public', 'reference', 'yaml');

/** The site path of every YAML page, gnux's own first, then in name order. */
export function yamlReferencePaths(): string[] {
  if (!existsSync(yamlRoot)) return [];
  const found: string[] = [];
  const walk = (dir: string) => {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      const path = join(dir, entry.name);
      if (entry.isDirectory()) walk(path);
      else if (entry.name.endsWith('.yaml'))
        found.push(relative(yamlRoot, path).split('\\').join('/'));
    }
  };
  walk(yamlRoot);
  const rank = (p: string) => (p === 'index.yaml' ? 0 : p === 'guide.yaml' ? 1 : 2);
  return found
    .sort((a, b) => rank(a) - rank(b) || a.localeCompare(b))
    .map((p) => `/reference/yaml/${p}`);
}

/** The section llms.txt ends with: where the YAML reference is. */
export function yamlReferenceSection(origin: string): string {
  const paths = yamlReferencePaths();
  if (paths.length === 0) return '';
  const lines = paths.map((p) => `- [${p.replace('/reference/yaml/', '')}](${origin}${p})`);
  return `\n\n## Reference as YAML\n\nThe same reference as structured YAML, one file per page: each tool's name, synopsis, usage, options and examples.\n\n${lines.join('\n')}\n`;
}
