import Link from 'next/link';
import { ArrowRight, BookOpen } from 'lucide-react';
import { HeroStage } from '@/components/home/hero-stage';
import {
  AnimatedBackground,
  buttonVariants,
  Container,
  GradientText,
  Heading,
  Kbd,
  Logo,
  Reveal
} from '@/components/ui';

/** The top of the landing page: what gnux is, where to go next, and gnux at work. */
export function Hero({ toolCount }: { toolCount: number }) {
  return (
    <section
      aria-labelledby="hero-title"
      className="relative isolate overflow-hidden pt-16 pb-12 sm:pt-24 sm:pb-16"
    >
      <AnimatedBackground />
      <Container className="flex flex-col items-center gap-6 text-center">
        <Reveal className="flex flex-col items-center gap-6">
          <Logo className="size-14 text-fd-foreground sm:size-16" />
          <p className="flex flex-wrap items-center justify-center gap-x-1.5 gap-y-2 font-mono text-xs text-fd-muted-foreground">
            <span>
              <Kbd>rm</Kbd> is <Kbd>rmx</Kbd>,
            </span>
            <span>
              <Kbd>ls</Kbd> is <Kbd>lsx</Kbd>:
            </span>
            <span>{toolCount} tools in one binary</span>
          </p>
        </Reveal>
        <Reveal delay={0.05}>
          <Heading
            level={1}
            size="display"
            id="hero-title"
            className="max-w-4xl"
          >
            The GNU core utilities, <GradientText>everywhere</GradientText>
          </Heading>
        </Reveal>
        <Reveal delay={0.1}>
          <p className="max-w-2xl text-lg text-pretty text-fd-muted-foreground">
            A GNU-compatible toolkit in Go: one binary, a library and a portable shell. The same
            flags, messages and exit statuses on Linux, macOS, Windows, in containers, in the
            browser and inside Go programs.
          </p>
        </Reveal>
        <Reveal
          delay={0.15}
          className="flex flex-wrap justify-center gap-3"
        >
          <Link
            href="/playground"
            className={buttonVariants({ variant: 'primary', size: 'lg' })}
          >
            Try it in your browser
            <ArrowRight aria-hidden="true" />
          </Link>
          <Link
            href="/docs"
            className={buttonVariants({ variant: 'secondary', size: 'lg' })}
          >
            <BookOpen aria-hidden="true" />
            Read the docs
          </Link>
        </Reveal>
        <Reveal
          delay={0.2}
          className="mt-8 w-full sm:mt-12"
        >
          <HeroStage />
        </Reveal>
      </Container>
    </section>
  );
}
