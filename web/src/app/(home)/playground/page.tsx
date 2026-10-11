import { Suspense } from 'react';
import { getTools } from 'lib/source';
import { Playground } from '@/components/playground/playground';
import { AnimatedBackground, Container, GradientText, Heading } from '@/components/ui';
import { siteConfig } from '@/config/site';
import { createMetadata } from '@/utils/metadata';

export const metadata = createMetadata(siteConfig, {
  title: 'Playground',
  description:
    'Run the real gnux in your browser: every tool and the gnux shell, compiled to WebAssembly, on an in-memory filesystem. Nothing touches your disk.',
  keywords: ['playground', 'online terminal', 'try coreutils online'],
  canonical: '/playground'
});

export default function PlaygroundPage() {
  return (
    <div className="relative isolate flex flex-1 flex-col">
      <AnimatedBackground variant="subtle" />
      <Container className="flex flex-1 flex-col gap-6 py-8">
        <header className="flex flex-col gap-2">
          <Heading
            level={1}
            size="2xl"
          >
            <GradientText>Playground</GradientText>
          </Heading>
          <p className="max-w-3xl text-pretty text-fd-muted-foreground">
            This is the real gnux, compiled to WebAssembly. Everything runs locally in your browser
            on a virtual filesystem held in memory: nothing touches your disk, nothing is uploaded,
            and external programs are unavailable.
          </p>
        </header>
        {/* The playground reads `?cmd=` from the URL, which needs a boundary to prerender. */}
        <Suspense>
          <Playground
            toolHelp={getTools().map(({ name, description }) => ({ name, description }))}
          />
        </Suspense>
      </Container>
    </div>
  );
}
