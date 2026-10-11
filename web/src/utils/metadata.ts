import { type Metadata } from 'next';

/** What the site says about itself. One object, in `src/config/site.ts`. */
export type SiteConfig = {
  /** The production origin, without a trailing slash. */
  url: string;
  /** The full title of the site, used where no page title applies. */
  name: string;
  shortName: string;
  description: string;
  keywords: string[];
  locale: string;
  author: { name: string; url: string };
  twitterCard: 'summary' | 'summary_large_image';
  twitterHandle?: string;
  /** True keeps the whole site out of search results. */
  noIndex: boolean;
  /**
   * The default share image, for a page that names none. A page's own
   * `openGraph` replaces the root's, so the image has to be restated on every
   * page rather than inherited from the `opengraph-image` file convention.
   */
  ogImage: string;
  manifest: string;
};

type OgImage = string | { url: string; width?: number; height?: number; alt?: string };

export type MetadataOptions = {
  /** A page title; the root template appends the site name. */
  title?: string;
  description?: string;
  /** Added to the site's keywords. */
  keywords?: string[];
  /** The page's path (`/docs/guide`). Resolved against the site URL. */
  canonical?: string;
  noIndex?: boolean;
  ogImage?: OgImage;
  ogType?: 'website' | 'article';
};

/** An absolute URL on the site, for structured data and the sitemap. */
export function absoluteUrl(site: SiteConfig, path = '/') {
  return `${site.url}${path === '/' ? '' : path}`;
}

/**
 * Builds the metadata of a page. Every page and layout goes through this, and
 * none writes a `Metadata` literal, because the canonical URL, the Open Graph
 * tags and the Twitter card are the only things a link preview reads and they
 * are exactly what a hand-written literal forgets.
 *
 * Next.js replaces `openGraph` and `twitter` wholesale when a page sets them,
 * so each call emits both in full.
 *
 * Called with no options it returns the site-wide defaults for the root layout.
 */
export function createMetadata(site: SiteConfig, options: MetadataOptions = {}): Metadata {
  const title = options.title ?? site.name;
  const description = options.description ?? site.description;
  const canonical = options.canonical ?? '/';
  const index = !(options.noIndex ?? site.noIndex);
  const image = options.ogImage ?? site.ogImage;
  const images = [
    typeof image === 'string' ? { url: image, width: 1200, height: 630, alt: title } : image
  ];

  return {
    metadataBase: new URL(site.url),
    title:
      options.title === undefined
        ? { default: site.name, template: `%s | ${site.shortName}` }
        : options.title,
    description,
    applicationName: site.shortName,
    keywords: [...site.keywords, ...(options.keywords === undefined ? [] : options.keywords)],
    authors: [site.author],
    creator: site.author.name,
    category: 'technology',
    manifest: site.manifest,
    alternates: { canonical },
    robots: {
      index,
      follow: index,
      googleBot: {
        index,
        follow: index,
        'max-image-preview': 'large',
        'max-snippet': -1,
        'max-video-preview': -1
      }
    },
    openGraph: {
      type: options.ogType ?? 'website',
      siteName: site.shortName,
      title,
      description,
      url: canonical,
      locale: site.locale,
      images
    },
    twitter: {
      card: site.twitterCard,
      title,
      description,
      ...(site.twitterHandle && { creator: site.twitterHandle }),
      images: images.map((img) => img.url)
    }
  };
}
