import { Heading, Reveal, Section, TypingTerminal } from '@/components/ui';
import { everywhereSession, platforms } from '@/config/demos';

/** One session typed on three platforms, printing the same thing on each. */
export function Everywhere() {
  return (
    <Section
      aria-labelledby="everywhere-title"
      className="relative isolate"
    >
      <Reveal className="mx-auto max-w-2xl text-center">
        <p className="font-mono text-sm text-lit">No branch per platform</p>
        <Heading
          level={2}
          id="everywhere-title"
          className="mt-2"
        >
          Same script, every OS
        </Heading>
        <p className="mt-3 text-pretty text-fd-muted-foreground">
          Linux, macOS and Windows are built and tested together. The command is the same, the
          output is the same, and so is the exit status.
        </p>
      </Reveal>
      <ul className="mt-10 grid gap-4 lg:grid-cols-3">
        {platforms.map((platform, i) => (
          <li
            key={platform.name}
            className="min-w-0"
          >
            <Reveal delay={i * 0.06}>
              <TypingTerminal
                scenes={everywhereSession}
                prompt={platform.prompt}
                title={`${platform.shell} on ${platform.name}`}
                label={`An animation of ${platform.shell} on ${platform.name} running catx piped into headx, then the gnux shell creating a directory tree with mkdirx, with the same output as the other platforms.`}
                screenClassName="h-52"
              />
            </Reveal>
          </li>
        ))}
      </ul>
    </Section>
  );
}
