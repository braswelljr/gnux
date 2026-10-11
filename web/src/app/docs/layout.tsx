import { DocsLayout } from 'fumadocs-ui/layouts/docs';
import { docs } from 'lib/source';
import { baseOptions } from '@/config/layout';

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <DocsLayout
      tree={docs.getPageTree()}
      {...baseOptions()}
    >
      {children}
    </DocsLayout>
  );
}
