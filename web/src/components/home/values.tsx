import { Boxes, Globe, Palette, Wrench } from 'lucide-react';
import { GridCell, Heading, Reveal, Section } from '@/components/ui';

const values = [
  {
    icon: Boxes,
    tag: 'Binary',
    title: 'One static binary',
    description:
      'Every tool in one file, built without cgo. Nothing to install alongside it, nothing that breaks when the system upgrades.'
  },
  {
    icon: Globe,
    tag: 'Portability',
    title: 'The same everywhere',
    description:
      'Linux, macOS, FreeBSD, Windows and the browser run the same code, so a script prints the same output and exits with the same status on each.'
  },
  {
    icon: Wrench,
    tag: 'Fixes',
    title: 'Fixes, not just copies',
    description:
      'GNU flags and messages, minus its rough edges: dux reads trees in parallel, recursive chmodx refuses /, nicex and nohupx really work on Windows. Every difference is documented.'
  },
  {
    icon: Palette,
    tag: 'Themes',
    title: 'Your colors, every tool',
    description:
      "More than fifty built-in themes, from Solarized and Xcode to iOS colors and monochrome. Pick one once with gnux theme and every tool uses it; pipes and scripts still get GNU's exact output."
  }
];

/** Why gnux rather than a port: four promises, each kept in the code. */
export function Values() {
  return (
    <Section
      aria-labelledby="values-title"
      index={3}
      label="Why gnux"
      bleed={
        <ul className="grid gap-px border-t border-line bg-line md:grid-cols-2 xl:grid-cols-4">
          {values.map(({ icon: Icon, tag, title, description }, i) => (
            <li
              key={title}
              className="min-w-0"
            >
              <GridCell
                label={tag}
                index={i + 1}
              >
                <Reveal
                  delay={i * 0.06}
                  className="flex flex-col gap-3 p-4 sm:p-6"
                >
                  <Icon
                    aria-hidden="true"
                    className="size-6 text-lit"
                  />
                  <Heading level={3}>{title}</Heading>
                  <p className="text-sm text-pretty text-fd-muted-foreground">{description}</p>
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
          id="values-title"
        >
          Compatible where it counts. Better where it can be.
        </Heading>
      </Reveal>
    </Section>
  );
}
