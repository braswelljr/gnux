import { HomeLayout } from 'fumadocs-ui/layouts/home';
import { SiteFooter } from '@/components/shared/site-footer';
import { ViewportGlow } from '@/components/ui';
import { baseOptions } from '@/config/layout';

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    // The header's content sits in the page's column (max-w-6xl, 72rem), between the rails.
    <HomeLayout
      {...baseOptions()}
      className="[--fd-layout-width:72rem]"
    >
      <ViewportGlow />
      {children}
      <SiteFooter />
    </HomeLayout>
  );
}
