import { ImageResponse } from 'next/og';
import { OgCard } from '@/components/shared/og-card';
import { siteConfig } from '@/config/site';

export const alt = siteConfig.name;
export const size = { width: 1200, height: 630 };
export const contentType = 'image/png';

/** The card every page shares unless it brings its own, as docs pages do. */
export default function Image() {
  return new ImageResponse(
    <OgCard
      eyebrow={siteConfig.shortName}
      title="The GNU core utilities, everywhere"
      description="A GNU-compatible toolkit in Go: one binary, a library and a portable shell."
      site={new URL(siteConfig.url).host}
    />,
    size
  );
}
