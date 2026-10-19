'use client';

import { useId, useState } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { isVersionPaths } from 'lib/guards';
import { Select } from '@/components/ui';

/** The value of the latest docs in the select. */
const latest = 'latest';

/** Splits a docs path into its version (or latest) and the page within it. */
function locate(pathname: string, versions: string[]): { version: string; page: string } {
  const rest = pathname.replace(/^\/docs/, '');
  const [, first = '', ...more] = rest.split('/');
  if (versions.includes(first)) {
    return { version: first, page: more.length > 0 ? '/' + more.join('/') : '' };
  }
  return { version: latest, page: rest };
}

/** The docs path of page in version. */
function pathOf(version: string, page: string): string {
  return version === latest ? `/docs${page}` : `/docs/${version}${page}`;
}

/**
 * Switches the documentation between the latest version and the snapshots of
 * earlier releases, staying on the same page when the other version has it
 * and going to its first page when it does not.
 */
export function VersionSelect({ versions }: { versions: string[] }) {
  const id = useId();
  const router = useRouter();
  const pathname = usePathname();
  const current = locate(pathname, versions);
  const [busy, setBusy] = useState(false);

  async function change(version: string) {
    setBusy(true);
    let target = pathOf(version, current.page);
    try {
      const response = await fetch('/api/docs-versions');
      const data: unknown = await response.json();
      const paths = isVersionPaths(data) ? data[version] : undefined;
      if (isVersionPaths(data) && !(Array.isArray(paths) && paths.includes(current.page))) {
        target = pathOf(version, '');
      }
    } catch {
      // Without the list, try the same page; a missing one falls back to the
      // version's first page on the server.
    }
    router.push(target);
    setBusy(false);
  }

  return (
    <div className="flex flex-col gap-1">
      <label
        htmlFor={id}
        className="text-xs font-medium text-fd-muted-foreground"
      >
        Documentation version
      </label>
      <Select
        id={id}
        value={current.version}
        disabled={busy}
        onChange={(e) => void change(e.target.value)}
      >
        <option value={latest}>Latest</option>
        {versions.map((v) => (
          <option
            key={v}
            value={v}
          >
            {v}
          </option>
        ))}
      </Select>
    </div>
  );
}
