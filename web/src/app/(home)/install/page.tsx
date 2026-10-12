import Link from 'next/link';
import { ArrowRight, BookOpen, SquareTerminal } from 'lucide-react';
import { getTools } from 'lib/source';
import {
  AnimatedBackground,
  Badge,
  buttonVariants,
  CodeBlock,
  Container,
  GradientText,
  Heading,
  Section
} from '@/components/ui';
import { installPlatforms } from '@/config/install';
import { siteConfig } from '@/config/site';
import { createMetadata } from '@/utils/metadata';

export const metadata = createMetadata(siteConfig, {
  title: 'Install',
  description:
    'Install gnux on macOS, Windows, Linux, FreeBSD, with Go, in GitHub Actions or in Docker: one static binary, then a name per tool.',
  keywords: ['install', 'download', 'homebrew', 'winget', 'scoop', 'github action'],
  canonical: '/install'
});

/** Every way onto every platform, each as a few numbered, copyable steps. */
export default function InstallPage() {
  const toolCount = getTools().length;

  return (
    <div className="relative isolate">
      <AnimatedBackground variant="subtle" />
      <Container className="flex flex-col gap-6 pt-12 pb-4 sm:pt-16">
        <header className="flex max-w-3xl flex-col gap-4">
          <Heading
            level={1}
            size="3xl"
          >
            Install <GradientText>gnux</GradientText>.
          </Heading>
          <p className="text-lg text-pretty text-fd-muted-foreground">
            One static binary carries all {toolCount} tools, with no runtime and nothing else to
            install. Put it on your machine, link the tool names to it, and each one is a command of
            its own.
          </p>
        </header>
        <nav aria-label="Platforms">
          <ul className="flex flex-wrap gap-2">
            {installPlatforms.map((platform) => (
              <li key={platform.id}>
                <a
                  href={`#${platform.id}`}
                  className="rounded-lg outline-none focus-visible:ring-2 focus-visible:ring-fd-ring"
                >
                  <Badge>{platform.name}</Badge>
                </a>
              </li>
            ))}
          </ul>
        </nav>
      </Container>

      {installPlatforms.map((platform) => (
        <Section
          key={platform.id}
          id={platform.id}
          aria-labelledby={`${platform.id}-title`}
          className="scroll-mt-20 py-8 sm:py-10"
        >
          <div className="grid gap-6 lg:grid-cols-[16rem_minmax(0,1fr)]">
            <div className="flex flex-col gap-2">
              <Heading
                level={2}
                id={`${platform.id}-title`}
              >
                {platform.name}
              </Heading>
              <p className="text-sm text-pretty text-fd-muted-foreground">{platform.summary}</p>
            </div>
            <div className="flex min-w-0 flex-col gap-4">
              <ol className="flex flex-col gap-4">
                {platform.steps.map((step, i) => (
                  <li
                    key={step.title}
                    className="grid min-w-0 grid-cols-[2rem_minmax(0,1fr)] gap-3"
                  >
                    <span
                      aria-hidden="true"
                      className="keycap grid size-8 place-items-center rounded-lg font-mono text-sm font-semibold keycap-mod [--cap-depth:3px]"
                    >
                      {i + 1}
                    </span>
                    <div className="flex min-w-0 flex-col gap-2">
                      <Heading level={3}>
                        <span className="sr-only">Step {i + 1}: </span>
                        {step.title}
                      </Heading>
                      <p className="text-sm text-pretty text-fd-muted-foreground">{step.detail}</p>
                      <CodeBlock
                        title={step.shell}
                        code={step.code}
                        className="bg-fd-card"
                      />
                    </div>
                  </li>
                ))}
              </ol>
              {platform.note && (
                <p className="rounded-lg border bg-fd-card/60 p-3 text-sm text-pretty text-fd-muted-foreground">
                  {platform.note}
                </p>
              )}
            </div>
          </div>
        </Section>
      ))}

      <Section aria-labelledby="next-title">
        <div className="flex flex-col gap-4 rounded-2xl plate p-6 sm:p-8">
          <Heading
            level={2}
            id="next-title"
          >
            Installed. Now what?
          </Heading>
          <p className="max-w-2xl text-pretty text-fd-muted-foreground">
            The getting started guide walks through your first commands, the GNU-name aliases and
            shell completion. Or try any tool in the browser first.
          </p>
          <div className="flex flex-wrap gap-3">
            <Link
              href="/docs/guides/getting-started"
              className={buttonVariants({ variant: 'primary' })}
            >
              <BookOpen aria-hidden="true" />
              Getting started
            </Link>
            <Link
              href="/playground"
              className={buttonVariants({ variant: 'secondary' })}
            >
              <SquareTerminal aria-hidden="true" />
              Open the playground
              <ArrowRight aria-hidden="true" />
            </Link>
          </div>
        </div>
      </Section>
    </div>
  );
}
