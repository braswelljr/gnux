import { GridCell, Heading, Reveal, Section, TypingTerminal } from '@/components/ui';
import { everywhereSession, platforms } from '@/config/demos';

/** One session typed on three platforms, printing the same thing on each. */
export function Everywhere() {
  return (
    <Section
      aria-labelledby="everywhere-title"
      index={2}
      label="Every OS"
      aside="No branch per platform"
      bleed={
        <ul className="grid gap-px border-t border-line bg-line lg:grid-cols-3">
          {platforms.map((platform, i) => (
            <li
              key={platform.name}
              className="min-w-0"
            >
              <GridCell
                label={platform.name}
                index={i + 1}
              >
                <Reveal
                  delay={i * 0.06}
                  className="p-4 sm:p-6"
                >
                  <TypingTerminal
                    scenes={everywhereSession}
                    prompt={platform.prompt}
                    title={`${platform.shell} on ${platform.name}`}
                    label={`An animation of ${platform.shell} on ${platform.name} running catx piped into headx, then the gnux shell creating a directory tree with mkdirx, with the same output as the other platforms.`}
                    screenClassName="h-44 sm:h-52"
                  />
                </Reveal>
              </GridCell>
            </li>
          ))}
        </ul>
      }
    >
      <Reveal className="max-w-2xl">
        <Heading
          level={2}
          id="everywhere-title"
        >
          Same script, every OS
        </Heading>
        <p className="mt-3 text-pretty text-fd-muted-foreground">
          Linux, macOS and Windows are built and tested together. The command is the same, the
          output is the same, and so is the exit status.
        </p>
      </Reveal>
    </Section>
  );
}
