import { type ReactNode } from 'react';
import Link from 'next/link';
import { ArrowRight } from 'lucide-react';
import { Heading, Reveal, Section, TypingCode, TypingTerminal } from '@/components/ui';
import { commandsSession, libraryDemo, shellSession } from '@/config/demos';

type Way = {
  /** What the reader types to reach it, set as code above the title. */
  entry: string;
  title: string;
  description: string;
  link: { href: string; text: string };
  demo: ReactNode;
};

const ways: Way[] = [
  {
    entry: 'rmx, cpx, lsx',
    title: 'As commands',
    description:
      'Each tool is a command of its own and a subcommand of the one binary. Both forms answer to the GNU name too, so rm and gnux rm reach the same engine.',
    link: { href: '/docs/guide', text: 'Installation and usage guide' },
    demo: (
      <TypingTerminal
        scenes={commandsSession}
        label="An animation of gnux tools at a prompt: cpx copies a directory verbosely, mvx renames a file, and headx prints its first two lines."
        title="zsh"
      />
    )
  },
  {
    entry: 'import "github.com/braswelljr/gnux/core/rm"',
    title: 'As a Go library',
    description:
      'Every tool is an importable engine. Pass the options its flags would set and the streams to use, and get an error back with the same diagnostics the command prints.',
    link: { href: '/docs/library', text: 'Using gnux from Go' },
    demo: (
      <TypingCode
        file={libraryDemo.file}
        code={libraryDemo.code}
        run={libraryDemo.run}
        label="An animation of a Go program that builds the rmx engine with recursive and verbose options and runs it on a build directory, then the output of go run."
      />
    )
  },
  {
    entry: 'gnux shell',
    title: 'As a portable shell',
    description:
      'A small POSIX sh subset whose commands are the tools, run in-process. Variables, pipes, redirection and && behave the same on every platform, Windows included.',
    link: { href: '/docs/shell', text: 'The gnux shell language' },
    demo: (
      <TypingTerminal
        scenes={shellSession}
        prompt="gnux>"
        label="An animation of the gnux shell: setting a variable, removing and recreating a directory tree with rmx and mkdirx, and echoing a message that uses the variable."
        title="gnux shell"
      />
    )
  }
];

/** The three ways to use gnux, each beside a short session that shows it. */
export function WaysToUse() {
  return (
    <Section aria-labelledby="ways-title">
      <Reveal className="max-w-2xl">
        <p className="font-mono text-sm text-lit">One engine per tool</p>
        <Heading
          level={2}
          id="ways-title"
          className="mt-2"
        >
          Three ways to reach it
        </Heading>
        <p className="mt-3 text-pretty text-fd-muted-foreground">
          The same engines sit behind all three, so a flag means the same thing wherever you reach
          it from.
        </p>
      </Reveal>
      <ul className="mt-12 flex flex-col gap-16 sm:gap-20">
        {ways.map((way, i) => (
          <li
            key={way.title}
            className="grid min-w-0 items-center gap-8 lg:grid-cols-2 lg:gap-12"
          >
            <Reveal className={i % 2 === 1 ? 'lg:order-2' : undefined}>
              <p className="truncate font-mono text-xs text-fd-muted-foreground">{way.entry}</p>
              <Heading
                level={3}
                size="xl"
                className="mt-2"
              >
                {way.title}
              </Heading>
              <p className="mt-3 max-w-lg text-pretty text-fd-muted-foreground">
                {way.description}
              </p>
              <Link
                href={way.link.href}
                className="mt-4 inline-flex items-center gap-1 rounded-sm text-sm font-medium underline-offset-4 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-fd-ring"
              >
                {way.link.text}
                <ArrowRight
                  aria-hidden="true"
                  className="size-4"
                />
              </Link>
            </Reveal>
            <Reveal
              delay={0.08}
              className="min-w-0"
            >
              {way.demo}
            </Reveal>
          </li>
        ))}
      </ul>
    </Section>
  );
}
