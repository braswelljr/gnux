import Link from 'next/link';
import { HomeLayout } from 'fumadocs-ui/layouts/home';
import { SiteFooter } from '@/components/shared/site-footer';
import {
  AnimatedBackground,
  buttonVariants,
  Container,
  GradientText,
  Heading,
  TypingTerminal
} from '@/components/ui';
import { notFoundSession } from '@/config/demos';
import { baseOptions } from '@/config/layout';
import { siteConfig } from '@/config/site';
import { createMetadata } from '@/utils/metadata';

export const metadata = createMetadata(siteConfig, {
  title: 'Page not found',
  description: 'The page you are looking for does not exist.',
  noIndex: true
});

export default function NotFound() {
  return (
    <HomeLayout {...baseOptions()}>
      <div className="relative isolate flex flex-1 items-center py-16 sm:py-24">
        <AnimatedBackground variant="subtle" />
        <Container className="grid items-center gap-10 lg:grid-cols-2">
          <div className="flex flex-col items-start gap-4">
            <p className="font-mono text-sm text-lit">exit status 2</p>
            <Heading
              level={1}
              size="display"
            >
              <GradientText>404</GradientText>
            </Heading>
            <Heading
              level={2}
              size="xl"
            >
              Page not found
            </Heading>
            <p className="max-w-md text-pretty text-fd-muted-foreground">
              There is nothing at this address. The documentation is a good place to start again.
            </p>
            <nav
              aria-label="Ways back"
              className="flex flex-wrap gap-3"
            >
              <Link
                href="/docs"
                className={buttonVariants({ variant: 'primary' })}
              >
                Go to the docs
              </Link>
              <Link
                href="/"
                className={buttonVariants({ variant: 'secondary' })}
              >
                Home
              </Link>
            </nav>
          </div>
          <TypingTerminal
            scenes={notFoundSession}
            loop={false}
            glow
            title="you@gnux: ~"
            label="An animation of lsx looking for this page and reporting that it does not exist, then echox printing the exit status 2."
            screenClassName="h-40"
          />
        </Container>
      </div>
      <SiteFooter />
    </HomeLayout>
  );
}
