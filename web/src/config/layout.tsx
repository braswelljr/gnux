import { type BaseLayoutProps } from 'fumadocs-ui/layouts/shared';
import { Logo } from '@/components/ui/logo';
import { project, siteConfig } from '@/config/site';

/** Navigation shared by the home layout and the docs layout. */
export function baseOptions(): BaseLayoutProps {
  return {
    nav: {
      title: (
        <span className="inline-flex items-center gap-2">
          <Logo className="size-6" />
          <span className="font-mono text-base font-semibold">{siteConfig.shortName}</span>
        </span>
      ),
      url: '/'
    },
    links: [
      { text: 'Docs', url: '/docs', active: 'nested-url' },
      { text: 'Playground', url: '/playground', active: 'nested-url' }
    ],
    githubUrl: project.github
  };
}
