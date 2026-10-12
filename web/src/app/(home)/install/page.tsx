import Link from 'next/link';
import { ArrowRight, BookOpen, SquareTerminal } from 'lucide-react';
import { getTools } from 'lib/source';
import {
  AnimatedBackground,
  Badge,
  buttonVariants,
  CodeBlock,
  Frame,
  GradientText,
  GridCell,
  Heading,
  pad2,
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
      <section aria-labelledby="install-title">
        <Frame>
          <div className="flex flex-col gap-6 px-4 pt-12 pb-10 sm:px-6 sm:pt-16">
            <header className="flex max-w-3xl flex-col gap-4">
              <Heading
                level={1}
                size="3xl"
                id="install-title"
              >
                Install <GradientText>gnux</GradientText>.
              </Heading>
              <p className="text-lg text-pretty text-fd-muted-foreground">
                One static binary carries all {toolCount} tools, with no runtime and nothing else to
                install. Put it on your machine, link the tool names to it, and each one is a
                command of its own.
              </p>
            </header>
            <nav aria-label="Platforms">
              <ul className="flex flex-wrap gap-2">
                {installPlatforms.map((platform, i) => (
                  <li key={platform.id}>
                    <a
                      href={`#${platform.id}`}
                      className="rounded-lg outline-none focus-visible:ring-2 focus-visible:ring-fd-ring"
                    >
                      <Badge>
                        <span className="text-faint">{pad2(i + 1)}</span> {platform.name}
                      </Badge>
                    </a>
                  </li>
                ))}
              </ul>
            </nav>
          </div>
        </Frame>
      </section>

      {installPlatforms.map((platform, p) => (
        <Section
          key={platform.id}
          id={platform.id}
          aria-labelledby={`${platform.id}-title`}
          index={p + 1}
          label={platform.name}
          aside={`${platform.steps.length} step${platform.steps.length === 1 ? '' : 's'}`}
          className="scroll-mt-14"
          bleed={
            <>
              <ol className="grid gap-px border-t border-line bg-line">
                {platform.steps.map((step, i) => (
                  <li
                    key={step.title}
                    className="min-w-0"
                  >
                    <GridCell
                      label={<span className="normal-case">{step.shell}</span>}
                      index={i + 1}
                    >
                      <div className="flex min-w-0 flex-col gap-2 p-4 sm:p-6">
                        <Heading level={3}>
                          <span className="sr-only">Step {i + 1}: </span>
                          {step.title}
                        </Heading>
                        <p className="text-sm text-pretty text-fd-muted-foreground">
                          {step.detail}
                        </p>
                        <CodeBlock
                          title={step.shell}
                          code={step.code}
                          className="mt-2 bg-fd-card"
                        />
                      </div>
                    </GridCell>
                  </li>
                ))}
              </ol>
              {platform.note && (
                <p className="border-t border-line bg-fd-background/70 px-4 py-4 text-sm text-pretty text-fd-muted-foreground sm:px-6">
                  {platform.note}
                </p>
              )}
            </>
          }
        >
          <Heading
            level={2}
            id={`${platform.id}-title`}
            className="max-w-3xl"
          >
            <span className="sr-only">{platform.name}: </span>
            {platform.summary}
          </Heading>
        </Section>
      ))}

      <Section
        aria-labelledby="next-title"
        index={installPlatforms.length + 1}
        label="Next"
        className="border-b border-line"
      >
        <div className="flex flex-col gap-4">
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
