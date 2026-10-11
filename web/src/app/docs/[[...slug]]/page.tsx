import { type Metadata } from 'next';
import { notFound } from 'next/navigation';
import { DocsBody, DocsDescription, DocsPage, DocsTitle } from 'fumadocs-ui/layouts/docs/page';
import { createRelativeLink } from 'fumadocs-ui/mdx';
import { docs, getPageImage } from 'lib/source';
import { getMDXComponents } from '@/components/mdx';
import { JsonLd } from '@/components/shared/json-ld';
import { siteConfig } from '@/config/site';
import { absoluteUrl, createMetadata } from '@/utils/metadata';

type Props = { params: Promise<{ slug?: string[] }> };

export default async function Page(props: Props) {
  const params = await props.params;
  const page = docs.getPage(params.slug);
  if (!page) notFound();

  const MDX = page.data.body;
  const url = absoluteUrl(siteConfig, page.url);

  // Home > Docs > (each ancestor that is itself a page) > this page. A folder
  // with no index page has no address, so it is left out rather than linked.
  const crumbs = [
    { name: siteConfig.shortName, url: absoluteUrl(siteConfig, '/') },
    { name: 'Docs', url: absoluteUrl(siteConfig, '/docs') },
    ...page.slugs.flatMap((_, i) => {
      const ancestor = docs.getPage(page.slugs.slice(0, i + 1));
      return ancestor
        ? [{ name: ancestor.data.title, url: absoluteUrl(siteConfig, ancestor.url) }]
        : [];
    })
  ];

  return (
    <DocsPage
      toc={page.data.toc}
      full={page.data.full}
      // The "On this page" outline as one thread that bends with the heading
      // levels, in the sidebar and in the popover on small screens alike.
      tableOfContent={{ style: 'clerk' }}
      tableOfContentPopover={{ style: 'clerk' }}
      // The previous and next cards keep one height whatever the page's
      // description: it is cut at two lines (the card's only direct paragraph).
      footer={{ className: '[&_a>p]:line-clamp-2' }}
    >
      <JsonLd
        data={{
          '@context': 'https://schema.org',
          '@type': 'BreadcrumbList',
          itemListElement: crumbs.map((crumb, i) => ({
            '@type': 'ListItem',
            position: i + 1,
            name: crumb.name,
            item: crumb.url
          }))
        }}
      />
      <JsonLd
        data={{
          '@context': 'https://schema.org',
          '@type': 'TechArticle',
          headline: page.data.title,
          description: page.data.description,
          url,
          mainEntityOfPage: url,
          inLanguage: 'en',
          image: absoluteUrl(siteConfig, getPageImage(page).url),
          author: { '@type': 'Person', name: siteConfig.author.name, url: siteConfig.author.url },
          isPartOf: {
            '@type': 'WebSite',
            name: siteConfig.shortName,
            url: absoluteUrl(siteConfig, '/')
          },
          about: { '@type': 'SoftwareApplication', name: siteConfig.shortName }
        }}
      />
      <DocsTitle>{page.data.title}</DocsTitle>
      <DocsDescription>{page.data.description}</DocsDescription>
      <DocsBody>
        <MDX
          components={getMDXComponents({
            // Lets generated pages link to each other by relative file path.
            a: createRelativeLink(docs, page)
          })}
        />
      </DocsBody>
    </DocsPage>
  );
}

export function generateStaticParams() {
  return docs.generateParams();
}

export async function generateMetadata(props: Props): Promise<Metadata> {
  const params = await props.params;
  const page = docs.getPage(params.slug);
  if (!page) return createMetadata(siteConfig, { title: 'Page not found', noIndex: true });

  return createMetadata(siteConfig, {
    title: page.data.title,
    description: page.data.description,
    canonical: page.url,
    ogType: 'article',
    ogImage: { url: getPageImage(page).url, width: 1200, height: 630, alt: page.data.title }
  });
}
