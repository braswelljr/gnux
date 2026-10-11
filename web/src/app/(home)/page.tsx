import { getTools } from 'lib/source';
import { CallToAction } from '@/components/home/call-to-action';
import { Everywhere } from '@/components/home/everywhere';
import { Hero } from '@/components/home/hero';
import { Install } from '@/components/home/install';
import { ToolGrid } from '@/components/home/tool-grid';
import { WaysToUse } from '@/components/home/ways-to-use';
import { JsonLd } from '@/components/shared/json-ld';
import { project, siteConfig } from '@/config/site';
import { absoluteUrl, createMetadata } from '@/utils/metadata';

export const metadata = createMetadata(siteConfig, {
  title: 'The GNU core utilities, everywhere',
  canonical: '/'
});

export default function HomePage() {
  const tools = getTools();

  return (
    <>
      <JsonLd
        data={{
          '@context': 'https://schema.org',
          '@type': 'SoftwareApplication',
          name: siteConfig.shortName,
          description: siteConfig.description,
          url: absoluteUrl(siteConfig),
          applicationCategory: 'DeveloperApplication',
          operatingSystem: 'Linux, macOS, Windows',
          programmingLanguage: 'Go',
          codeRepository: project.github,
          license: project.license,
          downloadUrl: project.github,
          softwareHelp: absoluteUrl(siteConfig, '/docs'),
          author: {
            '@type': 'Person',
            name: siteConfig.author.name,
            url: siteConfig.author.url
          },
          offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' }
        }}
      />
      <Hero toolCount={tools.length} />
      <WaysToUse />
      <Everywhere />
      <ToolGrid tools={tools} />
      <Install />
      <CallToAction />
    </>
  );
}
