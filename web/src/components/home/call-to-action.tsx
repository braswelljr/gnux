import Link from 'next/link';
import { ArrowRight, BookOpen } from 'lucide-react';
import {
  AnimatedBackground,
  buttonVariants,
  Frame,
  GradientText,
  Heading,
  Keycap,
  Reveal
} from '@/components/ui';
import { project } from '@/config/site';

/** The page's last word: try it now, or read up first. */
export function CallToAction() {
  return (
    <section
      aria-labelledby="cta-title"
      className="border-b border-line"
    >
      <Frame>
        <Reveal className="px-4 py-12 sm:px-6 sm:py-20">
          <div className="relative isolate overflow-hidden rounded-2xl plate px-6 py-14 text-center sm:px-12 sm:py-20">
            <AnimatedBackground variant="subtle" />
            {/* The name, spelled out on four keys; the x is the one that matters. */}
            <div
              aria-hidden="true"
              className="mb-8 flex justify-center gap-2 font-mono text-xl font-semibold"
            >
              {['g', 'n', 'u', 'x'].map((letter) => (
                <Keycap
                  key={letter}
                  tone={letter === 'x' ? 'accent' : 'alpha'}
                  className="grid size-12 place-items-center rounded-lg [--cap-depth:5px] sm:size-14"
                >
                  {letter}
                </Keycap>
              ))}
            </div>
            <Heading
              level={2}
              id="cta-title"
              size="3xl"
            >
              Run it <GradientText>before you install it</GradientText>
            </Heading>
            <p className="mx-auto mt-4 max-w-xl text-pretty text-fd-muted-foreground">
              The playground is the real gnux compiled to WebAssembly, on a filesystem held in
              memory. Every tool and the shell, and nothing leaves your machine.
            </p>
            <div className="mt-8 flex flex-wrap justify-center gap-3">
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
            </div>
            <nav
              aria-label="Project"
              className="mt-8 flex flex-wrap justify-center gap-x-6 gap-y-2 text-sm text-fd-muted-foreground"
            >
              <a
                href={project.github}
                rel="noreferrer"
                className="rounded-sm underline-offset-4 outline-none hover:text-fd-foreground hover:underline focus-visible:ring-2 focus-visible:ring-fd-ring"
              >
                Source on GitHub
              </a>
              <a
                href={project.goReference}
                rel="noreferrer"
                className="rounded-sm underline-offset-4 outline-none hover:text-fd-foreground hover:underline focus-visible:ring-2 focus-visible:ring-fd-ring"
              >
                Go API reference
              </a>
              <a
                href={project.releases}
                rel="noreferrer"
                className="rounded-sm underline-offset-4 outline-none hover:text-fd-foreground hover:underline focus-visible:ring-2 focus-visible:ring-fd-ring"
              >
                Releases
              </a>
            </nav>
          </div>
        </Reveal>
      </Frame>
    </section>
  );
}
