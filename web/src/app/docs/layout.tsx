import { DocsLayout } from 'fumadocs-ui/layouts/docs';
import { docs, docsVersions } from 'lib/source';
import { VersionSelect } from '@/components/shared/version-select';
import { ViewportGlow } from '@/components/ui';
import { baseOptions } from '@/config/layout';

export default function Layout({ children }: { children: React.ReactNode }) {
  const versions = docsVersions();
  return (
    <DocsLayout
      tree={docs.getPageTree()}
      // Each release's docs are a root folder of their own, picked with the
      // version select rather than with Fumadocs' tabs.
      tabs={false}
      sidebar={{ banner: versions.length > 0 ? <VersionSelect versions={versions} /> : undefined }}
      {...baseOptions()}
    >
      <ViewportGlow />
      {children}
    </DocsLayout>
  );
}
