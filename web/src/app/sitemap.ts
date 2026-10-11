import { type MetadataRoute } from 'next';
import { docs } from 'lib/source';
import { siteConfig } from '@/config/site';

const STATIC_ROUTES = ['', '/playground'];

export default function sitemap(): MetadataRoute.Sitemap {
  // Every page of the docs source, so a newly generated page is listed
  // without anything here changing.
  const pages = docs.getPages().map((page) => ({
    url: `${siteConfig.url}${page.url}`,
    changeFrequency: 'weekly' as const,
    priority: page.url === '/docs' ? 0.9 : 0.7
  }));

  return [
    ...STATIC_ROUTES.map((route) => ({
      url: `${siteConfig.url}${route}`,
      changeFrequency: 'weekly' as const,
      priority: route ? 0.8 : 1
    })),
    ...pages
  ];
}
