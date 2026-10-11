import { notFound } from 'next/navigation';
import { ImageResponse } from 'next/og';
import { docs, getPageImage } from 'lib/source';
import { OgCard } from '@/components/shared/og-card';
import { siteConfig } from '@/config/site';

export const revalidate = false;

type Props = { params: Promise<{ slug: string[] }> };

export async function GET(_req: Request, { params }: Props) {
  const { slug } = await params;
  const page = docs.getPage(slug.slice(0, -1));
  if (!page) notFound();

  return new ImageResponse(
    <OgCard
      eyebrow={`${siteConfig.shortName} docs`}
      title={page.data.title}
      description={page.data.description}
      site={new URL(siteConfig.url).host}
    />,
    {
      width: 1200,
      height: 630
    }
  );
}

export function generateStaticParams() {
  return docs.getPages().map((page) => ({
    slug: getPageImage(page).segments
  }));
}
