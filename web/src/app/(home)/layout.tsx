import { HomeLayout } from 'fumadocs-ui/layouts/home';
import { SiteFooter } from '@/components/shared/site-footer';
import { baseOptions } from '@/config/layout';

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <HomeLayout {...baseOptions()}>
      {children}
      <SiteFooter />
    </HomeLayout>
  );
}
